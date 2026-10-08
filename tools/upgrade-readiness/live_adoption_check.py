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
}
SQL_OPERATIONS = {
    'jobs', 'run_details', 'recurring_run_states', 'recurring_run_adoptions'
}


def command_failure(operation, stderr):
    # Never include backend messages: they can contain queries or credentials.
    if operation in SQL_OPERATIONS:
        code = re.search(r'ERROR ([0-9]+) \([A-Z0-9]+\)', stderr)
        category = MYSQL_ERRORS.get(code.group(1),
                                    'command') if code else 'command'
        return 'fixture_sql_' + operation + '_' + category + '_failed'
    return 'fixture_kubernetes_operation_failed'


def kube(*args, value=None, operation='kubernetes'):
    result = subprocess.run(
        ['kubectl', '--context', CONTEXT, '--request-timeout=30s', *args],
        input=json.dumps(value) if value is not None else None,
        text=True,
        capture_output=True,
        timeout=45,
        check=False)
    if result.returncode:
        raise AdoptionError(command_failure(operation, result.stderr))
    if len(result.stdout) > 4 * 1024 * 1024:
        raise AdoptionError('fixture_collection_limit_exceeded')
    return result.stdout


def get(namespace, resource):
    return json.loads(kube('-n', namespace, 'get', resource, '-o', 'json'))


def sql(table, columns, where=''):
    # Only fixed table/column names and this fixture namespace reach SQL.
    require(table in SQL_OPERATIONS, 'fixture_sql_table_not_allowed')
    fields = ','.join("'%s', `%s`" % (c, c) for c in columns)
    query = f'SELECT JSON_OBJECT({fields}) FROM {table} {where} ORDER BY 1'
    raw = kube(
        '-n',
        'kubeflow',
        'exec',
        'deployment/mysql',
        '--',
        'mysql',
        '-uroot',
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
            run['Name'] == workflow['name'], 'workflow_run_mismatch')
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
        require(
            state['LastRunIndex'] == int(trigger['lastWorkflowIndex']) and
            state['LastRunIndex'] > 0 and not state['Pending'],
            'historical_progress_reset')
        scheduled_at = int(
            datetime.fromisoformat(trigger['lastTriggeredTime'].replace(
                'Z', '+00:00')).timestamp())
        require(
            state['LastScheduledAtInSec'] == scheduled_at and
            schedule['trigger']['lastTriggeredTime']
            == trigger['lastTriggeredTime'], 'historical_time_changed')
        require(
            state['LastRunIndex'] == int(
                schedule['trigger']['lastWorkflowIndex']),
            'synchronized_progress_mismatch')
        matches = [
            w for w in before['workflows']
            if w['owners'] == [schedule['uid']] and
            w['index'] == state['LastRunIndex']
        ]
        require(
            len(matches) == 1 and state['LastRunUUID'] == matches[0]['run_id'],
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


def prepare_active(state, fixture):
    client = FixtureClient('http://127.0.0.1:8888', state / 'token')
    case = next(r for r in fixture['schedules'] if r['scenario'] == 'default')
    baseline = snapshot()
    old = {w['uid'] for w in baseline['workflows']}
    client.post(
        '/apis/v2beta1/recurringruns/' + case['schedule_uid'] + ':enable', {})
    deadline = time.monotonic() + 180
    while time.monotonic() < deadline:
        current = snapshot()
        fresh = [w for w in current['workflows'] if w['uid'] not in old]
        if fresh:
            require(len(fresh) == 1, 'multiple_active_workflows')
            workflow = fresh[0]
            kube('-n', NAMESPACE, 'patch', 'workflow/' + workflow['name'],
                 '--type=merge', '-p', '{"spec":{"suspend":true}}')
            if (any(r['UUID'] == workflow['run_id'] and
                    str(r['State'] or r['Conditions']).upper() not in TERMINAL
                    for r in current['runs']) and
                    any(s['uid'] == case['schedule_uid'] and
                        int(s['trigger'].get('lastWorkflowIndex',
                                             0)) == workflow['index']
                        for s in current['schedules'])):
                write_object(state / 'active.json', workflow)
                return
        time.sleep(3)
    raise AdoptionError('source_active_run_not_persisted')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        'phase',
        choices=('active', 'snapshot', 'stopped', 'job', 'adopted',
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
            prepare_active(state, fixture)
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
    write_object(state / 'reports' / ('adoption-' + args.phase + '.json'),
                 report)
    return 0 if report['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
