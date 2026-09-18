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
"""Capture read-only baselines for explicitly selected isolated test
schedules."""

import argparse
from datetime import datetime
from datetime import timezone
import json

from kfp_http import Client
from kfp_http import CollectionError
from live_schedule_check import list_events
from live_schedule_check import list_runs
from live_schedule_check import validate
from readiness import kubectl_get
from readiness import read_json


def capture(client, context, namespace, definitions, report):
    start = datetime.now(timezone.utc).isoformat()
    # Validate case/prediction association before any API request.
    cases = []
    for definition in definitions.get('cases', []):
        case = dict(definition)
        case['baseline_run_ids'] = []
        case['baseline_event_counts'] = {}
        cases.append(case)
    result = dict(namespace=namespace, observation_start=start, cases=cases)
    validate(result, report, namespace)
    data, error = kubectl_get(context, namespace,
                              'scheduledworkflows.kubeflow.org')
    if error or not isinstance(data, dict) or not isinstance(
            data.get('items'), list):
        raise CollectionError('schedule_collection_failed')
    for case in cases:
        matches = [
            s for s in data['items'] if isinstance(s, dict) and
            s.get('metadata', {}).get('uid') == case['schedule_uid'] and
            s.get('metadata', {}).get('name') == case['schedule_name'] and
            s.get('metadata', {}).get('namespace') == namespace
        ]
        if len(matches) != 1:
            raise CollectionError('schedule_identity_mismatch')
    events = list_events(context, namespace)
    for case in cases:
        records = list_runs(client, namespace, case['schedule_uid'])
        case['baseline_run_ids'] = [
            r.get('run_id', r.get('runId')) for r in records
        ]
        for event in events:
            reference = event.get('involvedObject', {})
            if reference.get('uid') != case['schedule_uid']:
                continue
            uid = event.get('metadata', {}).get('uid')
            count = event.get('count', 1)
            if not isinstance(
                    uid, str) or not uid or type(count) is not int or count < 0:
                raise CollectionError('invalid_event_baseline')
            case['baseline_event_counts'][uid] = count
    validate(result, report, namespace)
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('context', 'namespace', 'kfp-endpoint', 'kfp-token-file',
                 'cases', 'prediction-report'):
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--kfp-ca-file')
    args = parser.parse_args()
    try:
        client = Client(args.kfp_endpoint, args.kfp_token_file,
                        args.kfp_ca_file)
        result = capture(client, args.context, args.namespace,
                         read_json(args.cases),
                         read_json(args.prediction_report))
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        parser.exit(
            1,
            'Unable to capture complete schedule baselines; check scope, evidence and collection access.\n'
        )
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
