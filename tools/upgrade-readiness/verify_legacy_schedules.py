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
"""Observe missing-state rejection of source fixtures in disposable upgrade
CI."""

import argparse
import json
import time

from kfp_http import Client
from live_schedule_check import activation_start
from live_schedule_check import list_events
from live_schedule_check import runs
from live_schedule_check import timestamp
from live_schedule_check import validate
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object


def rejected(events, case, start):
    for event in events:
        ref = event.get('involvedObject', {})
        if (ref.get('uid') != case['schedule_uid'] or
                ref.get('name') != case['schedule_name'] or
                ref.get('namespace') != NAMESPACE or
                ref.get('kind') != 'ScheduledWorkflow' or
                event.get('reason') != 'Failed' or
                event.get('type') != 'Warning' or
                event.get('source', {}).get('component')
                != 'scheduled-workflow-controller'):
            continue
        uid = event.get('metadata', {}).get('uid')
        count = event.get('count', 1)
        if (not isinstance(uid, str) or type(count) is not int or
                count <= case['baseline_event_counts'].get(uid, 0)):
            continue
        observed = event.get('series', {}).get('lastObservedTime') or event.get(
            'lastTimestamp') or event.get('eventTime')
        if timestamp(observed) < start:
            continue
        message = event.get('message', '')
        signature = (
            'Recurring run ' + case['schedule_uid'] +
            ' has no trusted scheduling state; recreate it through the KFP API')
        if (isinstance(message, str) and
                'code = FailedPrecondition' in message and
                signature in message):
            return True
    return False


def observe(client,
            cases,
            start,
            timeout,
            get=list_events,
            collect=runs,
            clock=time.monotonic,
            sleep=time.sleep,
            client_factory=None):
    deadline = clock() + timeout
    seen = set()
    while True:
        # Collect once more at/after the deadline; never use stale final evidence.
        final = clock() >= deadline
        cycle_client = client_factory() if client_factory else client
        events = get(CONTEXT, NAMESPACE)
        for case in cases:
            if collect(cycle_client, NAMESPACE, case, start):
                return dict(
                    outcome='failed', reason='legacy_schedule_created_run')
            if rejected(events, case, start):
                seen.add(case['schedule_uid'])
        if final:
            break
        sleep(min(5, max(0, deadline - clock())))
    return dict(
        outcome='passed' if len(seen) == len(cases) else 'inconclusive',
        scope='legacy_schedule_migration_rejection',
        namespace=NAMESPACE,
        observation_start=start.isoformat(),
        schedule_uids=sorted(seen))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('context', 'baseline', 'prediction-report', 'not-before',
                 'kfp-endpoint', 'kfp-token-file'):
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--timeout-seconds', type=int, default=180)
    args = parser.parse_args()
    try:
        if args.context != CONTEXT or not 30 <= args.timeout_seconds <= 600:
            raise ValueError('invalid_disposable_fixture_scope')
        cases, baseline_start = validate(
            read_object(args.baseline), read_object(args.prediction_report),
            NAMESPACE)
        if len(cases) != 3:
            raise ValueError('three_source_fixtures_required')
        start = activation_start(baseline_start, args.not_before)
        result = observe(
            None,
            cases,
            start,
            args.timeout_seconds,
            client_factory=lambda: Client(args.kfp_endpoint, args.kfp_token_file
                                         ))
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        result = dict(
            outcome='inconclusive', reason='invalid_or_incomplete_evidence')
    print(json.dumps(result, sort_keys=True))
    return 0 if result['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
