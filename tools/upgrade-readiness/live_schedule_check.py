# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Read-only, bounded observation of prepared isolated schedule fixtures.

A timeout is never evidence of policy rejection. This is a CI acceptance
helper, not an operator readiness scan. Successful run completion is
required only by explicit expectations or --require-run-success.
"""

import argparse
from datetime import datetime
from datetime import timezone
import json
import re
import time

from kfp_http import Client
from kfp_http import CollectionError
from kfp_inventory import field
from kfp_inventory import identifier
from readiness import kubectl_get

RUN_STATES = frozenset(
    ('RUNTIME_STATE_UNSPECIFIED', 'PENDING', 'RUNNING', 'SUCCEEDED', 'SKIPPED',
     'FAILED', 'CANCELING', 'CANCELED', 'PAUSED'))
FAILED_STATES = frozenset(('FAILED', 'CANCELED', 'SKIPPED'))
POSITIVE_OUTCOMES = ('run_created', 'run_succeeded')


def timestamp(value):
    if not isinstance(value, str):
        raise ValueError('invalid_timestamp')
    result = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if result.tzinfo is None:
        raise ValueError('invalid_timestamp')
    return result


def validate(bundle, report, namespace):
    if not isinstance(namespace,
                      str) or len(namespace) > 63 or not re.fullmatch(
                          r'[a-z0-9]([-a-z0-9]*[a-z0-9])?', namespace):
        raise ValueError('invalid_namespace')
    if bundle.get('namespace') != namespace:
        raise ValueError('namespace_mismatch')
    start = timestamp(bundle.get('observation_start'))
    if start > datetime.now(timezone.utc):
        raise ValueError('future_observation_start')
    cases = bundle.get('cases')
    if not isinstance(cases, list) or not 1 <= len(cases) <= 20:
        raise ValueError('invalid_cases')
    seen = set()
    for case in cases:
        for key in ('schedule_uid', 'schedule_name', 'service_account',
                    'scenario'):
            if not isinstance(case.get(key), str) or not re.fullmatch(
                    r'[a-zA-Z0-9][a-zA-Z0-9_.-]{0,252}', case[key]):
                raise ValueError('invalid_case_identifier')
        if case['schedule_uid'] in seen:
            raise ValueError('duplicate_schedule')
        seen.add(case['schedule_uid'])
        if case.get('expected_outcome') not in (*POSITIVE_OUTCOMES, 'blocked'):
            raise ValueError('invalid_outcome')
        allowed = (
            'policy_rejection',) if case['expected_outcome'] == 'blocked' else (
                'no_issue_detected', 'operational_impact')
        if case.get('expected_prediction') not in allowed:
            raise ValueError('incompatible_prediction')
        baseline = case.get('baseline_run_ids')
        counts = case.get('baseline_event_counts')
        if not isinstance(baseline, list) or len(baseline) > 10000 or not all(
                isinstance(i, str) and i for i in baseline):
            raise ValueError('invalid_run_baseline')
        if not isinstance(counts, dict) or len(counts) > 10000 or not all(
                isinstance(k, str) and k and type(v) is int and v >= 0
                for k, v in counts.items()):
            raise ValueError('invalid_event_baseline')
        matches = [
            f for f in report.get('findings', [])
            if f.get('rule') == 'schedule.targetMainAccount' and
            f.get('resource') == 'ScheduledWorkflow/' + namespace + '/' +
            case['schedule_name']
        ]
        if len(matches) != 1 or matches[0].get(
                'status') != case['expected_prediction']:
            raise ValueError('prediction_mismatch')
    if any(c['expected_outcome'] == 'blocked' for c in cases) and not any(
            c['expected_outcome'] in POSITIVE_OUTCOMES and
            c['expected_prediction'] == 'no_issue_detected' for c in cases):
        raise ValueError('positive_control_required')
    return cases, start


def activation_start(baseline_start, value):
    start = timestamp(value)
    if start < baseline_start or start > datetime.now(timezone.utc):
        raise ValueError('invalid_activation_start')
    return start


def list_runs(client, namespace, schedule_uid):
    """Verify every returned record even though the server receives a
    filter."""
    token, seen, fresh = '', set(), []
    experiments = set()
    for _ in range(20):
        response = client.get(
            '/apis/v2beta1/runs',
            dict(
                namespace=namespace,
                page_size=100,
                page_token=token,
                filter=json.dumps({
                    'predicates': [{
                        'key': 'recurring_run_id',
                        'operation': 'EQUALS',
                        'string_value': schedule_uid
                    }]
                })))
        if not isinstance(response, dict) or response.get('error'):
            raise CollectionError('invalid_run_response')
        records = response.get('runs', [])
        if not isinstance(records, list):
            raise CollectionError('invalid_run_list')
        for run in records:
            if not isinstance(run, dict) or run.get('error'):
                raise CollectionError('invalid_run')
            if field(run, 'recurring_run_id', 'recurringRunId') != schedule_uid:
                raise CollectionError('run_schedule_mismatch')
            experiment_id = field(run, 'experiment_id', 'experimentId')
            if experiment_id not in experiments:
                experiment = client.get('/apis/v2beta1/experiments/' +
                                        identifier(experiment_id))
                if field(experiment, 'experiment_id',
                         'experimentId') != experiment_id or experiment.get(
                             'namespace') != namespace:
                    raise CollectionError('run_namespace_mismatch')
                experiments.add(experiment_id)
            uid = field(run, 'run_id', 'runId')
            if not isinstance(uid, str) or not uid:
                raise CollectionError('invalid_run_id')
            fresh.append(run)
        token = field(response, 'next_page_token', 'nextPageToken')
        if token in (None, ''):
            return fresh
        if not isinstance(token, str) or token in seen:
            raise CollectionError('invalid_run_pagination')
        seen.add(token)
    raise CollectionError('run_page_limit')


def list_events(context, namespace):
    events, error = kubectl_get(context, namespace, 'events')
    if error:
        raise CollectionError(error)
    if not isinstance(events, dict) or not isinstance(
            events.get('items'), list):
        raise CollectionError('invalid_events')
    if len(events['items']) > 10000 or not all(
            isinstance(e, dict) for e in events['items']):
        raise CollectionError('invalid_events')
    return events['items']


def runs(client, namespace, case, start):
    return [
        record['run_id']
        for record in run_evidence(client, namespace, case, start)
    ]


def run_evidence(client, namespace, case, start):
    """Retain only fresh run identifiers and recognized API runtime states."""
    fresh = []
    for run in list_runs(client, namespace, case['schedule_uid']):
        uid = field(run, 'run_id', 'runId')
        if uid in case['baseline_run_ids'] or timestamp(
                field(run, 'created_at', 'createdAt')) < start:
            continue
        if field(run, 'service_account',
                 'serviceAccount') != case['service_account']:
            raise CollectionError('run_account_mismatch')
        if not re.fullmatch(r'[a-zA-Z0-9][a-zA-Z0-9_.-]{0,252}', uid):
            raise CollectionError('invalid_run_id')
        state = run.get('state')
        fresh.append(
            dict(
                run_id=uid,
                state=state if isinstance(state, str) and state in RUN_STATES
                else 'UNKNOWN'))
    return fresh


def denied(events, namespace, case, start):
    """Require a fresh controller Event for this UID and precise SA use
    denial."""
    for event in events:
        ref = event.get('involvedObject', {})
        if (ref.get('uid') != case['schedule_uid'] or
                ref.get('name') != case['schedule_name'] or
                ref.get('namespace') != namespace or
                ref.get('kind') != 'ScheduledWorkflow'):
            continue
        if (event.get('reason') != 'Failed' or event.get('type') != 'Warning' or
                event.get('source', {}).get('component')
                != 'scheduled-workflow-controller'):
            continue
        uid = event.get('metadata', {}).get('uid')
        count = event.get('count', 1)
        if not isinstance(uid, str) or type(
                count) is not int or count <= case['baseline_event_counts'].get(
                    uid, 0):
            continue
        observed = event.get('series', {}).get('lastObservedTime') or event.get(
            'lastTimestamp') or event.get('eventTime')
        if timestamp(observed) < start:
            continue
        message = event.get('message', '')
        required = ('code = PermissionDenied',
                    'service account authorization error', 'Verb:use,',
                    'Group:,', 'Resource:serviceaccounts,', 'Subresource:,',
                    'Namespace:' + namespace + ',',
                    'Name:' + case['service_account'] + ',')
        if isinstance(message, str) and all(
                part in message for part in required):
            return True
    return False


def observe(client,
            context,
            namespace,
            cases,
            start,
            timeout,
            get=list_events,
            clock=time.monotonic,
            sleep=time.sleep):
    deadline = clock() + timeout
    observed = {c['schedule_uid']: {'runs': {}, 'denial': False} for c in cases}
    while True:
        final_collection = clock() >= deadline
        events = get(context, namespace)
        for case in cases:
            state = observed[case['schedule_uid']]
            state['runs'].update({
                record['run_id']: record['state']
                for record in run_evidence(client, namespace, case, start)
            })
            state['denial'] |= denied(events, namespace, case, start)
            if case['expected_outcome'] == 'blocked' and state['runs']:
                return result(cases, observed, 'failed')
            if case['expected_outcome'] == 'run_succeeded' and any(
                    value in FAILED_STATES for value in state['runs'].values()):
                return result(cases, observed, 'failed')
        # Finish with evidence collected after the entire observation window,
        # including when an earlier collection crosses the deadline.
        if final_collection:
            break
        sleep(min(10, max(0, deadline - clock())))
    require_success = any(
        c['expected_outcome'] == 'run_succeeded' for c in cases)
    control = any(
        positive_complete(c, observed[c['schedule_uid']], require_success)
        for c in cases
        if c['expected_outcome'] in POSITIVE_OUTCOMES and
        c['expected_prediction'] == 'no_issue_detected')
    for case in cases:
        state = observed[case['schedule_uid']]
        if case['expected_outcome'] == 'blocked':
            if not (state['denial'] and control):
                return result(cases, observed, 'inconclusive')
        elif not positive_complete(case, state):
            return result(cases, observed, 'inconclusive')
    return result(cases, observed, 'passed')


def positive_complete(case, observed, require_success=False):
    if require_success or case['expected_outcome'] == 'run_succeeded':
        return 'SUCCEEDED' in observed['runs'].values()
    return bool(observed['runs'])


def result(cases, observed, outcome):
    reports = []
    scope = 'schedule_run_creation_only'
    for case in cases:
        if case['expected_outcome'] == 'run_succeeded':
            scope = 'schedule_run_completion'
        state = observed[case['schedule_uid']]
        reports.append(
            dict(
                scenario=case['scenario'],
                schedule_uid=case['schedule_uid'],
                expected_prediction=case['expected_prediction'],
                expected_outcome=case['expected_outcome'],
                run_observed=bool(state['runs']),
                success_observed='SUCCEEDED' in state['runs'].values(),
                runs=[
                    dict(run_id=uid, state=value)
                    for uid, value in sorted(state['runs'].items())
                ],
                denial_observed=state['denial']))
    return dict(outcome=outcome, scope=scope, cases=reports)


def load(path):
    with open(path, 'rb') as stream:
        data = stream.read(16 * 1024 * 1024 + 1)
    if len(data) > 16 * 1024 * 1024:
        raise ValueError('input_too_large')
    value = json.loads(data)
    if not isinstance(value, dict):
        raise ValueError('invalid_input')
    return value


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for arg in ('context', 'namespace', 'kfp-endpoint', 'kfp-token-file',
                'expectations', 'prediction-report', 'not-before'):
        parser.add_argument('--' + arg, required=True)
    parser.add_argument('--kfp-ca-file')
    parser.add_argument('--timeout-seconds', type=int, default=60)
    parser.add_argument(
        '--require-run-success',
        action='store_true',
        help='Require SUCCEEDED for every positive case; reject failed, canceled or skipped runs.'
    )
    args = parser.parse_args()
    try:
        if not 30 <= args.timeout_seconds <= 600:
            raise ValueError('invalid_timeout')
        cases, start = validate(
            load(args.expectations), load(args.prediction_report),
            args.namespace)
        if args.require_run_success:
            cases = [
                dict(case, expected_outcome='run_succeeded')
                if case['expected_outcome'] == 'run_created' else case
                for case in cases
            ]
        start = activation_start(start, args.not_before)
        client = Client(args.kfp_endpoint, args.kfp_token_file,
                        args.kfp_ca_file)
        report = observe(client, args.context, args.namespace, cases, start,
                         args.timeout_seconds)
        report['observation_start'] = start.isoformat()
        print(json.dumps(report, sort_keys=True))
        return 0 if report['outcome'] == 'passed' else 1
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        print(
            json.dumps({
                'outcome': 'inconclusive',
                'reason': 'invalid_or_incomplete_evidence'
            }))
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
