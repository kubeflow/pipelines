# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Mutating reporting recovery probe for the disposable readiness cluster.

Capture suspended, persisted source runs before upgrading. The recover
phase withholds API-server Workflow GET, resumes those same Workflows,
observes terminal Kubernetes state without terminal API state, restores
the permission, and requires the existing persistence worker to catch up
without restart.
"""
import argparse
import copy
import http.client
import json
from pathlib import Path
import re
import subprocess
import sys
import time

from fixture_http import FixtureClient
from kfp_http import Client
from live_schedule_check import list_runs
from provision_live_schedules import api_identifier
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
from provision_live_schedules import verify_state
from provision_live_schedules import write_object

API_ROLE = 'fixture-ml-pipeline-infrastructure'

TERMINAL = {'SUCCEEDED', 'FAILED', 'ERROR', 'CANCELED', 'SKIPPED'}


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def kube(*args, value=None):
    result = subprocess.run(
        ['kubectl', '--context', CONTEXT, '--request-timeout=20s', *args],
        input=json.dumps(value) if value is not None else None,
        capture_output=True,
        text=True,
        timeout=30,
        check=False)
    require(result.returncode == 0, 'reporting_kubernetes_operation_failed')
    require(len(result.stdout) <= 4 * 1024 * 1024, 'reporting_collection_limit')
    return result.stdout


def get(resource, namespace=NAMESPACE):
    return json.loads(kube('-n', namespace, 'get', resource, '-o', 'json'))


def field(obj, snake, camel):
    return obj.get(snake, obj.get(camel))


def run(client, uid):
    obj = client.get('/apis/v2beta1/runs/' + uid)
    require(field(obj, 'run_id', 'runId') == uid, 'reporting_run_id_changed')
    return obj


def project(workflow, obj):
    meta = workflow['metadata']
    uid = field(obj, 'run_id', 'runId')
    require(
        meta.get('namespace') == NAMESPACE and
        meta.get('labels', {}).get('pipeline/runid') == uid and
        bool(meta.get('uid')), 'reporting_workflow_identity_mismatch')
    return dict(
        run_id=uid,
        workflow_name=meta['name'],
        workflow_uid=meta['uid'],
        experiment_id=field(obj, 'experiment_id', 'experimentId'),
        recurring_run_id=field(obj, 'recurring_run_id', 'recurringRunId'))


def capture(client, run_ids):
    require(1 <= len(run_ids) <= 10 and len(set(run_ids)) == len(run_ids),
            'reporting_invalid_source_run_count')
    workflows = get('workflows.argoproj.io')['items']
    records = []
    for uid in run_ids:
        obj = run(client, uid)
        matches = [
            w for w in workflows if w.get('metadata', {}).get('labels', {}).get(
                'pipeline/runid') == uid
        ]
        require(len(matches) == 1, 'reporting_source_workflow_missing')
        workflow = matches[0]
        require(
            workflow.get('spec', {}).get('suspend') is True and
            obj.get('state') not in TERMINAL,
            'reporting_source_not_suspended_and_active')
        records.append(project(workflow, obj))
    return dict(scope='source_reporting_recovery', runs=records)


class SourceClient(FixtureClient):
    """Extend the bounded fixture transport only for V1 run creation."""

    def post(self, path, body):
        if path != '/apis/v1beta1/runs':
            return super().post(path, body)
        connection = http.client.HTTPConnection(
            self.host, self.port, timeout=20)
        try:
            connection.request(
                'POST',
                path,
                body=json.dumps(body),
                headers={
                    'Authorization': 'Bearer ' + self.token,
                    'Content-Type': 'application/json',
                    'Accept-Encoding': 'identity'
                })
            response = connection.getresponse()
            require(200 <= response.status < 300,
                    'reporting_source_creation_rejected')
            deadline = time.monotonic() + 20
            chunks, size = [], 0
            while True:
                remaining = deadline - time.monotonic()
                require(remaining > 0, 'reporting_source_creation_timeout')
                raw = getattr(getattr(response, 'fp', None), 'raw', None)
                sock = getattr(raw, '_sock', None) or connection.sock
                if sock is not None:
                    sock.settimeout(remaining)
                chunk = response.read1(min(65536, 4 * 1024 * 1024 - size + 1))
                size += len(chunk)
                require(size <= 4 * 1024 * 1024, 'reporting_response_limit')
                if not chunk:
                    break
                chunks.append(chunk)
            return json.loads(b''.join(chunks))
        except (OSError, http.client.HTTPException):
            raise ValueError('reporting_source_creation_failed') from None
        finally:
            connection.close()


def reference(kind, uid):
    return dict(key=dict(type=kind, id=uid), relationship='OWNER')


def v1_workflow():
    # A real V1 workflow, with no V2 IR, launcher or package installation.
    return dict(
        apiVersion='argoproj.io/v1alpha1',
        kind='Workflow',
        metadata=dict(generateName='readiness-v1-'),
        spec=dict(
            entrypoint='hello',
            templates=[
                dict(
                    name='hello',
                    container=dict(
                        image='docker.io/alpine:3.23',
                        command=['sh', '-c'],
                        args=['echo readiness-v1']))
            ]))


def prepare_pair(client, read_client, state_dir):
    """Create real raw-Argo immediate and recurring runs on source 2.17."""
    path = state_dir / 'reporting-created.json'
    require(not path.exists(), 'reporting_preparation_already_started')
    state = dict(scope='source_reporting_preparation')
    write_object(path, state)
    experiment = client.post(
        '/apis/v1beta1/experiments',
        dict(
            name='reporting-recovery-' + state_dir.name,
            resource_references=[reference('NAMESPACE', NAMESPACE)]))
    state['experiment_id'] = api_identifier(experiment.get('id'))
    write_object(path, state)
    workflow = v1_workflow()
    workflow['spec']['suspend'] = True
    payload = dict(
        name='reporting-recovery-immediate',
        pipeline_spec=dict(workflow_manifest=json.dumps(workflow)),
        resource_references=[
            reference('EXPERIMENT', state['experiment_id']),
            reference('NAMESPACE', NAMESPACE)
        ])
    response = client.post('/apis/v1beta1/runs', payload)
    state['immediate_run_id'] = api_identifier(
        response.get('run', response).get('id'))
    write_object(path, state)
    payload.update(
        name='reporting-recovery-recurring',
        enabled=False,
        max_concurrency='1',
        no_catchup=True,
        trigger=dict(periodic_schedule=dict(interval_second='30')))
    response = client.post('/apis/v1beta1/jobs', payload)
    state['schedule_id'] = api_identifier(response.get('id'))
    write_object(path, state)
    prefix = '/apis/v1beta1/jobs/' + state['schedule_id']
    try:
        client.post(prefix + '/enable', {})

        def scheduled_run():
            records = list_runs(read_client, NAMESPACE, state['schedule_id'])
            require(len(records) <= 1, 'reporting_multiple_source_ticks')
            return records

        records = wait_for(scheduled_run, 300, phase='source_recurring_run')
        state['recurring_run_id'] = api_identifier(
            field(records[0], 'run_id', 'runId'))
        write_object(path, state)
    finally:
        client.post(prefix + '/disable', {})
    run_ids = [state['immediate_run_id'], state['recurring_run_id']]

    def captured():
        # Creation can precede the Workflow being visible to the controller.
        workflows = get('workflows.argoproj.io')['items']
        found = {
            w.get('metadata', {}).get('labels', {}).get('pipeline/runid')
            for w in workflows
        }
        return capture(read_client, run_ids) if set(run_ids) <= found else None

    return wait_for(captured, 120, phase='source_workflow_capture')


def ownership_evidence(client, records, state_dir=None):
    from ownership_diagnostics import collect
    findings, coverage = collect(client, [NAMESPACE])
    resources = {
        'runs/' + NAMESPACE + '/' + record['run_id'] for record in records
    }
    selected = [
        f for f in findings
        if f['resource'] in resources or f['resource'] == 'runs/' + NAMESPACE
    ]
    evidence = dict(
        scope='source_ownership_evidence', findings=selected, coverage=coverage)
    if state_dir is not None:
        write_object(state_dir / 'reporting-ownership.json', evidence)
    require('runs/' + NAMESPACE in coverage.get('completed_scopes', []),
            'reporting_source_ownership_collection_incomplete')
    for resource in resources:
        observed = {
            f['rule']
            for f in selected
            if f['resource'] == resource and f['status'] == 'observed'
        }
        if not {
                'ownership.stored_identity_present',
                'ownership.experiment_namespace_present'
        } <= observed:
            return None
    return dict(
        scope='source_ownership_evidence', findings=selected, coverage=coverage)


def prepare(client, read_client, state_dir):
    prepared = []
    for name in ('outage', 'deletion'):
        pair_dir = state_dir / name
        pair_dir.mkdir(parents=True, exist_ok=True)
        prepared.append(prepare_pair(client, read_client, pair_dir))
    result = dict(
        scope='source_reporting_recovery',
        runs=prepared[0]['runs'],
        deletion_runs=prepared[1]['runs'])
    write_object(state_dir / 'reporting-source.json', result)
    evidence = wait_for(
        lambda: ownership_evidence(read_client, result['runs'] + result[
            'deletion_runs'], state_dir),
        180,
        phase='source_stored_ownership')
    write_object(state_dir / 'reporting-ownership.json', evidence)
    return result


def restore(state_dir):
    path = state_dir / 'reporting-rbac-restore.json'
    if path.exists():
        saved = read_object(path)
        require(
            saved.get('kind') == 'Role' and
            saved.get('metadata', {}).get('name') == API_ROLE and
            saved.get('metadata', {}).get('namespace') == NAMESPACE,
            'reporting_invalid_restore_state')
        current = get('role/' + API_ROLE)
        if current['rules'] != saved['rules']:
            fault = without_workflow_get(saved['rules'])
            kube(
                '-n', NAMESPACE, 'patch', 'role/' + API_ROLE, '--type=json',
                '-p',
                json.dumps([
                    dict(op='test', path='/rules', value=fault),
                    dict(op='replace', path='/rules', value=saved['rules'])
                ]))


def deleted_evidence(client, proxy, records):
    response = proxy.get('/apis/v2beta1/reporting-fixture-evidence')
    require(
        response.get('namespace') == NAMESPACE,
        'reporting_proxy_namespace_mismatch')
    evidence = response.get('runs', [])
    require(isinstance(evidence, list), 'reporting_proxy_evidence_invalid')
    live = get('workflows.argoproj.io')['items']
    names = {w['metadata']['name'] for w in live}
    result = []
    for record in records:
        matches = [
            item for item in evidence if item.get('run_id') == record['run_id']
        ]
        require(len(matches) == 1, 'reporting_proxy_run_missing')
        item = matches[0]
        require(
            item.get('workflow_name') == record['workflow_name'] and
            item.get('workflow_uid') == record['workflow_uid'],
            'reporting_proxy_identity_mismatch')
        require(not item.get('deletion_failed'),
                'reporting_proxy_delete_failed')
        if not item.get('deleted') or not item.get('upstream_code'):
            return None
        # Final-state persistence can succeed before its label update discovers
        # NotFound. The API row, rather than RPC status alone, proves completion.
        require(item['upstream_code'] in ('OK', 'NotFound'),
                'reporting_captured_report_rejected')
        require(record['workflow_name'] not in names,
                'reporting_deleted_workflow_still_exists')
        obj = run(client, record['run_id'])
        require(
            field(obj, 'experiment_id',
                  'experimentId') == record['experiment_id'] and
            field(obj, 'recurring_run_id',
                  'recurringRunId') == record['recurring_run_id'],
            'reporting_deleted_run_ownership_changed')
        if obj.get('state') != 'SUCCEEDED':
            require(
                obj.get('state') not in TERMINAL,
                'reporting_deleted_run_failed')
            return None
        result.append(
            dict(
                record,
                state=obj['state'],
                upstream_code=item['upstream_code'],
                deleted=True))
    return result


def delete_captured(client, proxy, state):
    records = state['deletion_runs']
    original_worker = worker_pods()
    for record in records:
        workflow = get('workflow/' + record['workflow_name'])
        require(
            project(workflow, run(client, record['run_id'])) == record and
            workflow.get('spec', {}).get('suspend') is True,
            'reporting_deletion_source_not_held')
        kube(
            '-n', NAMESPACE, 'patch', 'workflow/' + record['workflow_name'],
            '--type=json', '-p',
            json.dumps([
                dict(
                    op='test',
                    path='/metadata/uid',
                    value=record['workflow_uid']),
                dict(op='replace', path='/spec/suspend', value=False)
            ]))
    result = wait_for(
        lambda: deleted_evidence(client, proxy, records),
        600,
        phase='captured_deletion')
    require(worker_pods() == original_worker,
            'reporting_worker_restarted_during_deletion')
    return dict(
        scope='worker_captured_terminal_deletion_race',
        outcome='passed',
        runs=result,
        worker_restarted=False,
        deletion_before_worker_snapshot_validated=False)


def without_workflow_get(rules):
    result = copy.deepcopy(rules)
    changed = False
    for rule in result:
        if ('argoproj.io' in rule.get('apiGroups', []) and
                'workflows' in rule.get('resources', []) and
                'get' in rule.get('verbs', [])):
            require(
                rule['apiGroups'] == ['argoproj.io'] and
                rule['resources'] == ['workflows'],
                'reporting_fault_requires_dedicated_workflow_rule')
            rule['verbs'].remove('get')
            changed = True
    require(changed, 'reporting_workflow_get_rule_missing')
    return result


def permitted(verb, account):
    # Impersonate the account including its normal service-account groups.
    output = kube(
        'create',
        '-f',
        '-',
        '-o',
        'json',
        value=dict(
            apiVersion='authorization.k8s.io/v1',
            kind='SubjectAccessReview',
            spec=dict(
                user='system:serviceaccount:kubeflow:' + account,
                groups=[
                    'system:serviceaccounts', 'system:serviceaccounts:kubeflow',
                    'system:authenticated'
                ],
                resourceAttributes=dict(
                    namespace=NAMESPACE,
                    group='argoproj.io',
                    resource='workflows',
                    verb=verb))))
    return json.loads(output).get('status', {}).get('allowed') is True


def worker_pods():
    items = json.loads(
        kube('-n', 'kubeflow', 'get', 'pods', '-l',
             'app=ml-pipeline-persistenceagent', '-o', 'json'))['items']
    require(bool(items), 'reporting_persistence_agent_missing')
    require(
        all(
            p.get('status', {}).get('phase') == 'Running' and all(
                c.get('ready')
                for c in p.get('status', {}).get('containerStatuses', [])) and
            p.get('status', {}).get('containerStatuses')
            for p in items), 'reporting_persistence_agent_not_ready')
    return sorted((p['metadata']['uid'],
                   tuple(
                       c.get('restartCount', 0)
                       for c in p['status']['containerStatuses']))
                  for p in items)


def observe(client, records, recovered=False):
    output = []
    for record in records:
        workflow = get('workflow/' + record['workflow_name'])
        obj = run(client, record['run_id'])
        require(
            project(workflow, obj) == record,
            'reporting_source_identity_changed')
        phase = workflow.get('status', {}).get('phase')
        if phase != 'Succeeded':
            require(
                phase not in ('Failed', 'Error'), 'reporting_workload_failed')
            return None
        if recovered:
            if obj.get('state') != 'SUCCEEDED':
                require(
                    obj.get('state') not in TERMINAL,
                    'reporting_wrong_terminal_state')
                return None
        else:
            require(
                obj.get('state') not in TERMINAL,
                'reporting_fault_did_not_block_persistence')
        output.append(
            dict(record, workflow_phase=phase, run_state=obj.get('state')))
    return output


def proxy_diagnostics():
    """Capture rollout reasons without container configuration or log
    bodies."""
    reasons = {
        'CrashLoopBackOff', 'ImagePullBackOff', 'ErrImagePull',
        'ErrImageNeverPull', 'CreateContainerConfigError',
        'CreateContainerError', 'RunContainerError', 'ContainerCreating',
        'PodInitializing', 'Completed', 'Error', 'OOMKilled'
    }
    pods = json.loads(
        kube('-n', 'kubeflow', 'get', 'pods', '-l', 'app=kfp-reporting-proxy',
             '-o', 'json')).get('items', [])
    require(
        isinstance(pods, list) and len(pods) <= 10, 'proxy_diagnostic_limit')
    projected = []
    pod_uids = {pod.get('metadata', {}).get('uid') for pod in pods} - {None, ''}
    for pod in pods:
        status = pod.get('status', {})
        phase = status.get('phase')
        containers = []
        for item in status.get('containerStatuses', []):
            if item.get('name') != 'kfp-reporting-proxy':
                continue
            states = {}
            for category in ('state', 'lastState'):
                for mode, value in item.get(category, {}).items():
                    if mode not in ('running', 'waiting', 'terminated'):
                        continue
                    reason = value.get('reason')
                    states[category] = dict(
                        mode=mode,
                        reason=reason if reason in reasons else 'other')
                    code = value.get('exitCode')
                    if type(code) is int and 0 <= code <= 255:
                        states[category]['exit_code'] = code
            containers.append(
                dict(
                    ready=item.get('ready') is True,
                    restart_count=item.get('restartCount') if type(
                        item.get('restartCount')) is int else None,
                    states=states))
        projected.append(
            dict(
                phase=phase if phase in ('Pending', 'Running', 'Succeeded',
                                         'Failed', 'Unknown') else 'unknown',
                scheduled=any(
                    c.get('type') == 'PodScheduled' and
                    c.get('status') == 'True'
                    for c in status.get('conditions', [])),
                ready=any(
                    c.get('type') == 'Ready' and c.get('status') == 'True'
                    for c in status.get('conditions', [])),
                containers=containers))
    events = []
    try:
        raw_events = get('events', 'kubeflow').get('items', [])
        require(
            isinstance(raw_events, list) and len(raw_events) <= 1000,
            'proxy_event_limit')
        for event in raw_events:
            if event.get('involvedObject', {}).get('uid') not in pod_uids:
                continue
            reason = event.get('reason')
            if reason not in ('Unhealthy', 'FailedScheduling', 'Failed',
                              'BackOff', 'Pulled', 'Created', 'Started'):
                reason = 'other'
            item = dict(reason=reason)
            message = event.get('message', '')
            if reason == 'Unhealthy' and isinstance(message, str):
                match = re.search(
                    r'HTTP probe failed with statuscode: ([1-5][0-9]{2})',
                    message)
                if match:
                    item['probe_http_status'] = int(match.group(1))
                elif 'connection refused' in message:
                    item['probe_connection'] = 'refused'
                elif 'timeout' in message.lower(
                ) or 'deadline exceeded' in message.lower():
                    item['probe_connection'] = 'timeout'
            if reason == 'FailedScheduling' and isinstance(message, str):
                item['insufficient_cpu'] = 'Insufficient cpu' in message
                item['insufficient_memory'] = 'Insufficient memory' in message
            events.append(item)
    except Exception:
        events = [dict(reason='event_collection_failed')]
    fixed_errors = ('fixture namespace or target limit invalid',
                    'fixture requires one to ten explicit targets',
                    'fixture target identity invalid or duplicated',
                    'fixture requires in-cluster credentials',
                    'fixture Kubernetes client unavailable',
                    'fixture upstream unavailable',
                    'fixture grpc listener unavailable',
                    'fixture listener stopped')
    observed = []
    for previous in (False, True):
        try:
            args = ('--previous',) if previous else ()
            logs = kube('-n', 'kubeflow', 'logs',
                        'deployment/kfp-reporting-proxy', '--tail=20', *args)
            observed.extend(
                error for error in fixed_errors if any(
                    line.endswith(error) for line in logs.splitlines()))
        except Exception:
            pass
    return dict(
        scope='proxy_rollout_diagnostics',
        pods=projected,
        events=events,
        fixed_startup_errors=sorted(set(observed)))


def timeout_diagnostics():
    """Only synthetic phase counts and controller watch namespace shapes."""
    result = dict(namespace=NAMESPACE, workflows={}, controllers={})
    items = get('workflows.argoproj.io').get('items', [])
    require(isinstance(items, list) and len(items) <= 1000, 'diagnostic_limit')
    for item in items:
        phase = item.get('status', {}).get('phase')
        phase = phase if phase in ('Pending', 'Running', 'Succeeded', 'Failed',
                                   'Error') else 'unknown'
        result['workflows'][phase] = result['workflows'].get(phase, 0) + 1
    for name in ('ml-pipeline-scheduledworkflow',
                 'ml-pipeline-persistenceagent'):
        obj = get('deployment/' + name, 'kubeflow')
        values = []
        for container in obj.get('spec',
                                 {}).get('template',
                                         {}).get('spec',
                                                 {}).get('containers', []):
            if container.get('name') != name:
                continue
            for env in container.get('env', []):
                if env.get('name') != 'NAMESPACE':
                    continue
                if env.get('valueFrom',
                           {}).get('fieldRef',
                                   {}).get('fieldPath') == 'metadata.namespace':
                    values.append('deployment_namespace')
                else:
                    value = env.get('value')
                    values.append(
                        value if value in (NAMESPACE, 'kubeflow') else 'other')
        result['controllers'][name] = values
    return result


def wait_for(check, seconds, phase='lookup_permission'):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        value = check()
        if value:
            return value
        time.sleep(5)
    report = dict(
        scope='reporting_timeout_diagnostics',
        phase=phase,
        timeout_seconds=seconds)
    try:
        report['observation'] = timeout_diagnostics()
    except Exception:
        report['observation'] = 'diagnostic_collection_failed'
    print(json.dumps(report, sort_keys=True), file=sys.stderr)
    raise ValueError('reporting_' + phase + '_deadline_exceeded')


def recover(client, state, state_dir):
    records = state['runs']
    role = get('role/' + API_ROLE)
    rules = role['rules']
    fault = without_workflow_get(rules)
    original_worker = worker_pods()
    require(permitted('get', 'ml-pipeline'), 'reporting_initial_lookup_denied')
    for verb in ('get', 'list', 'watch'):
        require(
            permitted(verb, 'ml-pipeline-persistenceagent'),
            'reporting_agent_workflow_access_missing')
    # Save restoration input before the mutation; CI cleanup can restore after
    # process termination. Never upload this file as acceptance evidence.
    write_object(state_dir / 'reporting-rbac-restore.json', role)
    try:
        kube(
            '-n', NAMESPACE, 'patch', 'role/' + API_ROLE, '--type=json', '-p',
            json.dumps([
                dict(op='test', path='/rules', value=rules),
                dict(op='replace', path='/rules', value=fault)
            ]))
        wait_for(lambda: not permitted('get', 'ml-pipeline'), 60)
        for verb in ('get', 'list', 'watch'):
            require(
                permitted(verb, 'ml-pipeline-persistenceagent'),
                'reporting_fault_affected_persistence_agent')
        for record in records:
            workflow = get('workflow/' + record['workflow_name'])
            require(
                project(workflow, run(client, record['run_id'])) == record and
                workflow.get('spec', {}).get('suspend') is True,
                'reporting_target_source_not_held')
            kube(
                '-n', NAMESPACE, 'patch', 'workflow/' + record['workflow_name'],
                '--type=json', '-p',
                json.dumps([
                    dict(
                        op='test',
                        path='/metadata/uid',
                        value=record['workflow_uid']),
                    dict(op='replace', path='/spec/suspend', value=False)
                ]))
        blocked = wait_for(
            lambda: observe(client, records),
            600,
            phase='terminal_during_fault')
        # Hold across informer delivery and retry windows; a single early read
        # before the worker received terminal state is not fault evidence.
        for _ in range(6):
            time.sleep(5)
            require(not permitted('get', 'ml-pipeline'),
                    'reporting_fault_disappeared')
            require(observe(client, records), 'reporting_blocked_evidence_lost')
        logs = kube('-n', 'kubeflow', 'logs',
                    'deployment/ml-pipeline-persistenceagent', '--since=5m')
        require(
            all(
                any(record['workflow_name'] in line and
                    'transient failure' in line
                    for line in logs.splitlines())
                for record in records), 'reporting_worker_retry_not_observed')
        write_object(state_dir / 'reporting-blocked.json',
                     dict(runs=blocked, worker_retry_observed=True))
    finally:
        kube(
            '-n', NAMESPACE, 'patch', 'role/' + API_ROLE, '--type=json', '-p',
            json.dumps([
                dict(op='test', path='/rules', value=fault),
                dict(op='replace', path='/rules', value=rules)
            ]))
    wait_for(lambda: permitted('get', 'ml-pipeline'), 60)
    recovered = wait_for(
        lambda: observe(client, records, recovered=True),
        600,
        phase='persistence_catchup')
    require(worker_pods() == original_worker,
            'reporting_worker_restarted_during_recovery')
    return dict(
        scope='live_persistence_lookup_recovery',
        outcome='passed',
        worker_restarted=False,
        runs=recovered,
        deletion_before_worker_snapshot_validated=False)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'phase',
        choices=('prepare', 'source', 'recover', 'delete', 'restore',
                 'proxy-diagnostics'))
    parser.add_argument('--fixture-state', type=Path, required=True)
    parser.add_argument('--state-dir', type=Path, required=True)
    parser.add_argument('--endpoint')
    parser.add_argument('--proxy-endpoint')
    parser.add_argument('--token-file')
    parser.add_argument('--run-id', action='append', default=[])
    args = parser.parse_args()
    fixture = read_object(args.fixture_state)
    verify_state(CONTEXT, fixture)
    args.state_dir.mkdir(parents=True, exist_ok=True)
    if args.phase == 'proxy-diagnostics':
        result = proxy_diagnostics()
        write_object(args.state_dir / 'reporting-proxy-diagnostics.json',
                     result)
        print(json.dumps(result, sort_keys=True), file=sys.stderr)
        return
    if args.phase == 'restore':
        restore(args.state_dir)
        return
    require(args.endpoint and args.token_file,
            'reporting_endpoint_and_token_required')
    client = Client(args.endpoint, args.token_file)
    source_path = args.state_dir / 'reporting-source.json'
    if args.phase == 'prepare':
        require(not source_path.exists(), 'reporting_source_already_captured')
        write_object(
            source_path,
            prepare(
                SourceClient(args.endpoint, args.token_file), client,
                args.state_dir))
    elif args.phase == 'source':
        require(not source_path.exists(), 'reporting_source_already_captured')
        write_object(source_path, capture(client, args.run_id))
    elif args.phase == 'delete':
        require(args.proxy_endpoint, 'reporting_proxy_endpoint_required')
        result = delete_captured(client, Client(args.proxy_endpoint),
                                 read_object(source_path))
        write_object(args.state_dir / 'reporting-deleted.json', result)
    else:
        result = recover(client, read_object(source_path), args.state_dir)
        write_object(args.state_dir / 'reporting-recovered.json', result)


if __name__ == '__main__':
    main()
