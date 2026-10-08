# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Mutating adoption acceptance for the disposable readiness Kind cluster
only."""

import argparse
import copy
from datetime import datetime
import json
from pathlib import Path
import re
import subprocess
import time

from fixture_http import FixtureClient
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
from provision_live_schedules import verify_state
from provision_live_schedules import write_object

CONTROLLERS = ('ml-pipeline', 'ml-pipeline-scheduledworkflow',
               'ml-pipeline-persistenceagent', 'workflow-controller')
PREFIX = 'scheduledworkflows.kubeflow.org/'
TERMINAL = {'SUCCEEDED', 'FAILED', 'ERROR', 'SKIPPED', 'CANCELED'}

MYSQL_ERRORS = {
    '1038': 'sort_memory',
    '1045': 'authentication',
    '1049': 'database_missing',
    '1054': 'column_missing',
    '1064': 'query_syntax',
    '1146': 'table_missing',
    '1305': 'function_missing',
    '3144': 'json_character_set',
    '2002': 'connection',
    '2003': 'connection',
}
SQL_OPERATIONS = {
    'jobs', 'run_details', 'recurring_run_states', 'recurring_run_adoptions'
}


def command_failure(operation, stderr):
    # Never include backend messages: they can contain queries or credentials.
    if operation in SQL_OPERATIONS:
        code = re.search(r'\bERROR\s+([0-9]{4})\b', stderr, re.IGNORECASE)
        category = MYSQL_ERRORS.get(code.group(1),
                                    'command') if code else 'command'
        return 'fixture_sql_' + operation + '_' + category + '_failed'
    return 'fixture_kubernetes_operation_failed'


def command_diagnostics(result):
    # Preserve unknown numeric SQL errors without exposing messages or queries.
    code = re.search(r'\bERROR\s+([0-9]{4})\b',
                     result.stderr + '\n' + result.stdout, re.IGNORECASE)
    return dict(
        exit_code=result.returncode,
        mysql_error=int(code.group(1)) if code else None)


def kube(*args, value=None, operation='kubernetes'):
    result = subprocess.run(
        ['kubectl', '--context', CONTEXT, '--request-timeout=30s', *args],
        input=json.dumps(value) if value is not None else None,
        text=True,
        capture_output=True,
        timeout=45,
        check=False)
    if result.returncode:
        error = AdoptionError(command_failure(operation, result.stderr))
        error.command_diagnostics = command_diagnostics(result)
        raise error
    if len(result.stdout) > 4 * 1024 * 1024:
        raise AdoptionError('fixture_collection_limit_exceeded')
    return result.stdout


def get(namespace, resource):
    return json.loads(kube('-n', namespace, 'get', resource, '-o', 'json'))


def sql(table, columns, where=''):
    # Only fixed table/column names and this fixture namespace reach SQL.
    require(table in SQL_OPERATIONS, 'fixture_sql_table_not_allowed')
    fields = ','.join("'%s', `%s`" % (c, c) for c in columns)
    # Callers put the primary identity column first. Sort that key, not the
    # generated JSON containing potentially large execution definitions.
    query = f'SELECT JSON_OBJECT({fields}) FROM {table} {where} ORDER BY `{columns[0]}`'
    # The source image's default client socket is unavailable in this fixture.
    # Exec into the same Pod but connect explicitly through loopback TCP.
    raw = kube(
        '-n',
        'kubeflow',
        'exec',
        'deployment/mysql',
        '--',
        'mysql',
        '-uroot',
        '--protocol=TCP',
        '--host=127.0.0.1',
        '--batch',
        '--skip-column-names',
        '--raw',
        'mlpipeline',
        '-e',
        query,
        operation=table)
    return [json.loads(line) for line in raw.splitlines() if line]


def snapshot(adopted=False):
    jobs = sql(
        'jobs',
        ('UUID', 'Name', 'Namespace', 'ServiceAccount', 'Enabled',
         'MaxConcurrency', 'NoCatchup', 'ExperimentUUID', 'PipelineId',
         'PipelineVersionId', 'PipelineSpecManifest', 'WorkflowSpecManifest',
         'RuntimeParameters', 'PipelineRoot', 'Parameters', 'IntervalSecond',
         'Schedule', 'PeriodicScheduleStartTimeInSec',
         'PeriodicScheduleEndTimeInSec', 'CronScheduleStartTimeInSec',
         'CronScheduleEndTimeInSec'), f"WHERE Namespace='{NAMESPACE}'")
    runs = sql('run_details', ('UUID', 'Name', 'JobUUID', 'ScheduledAtInSec',
                               'CreatedAtInSec', 'State', 'Conditions'),
               f"WHERE Namespace='{NAMESPACE}'")
    schedules = []
    for obj in get(NAMESPACE, 'scheduledworkflows')['items']:
        schedules.append(
            dict(
                uid=obj['metadata']['uid'],
                name=obj['metadata']['name'],
                enabled=obj['spec'].get('enabled', False),
                trigger=obj.get('status', {}).get('trigger', {})))
    workflows = []
    for obj in get(NAMESPACE, 'workflows')['items']:
        meta = obj['metadata']
        labels = meta.get('labels', {})
        owners = [
            o['uid']
            for o in meta.get('ownerReferences', [])
            if o.get('kind') == 'ScheduledWorkflow' and o.get('controller')
        ]
        workflows.append(
            dict(
                uid=meta['uid'],
                name=meta['name'],
                owners=owners,
                run_id=labels.get('pipeline/runid'),
                index=int(labels.get(PREFIX + 'workflowIndex', '0')),
                epoch=int(labels.get(PREFIX + 'workflowEpoch', '0')),
                phase=obj.get('status', {}).get('phase', ''),
                suspended=obj.get('spec', {}).get('suspend', False)))
    result = dict(
        jobs=sorted(jobs, key=lambda row: row['UUID']),
        runs=sorted(runs, key=lambda row: row['UUID']),
        schedules=sorted(schedules, key=lambda row: row['uid']),
        workflows=sorted(workflows, key=lambda row: row['uid']))
    if adopted:
        result['states'] = sql(
            'recurring_run_states',
            ('JobUUID', 'LastRunUUID', 'LastRunIndex', 'LastScheduledAtInSec',
             'LastCreatedAtInSec', 'RequestKey', 'Pending'))
        result['receipts'] = sql(
            'recurring_run_adoptions',
            ('ID', 'AdoptedCount', 'CompletedAt', 'JobIDs', 'Ready'))
    return result


class AdoptionError(ValueError):
    """Only fixed local invariant codes; never wrap backend error text."""


def require(condition, reason):
    if not condition:
        raise AdoptionError(reason)


def indexed(items, key):
    result = {item[key]: item for item in items}
    require(len(result) == len(items), 'duplicate_identity')
    return result


def validate_inventory(value, fixture):
    ids = {r['schedule_uid'] for r in fixture['schedules']}
    require(len(ids) == 3, 'three_source_schedules_required')
    require(set(indexed(value['jobs'], 'UUID')) == ids, 'job_identity_changed')
    require(
        set(indexed(value['schedules'], 'uid')) == ids,
        'schedule_identity_changed')
    runs = indexed(value['runs'], 'UUID')
    seen = set()
    for workflow in value['workflows']:
        require(
            len(workflow['owners']) == 1 and workflow['owners'][0] in ids,
            'workflow_owner_mismatch')
        identity = (workflow['owners'][0], workflow['index'])
        require(workflow['index'] > 0 and identity not in seen,
                'duplicate_tick')
        seen.add(identity)
        run = runs.get(workflow['run_id'])
        require(
            run is not None and run['JobUUID'] == workflow['owners'][0] and
            run['Name'] == workflow['name'] and
            run['ScheduledAtInSec'] == workflow['epoch'],
            'workflow_run_mismatch')
    return ids


def validate_adoption(before, after, fixture):
    ids = validate_inventory(before, fixture)
    validate_inventory(after, fixture)
    require(before['jobs'] == after['jobs'], 'stored_definition_changed')
    require(before['runs'] == after['runs'], 'historical_run_changed')
    require(before['workflows'] == after['workflows'],
            'workflow_changed_offline')
    require(len(after['receipts']) == 1, 'single_receipt_required')
    receipt = after['receipts'][0]
    require(
        receipt['ID'] == 'legacy-2.18' and receipt['Ready'] == 1 and
        receipt['AdoptedCount'] == len(ids) and receipt['CompletedAt'] > 0 and
        set(json.loads(receipt['JobIDs'])) == ids, 'invalid_receipt')
    states = indexed(after['states'], 'JobUUID')
    require(set(states) == ids, 'adoption_state_inventory_mismatch')
    original = indexed(before['schedules'], 'uid')
    for schedule in after['schedules']:
        previous = original[schedule['uid']]
        state = states[schedule['uid']]
        trigger = previous['trigger']
        require(schedule['enabled'] == previous['enabled'],
                'enablement_changed')
        acknowledged = int(trigger['lastWorkflowIndex'])
        scheduled_at = int(
            datetime.fromisoformat(trigger['lastTriggeredTime'].replace(
                'Z', '+00:00')).timestamp())
        workflows = [
            w for w in before['workflows'] if w['owners'] == [schedule['uid']]
        ]
        latest = max(workflows, key=lambda w: w['index'])
        require(latest['index'] in (acknowledged, acknowledged + 1),
                'source_progress_gap')
        run = next(r for r in before['runs'] if r['UUID'] == latest['run_id'])
        if latest['index'] == acknowledged + 1:
            # Source 2.17.2 can persist submission before acknowledging the
            # trigger. Recover the due tick under this fixture's stored
            # periodic/no-catchup contract; never reset source CR status.
            job = next(
                j for j in before['jobs'] if j['UUID'] == schedule['uid'])
            require(job['NoCatchup'] == 1 and job['IntervalSecond'] == 30,
                    'fixture_schedule_timing_changed')
            next_due = scheduled_at + job['IntervalSecond']
            scheduled_at = latest['epoch']
            # This fixture's source records use the controller request name.
            # Only the legacy timestamp-equal representation needs due-time
            # reconstruction; otherwise the stored scheduled epoch is trusted.
            if run['ScheduledAtInSec'] == run['CreatedAtInSec']:
                scheduled_at = latest['epoch'] if latest[
                    'epoch'] >= next_due + job['IntervalSecond'] else next_due
            require(scheduled_at <= latest['epoch'],
                    'source_submission_not_due')
        require(
            state['LastRunIndex'] == latest['index'] and
            state['LastRunIndex'] > 0 and not state['Pending'],
            'historical_progress_reset')
        synchronized_at = int(
            datetime.fromisoformat(
                schedule['trigger']['lastTriggeredTime'].replace(
                    'Z', '+00:00')).timestamp())
        require(
            state['LastScheduledAtInSec'] == scheduled_at and
            synchronized_at == scheduled_at, 'historical_time_changed')
        require(
            state['LastRunIndex'] == int(
                schedule['trigger']['lastWorkflowIndex']),
            'synchronized_progress_mismatch')
        require(
            state['LastRunUUID'] == latest['run_id'] and
            state['LastCreatedAtInSec'] == run['CreatedAtInSec'],
            'last_run_identity_changed')
    return receipt


def validate_continuation(before, current, fixture, held=False, drained=False):
    validate_inventory(current, fixture)
    expected_jobs = copy.deepcopy(before['jobs'])
    if drained:
        for job in expected_jobs:
            job['Enabled'] = 0
    require(expected_jobs == current['jobs'], 'job_definition_changed')
    old_runs = indexed(before['runs'], 'UUID')
    new_runs = indexed(current['runs'], 'UUID')
    require(set(old_runs) <= set(new_runs), 'historical_runs_lost')
    for uid, previous in old_runs.items():
        require(
            all(new_runs[uid][k] == previous[k]
                for k in ('UUID', 'Name', 'JobUUID', 'ScheduledAtInSec',
                          'CreatedAtInSec')), 'historical_identity_changed')
    enabled = {j['UUID'] for j in before['jobs'] if j['Enabled']}
    require(len(enabled) == 1, 'one_enabled_schedule_required')
    extra = [r for r in current['runs'] if r['UUID'] not in old_runs]
    require(
        all(r['JobUUID'] in enabled for r in extra), 'disabled_schedule_fired')
    if drained:
        require(
            all(
                str(r['State'] or r['Conditions']).upper() == 'SUCCEEDED'
                for r in current['runs']), 'fixture_runs_not_drained')
        require(not any(s['enabled'] for s in current['schedules']),
                'fixture_schedule_not_disabled')
    if held:
        require(not extra, 'active_run_escaped_concurrency_accounting')
    else:
        require(
            extra and any(
                str(r['State'] or r['Conditions']).upper() == 'SUCCEEDED'
                for r in extra), 'candidate_tick_not_successful')
        active = [w for w in before['workflows'] if w['suspended']]
        require(
            len(active) == 1 and
            str(new_runs[active[0]['run_id']]['State'] or
                new_runs[active[0]['run_id']]['Conditions']).upper()
            == 'SUCCEEDED', 'source_active_run_not_completed')
        old_indices = {
            w['index'] for w in before['workflows'] if w['owners'][0] in enabled
        }
        fresh = [
            w['index']
            for w in current['workflows']
            if w['run_id'] not in old_runs
        ]
        require(
            fresh and min(fresh) == max(old_indices) + 1,
            'candidate_tick_skipped_or_replayed')


def validate_idempotent(first, repeated):
    require(first == repeated, 'rerun_changed_adoption')


def require_stopped():
    for name in CONTROLLERS:
        deployment = get('kubeflow', 'deployment/' + name)
        require(
            deployment['spec']['replicas'] == 0 and
            deployment.get('status', {}).get('replicas', 0) == 0,
            'writers_not_stopped')
        labels = deployment['spec']['selector']['matchLabels']
        require(bool(labels), 'writer_selector_missing')
        selector = ','.join(
            key + '=' + value for key, value in sorted(labels.items()))
        pods = json.loads(
            kube('-n', 'kubeflow', 'get', 'pods', '-l', selector, '-o', 'json'))
        require(pods.get('items') == [], 'writer_pods_not_terminated')


def offline_job(deployment, name):
    spec = copy.deepcopy(deployment['spec']['template']['spec'])
    require(len(spec['containers']) == 1, 'fixture_api_sidecar_unexpected')
    spec['restartPolicy'] = 'Never'
    container = spec['containers'][0]
    container['command'] = ['/bin/apiserver']
    container['args'] = [
        '--config=/config', '--adopt-legacy-recurring-runs', '-logtostderr=true'
    ]
    for key in ('startupProbe', 'readinessProbe', 'livenessProbe', 'ports'):
        container.pop(key, None)
    return dict(
        apiVersion='batch/v1',
        kind='Job',
        metadata=dict(name=name, namespace='kubeflow'),
        spec=dict(
            backoffLimit=0,
            activeDeadlineSeconds=300,
            template=dict(metadata=dict(labels={'app': name}), spec=spec)))


def adoption_log_categories(text):
    lowered = text.lower()
    patterns = {
        'incomplete_progress': ('invalid or incomplete scheduling progress',),
        'future_trigger_time':
            ('last triggered time is invalid or in the future',),
        'duplicate_run': ('duplicate run ',),
        'duplicate_run_index': ('multiple runs claim index',),
        'skipped_run_index': ('skips scheduling indices',),
        'invalid_run_timestamps': ('has invalid execution timestamps',),
        'trigger_time_conflict': ('conflicts with the last triggered time',),
        'earlier_progress_conflict':
            ('conflicts with earlier scheduling progress',),
        'workflow_identity_conflict': ('has conflicting schedule identity',),
        'workflow_index_invalid': ('has an invalid or skipped index',),
        'duplicate_workflow_index': ('multiple workflows claim index',),
        'workflow_time_invalid': ('has an invalid scheduled time',),
        'workflow_run_mismatch': ('conflicts with its persisted run',),
        'due_recovery_failed': ('cannot recover the due time of run',),
        'unacknowledged_time_not_advanced': ('does not advance scheduled time',
                                            ),
        'persisted_time_mismatch': ('persisted execution time differs',),
        'persisted_index_invalid': ('persisted execution index is invalid',),
        'panic': ('panic:',),
        'runtime_fault': ('runtime error:', 'fatal error:'),
        'flag_parse': ('flag provided but not defined', 'invalid value',
                       'flag needs an argument'),
        'adoption_failed': ('legacy recurring-run adoption failed',),
        'execution_index':
            ('has no valid controller index', 'has an invalid or skipped index',
             'multiple runs claim index', 'multiple workflows claim index',
             'persisted execution index is invalid'),
        'execution_time': ('has invalid execution timestamps',
                           'last triggered time is invalid',
                           'conflicts with the last triggered time',
                           'persisted execution time differs',
                           'stored schedule had no tick due',
                           'schedule creation time is missing'),
        'execution_mismatch': ('has inconsistent execution identity',
                               'conflicts with its persisted run'),
        'concurrency_accounting': ('would escape concurrency accounting',),
        'backing_cr_unavailable': ('backing cr is unavailable',),
        'adoption_mode': ('adoption requires multiuser=true',),
        'adoption_inventory_changed':
            ('rebuild and review the adoption inventory',
             'rebuild the adoption inventory'),
        'unpersisted_workflow': ('has no persisted run',),
        'progress': ('scheduling progress', 'scheduling indices', 'due time',
                     'scheduled time'),
        'identity': ('identities differ', 'identity changed',
                     'identity differs', 'conflicting schedule identity'),
        'namespace': ('no stored namespace', 'namespace is empty'),
        'receipt': ('receipt',),
        'authorization': ('forbidden', 'permissiondenied', 'unauthorized'),
        'connection': ('connection refused', "can't connect", 'no such host'),
        'configuration':
            ('failed to parse', 'flag provided but not defined', 'config file'),
        'initialization': ('failed to initialize clientmanager',
                           'failed to initialize config',
                           'failed to initialize pipeline size limits'),
        'database_initialization':
            ('failed to detect schema version',
             'failed to initialize experiment store',
             'failed to initialize db status store',
             'failed to initialize default experiment store',
             'failed to retrieve *sql.db'),
        'object_store_initialization': ('failed to initialize object store',
                                        'failed to open blob storage bucket'),
        'init_dependency_timeout': ('error: timed out waiting for',),
        'init_dependency_configuration':
            ('error: wait_host or wait_port is not set.',),
        'synchronization': ('not synchronized',),
        'inventory': ('inventory unavailable', 'inventory is unavailable'),
        'validation': ('cannot be adopted', 'invalid input', 'invalidinput'),
    }
    return sorted(name for name, fragments in patterns.items()
                  if any(fragment in lowered for fragment in fragments))


def adoption_startup_milestones(text):
    # Presence only: a truncated log tail cannot prove an earlier stage absent.
    patterns = {
        'database_started':
            'Initializing DB client...',
        'database_ready':
            'DB client initialized successfully',
        'legacy_schema_detected':
            'Detected legacy schema. Running upgrade flow.',
        'schema_migration_started':
            'Running AutoMigrate.',
        'object_store_started':
            'Initializing Object store client...',
        'object_store_ready':
            'Object store client initialized successfully',
        'client_manager_ready':
            'Client manager initialized successfully',
    }
    return sorted(
        name for name, fragment in patterns.items() if fragment in text)


def adoption_stack_frames(text):
    """Return only source-verified repository locations and function names."""
    root = Path(__file__).resolve().parents[2]
    frames = []
    previous = ''
    for line in text.splitlines():
        match = re.fullmatch(
            r'\s+(?:[^\s]*?/)?(backend/[A-Za-z0-9_/-]+\.go):([0-9]+)(?: \+0x[0-9a-f]+)?',
            line)
        if match and len(frames) < 20:
            path = root / match[1]
            # A path-shaped payload is not evidence of a repository frame.
            if path.is_file() and path.resolve().is_relative_to(root):
                source = path.read_text()
                number = int(match[2])
                if 0 < number <= len(source.splitlines()):
                    frame = dict(file=match[1], line=number)
                    symbol = re.search(
                        r'\.([A-Za-z_][A-Za-z_0-9]*)(?:\.[0-9]+)?\(', previous)
                    if symbol and re.search(
                            r'func\s+(?:\([^\n]*?\)\s+)?' +
                            re.escape(symbol[1]) + r'\s*\(', source):
                        frame['symbol'] = symbol[1]
                    if frame not in frames:
                        frames.append(frame)
        previous = line
    return frames


def adoption_job_diagnostics(name):
    result = dict(containers=[], pods=[], receipt_logged=False)
    try:
        pods = json.loads(
            kube('-n', 'kubeflow', 'get', 'pods', '-l', 'job-name=' + name,
                 '-o', 'json'))['items']
        require(len(pods) <= 2, 'adoption_job_pod_limit')
        reasons = {
            'Completed', 'Error', 'OOMKilled', 'ContainerCreating',
            'CrashLoopBackOff', 'ImagePullBackOff', 'ErrImagePull',
            'CreateContainerConfigError', 'CreateContainerError',
            'PodInitializing', 'DeadlineExceeded', 'StartError'
        }
        for pod in pods:
            phase = pod.get('status', {}).get('phase')
            result['pods'].append(
                dict(
                    phase=phase if phase in ('Pending', 'Running', 'Succeeded',
                                             'Failed',
                                             'Unknown') else 'unknown',
                    scheduled=any(
                        c.get('type') == 'PodScheduled' and
                        c.get('status') == 'True'
                        for c in pod.get('status', {}).get('conditions', []))))
            for role, field_name in (('init', 'initContainerStatuses'),
                                     ('main', 'containerStatuses')):
                statuses = pod.get('status', {}).get(field_name, [])
                require(len(statuses) <= 6, 'adoption_job_container_limit')
                for container in statuses:
                    state = container.get('state', {})
                    phase = next(
                        (p for p in ('waiting', 'running', 'terminated')
                         if p in state), 'unknown')
                    detail = state.get(phase, {})
                    entry = dict(
                        role=role,
                        state=phase,
                        reason=detail.get('reason')
                        if detail.get('reason') in reasons else 'other',
                        exit_code=detail.get('exitCode'))
                    try:
                        logs = kube('-n', 'kubeflow', 'logs',
                                    pod['metadata']['name'],
                                    '--container=' + container['name'],
                                    '--limit-bytes=1048576')
                        entry['log_categories'] = adoption_log_categories(logs)
                        entry['stack_frames'] = adoption_stack_frames(logs)
                        entry[
                            'startup_milestones'] = adoption_startup_milestones(
                                logs)
                        if role == 'main' and re.search(
                                r'recurring_run_adoption id=legacy-2.18 ready=true adopted_count=3 ',
                                logs):
                            result['receipt_logged'] = True
                    except (OSError, ValueError, subprocess.TimeoutExpired):
                        entry['log_collection'] = 'unavailable'
                    result['containers'].append(entry)
        result['pod_count'] = len(pods)
    except (OSError, ValueError, KeyError, TypeError,
            subprocess.TimeoutExpired):
        result['collection'] = 'unavailable'
    return result


def wait_adoption_job(name):
    require(
        name in ('readiness-adopt-first', 'readiness-adopt-repeat'),
        'invalid_job_name')
    deadline = time.monotonic() + 330
    outcome = 'timeout'
    try:
        while time.monotonic() < deadline:
            status = get('kubeflow', 'job/' + name).get('status', {})
            conditions = {
                c.get('type')
                for c in status.get('conditions', [])
                if c.get('status') == 'True'
            }
            if 'Failed' in conditions or status.get('failed', 0) > 0:
                outcome = 'failed'
                break
            if 'Complete' in conditions:
                outcome = 'complete'
                break
            time.sleep(5)
    except (OSError, ValueError, KeyError, TypeError,
            subprocess.TimeoutExpired):
        outcome = 'collection_failed'
    evidence = adoption_job_diagnostics(name)
    evidence['job_outcome'] = outcome
    if (outcome != 'complete' or not evidence['receipt_logged'] or
            evidence.get('collection') == 'unavailable'):
        error = AdoptionError('adoption_job_not_successfully_completed')
        error.job_evidence = evidence
        raise error
    return evidence


def prepare_active(state, fixture):
    client = FixtureClient('http://127.0.0.1:8888', state / 'token')
    case = next(r for r in fixture['schedules'] if r['scenario'] == 'default')
    baseline = snapshot()
    old = {w['uid'] for w in baseline['workflows']}
    client.post(
        '/apis/v2beta1/recurringruns/' + case['schedule_uid'] + ':enable', {})
    deadline = time.monotonic() + 180
    observation = {}
    while time.monotonic() < deadline:
        current = snapshot()
        fresh = [w for w in current['workflows'] if w['uid'] not in old]
        schedule = next((s for s in current['schedules']
                         if s['uid'] == case['schedule_uid']), {})
        job = next(
            (j for j in current['jobs'] if j['UUID'] == case['schedule_uid']),
            {})
        observation = dict(
            fresh_workflow_count=len(fresh),
            schedule_enabled=schedule.get('enabled') is True,
            job_enabled=bool(job.get('Enabled')),
            schedule_index=int(
                schedule.get('trigger', {}).get('lastWorkflowIndex', 0)),
            persisted_run_found=False,
            persisted_run_nonterminal=False,
            controller_acknowledged=False)
        if fresh:
            require(len(fresh) == 1, 'multiple_active_workflows')
            workflow = fresh[0]
            kube('-n', NAMESPACE, 'patch', 'workflow/' + workflow['name'],
                 '--type=merge', '-p', '{"spec":{"suspend":true}}')
            run = next(
                (r for r in current['runs'] if r['UUID'] == workflow['run_id']),
                None)
            observation.update(
                workflow_index=workflow['index'],
                workflow_has_run_id=bool(workflow['run_id']),
                workflow_suspended=workflow['suspended'],
                workflow_phase=workflow['phase']
                if workflow['phase'] in ('Pending', 'Running', 'Succeeded',
                                         'Failed', 'Error') else 'unset',
                persisted_run_found=run is not None,
                persisted_run_nonterminal=run is not None and
                str(run['State'] or run['Conditions']).upper() not in TERMINAL,
                controller_acknowledged=observation['schedule_index'] ==
                workflow['index'])
            observation['recoverable_unacknowledged_submission'] = (
                workflow['index'] == observation['schedule_index'] + 1)
            if (observation['persisted_run_nonterminal'] and
                    workflow['suspended'] and
                (observation['controller_acknowledged'] or
                 observation['recoverable_unacknowledged_submission'])):
                write_object(state / 'active.json', workflow)
                return observation
        time.sleep(3)
    error = AdoptionError('source_active_run_not_persisted')
    error.source_observation = observation
    raise error


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'phase',
        choices=('active', 'snapshot', 'stopped', 'job', 'wait-job', 'adopted',
                 'idempotent', 'held', 'completed', 'drained'))
    parser.add_argument('--state', required=True)
    parser.add_argument('--job-name')
    args = parser.parse_args()
    state = Path(args.state)
    fixture = read_object(state / 'fixture/state.json')
    verify_state(CONTEXT, fixture)
    report = dict(
        scope='populated_legacy_adoption',
        phase=args.phase,
        outcome='inconclusive')
    try:
        if args.phase == 'active':
            report['source_observation'] = prepare_active(state, fixture)
        elif args.phase == 'wait-job':
            report['job_evidence'] = wait_adoption_job(args.job_name)
        elif args.phase == 'job':
            require(
                args.job_name
                in ('readiness-adopt-first', 'readiness-adopt-repeat'),
                'invalid_job_name')
            require_stopped()
            kube(
                'create',
                '-f',
                '-',
                value=offline_job(
                    get('kubeflow', 'deployment/ml-pipeline'), args.job_name))
        elif args.phase == 'stopped':
            require_stopped()
        elif args.phase == 'snapshot':
            require_stopped()
            current = snapshot()
            validate_inventory(current, fixture)
            require(
                sum(bool(j['Enabled']) for j in current['jobs']) == 1,
                'one_enabled_schedule_required')
            require(
                sum(w['suspended'] for w in current['workflows']) == 1,
                'one_active_workflow_required')
            require(
                all(j['MaxConcurrency'] == 1 and j['IntervalSecond'] == 30 and
                    j['NoCatchup'] == 1 for j in current['jobs']),
                'fixture_schedule_timing_changed')
            active = [w for w in current['workflows'] if w['suspended']][0]
            run = indexed(current['runs'], 'UUID')[active['run_id']]
            require(
                str(run['State'] or run['Conditions']).upper() not in TERMINAL,
                'source_active_run_is_terminal')
            write_object(state / 'before-adoption.json', current)
        else:
            before = read_object(state / 'before-adoption.json')
            current = snapshot(adopted=True)
            if args.phase == 'adopted':
                report['receipt'] = validate_adoption(before, current, fixture)
                write_object(state / 'after-adoption.json', current)
            elif args.phase == 'idempotent':
                validate_idempotent(
                    read_object(state / 'after-adoption.json'), current)
            else:
                validate_continuation(before, current, fixture,
                                      args.phase == 'held',
                                      args.phase == 'drained')
            report['schedule_count'] = len(current['jobs'])
            report['run_count'] = len(current['runs'])
        report['outcome'] = 'passed'
    except (OSError, ValueError, KeyError, TypeError,
            subprocess.TimeoutExpired) as error:
        # Never expose backend payloads, SQL errors, manifests or credentials.
        report['reason'] = str(error) if type(error) is AdoptionError else \
            'adoption_validation_or_collection_failed'
        if hasattr(error, 'job_evidence'):
            report['job_evidence'] = error.job_evidence
        if hasattr(error, 'source_observation'):
            report['source_observation'] = error.source_observation
        if report['reason'].startswith('fixture_sql_'):
            report['failed_command'] = getattr(error, 'command_diagnostics', {})
    phase_label = args.phase + ('-' +
                                (args.job_name or 'invalid').rsplit('-', 1)[-1]
                                if args.phase == 'wait-job' else '')
    write_object(state / 'reports' / ('adoption-' + phase_label + '.json'),
                 report)
    return 0 if report['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
