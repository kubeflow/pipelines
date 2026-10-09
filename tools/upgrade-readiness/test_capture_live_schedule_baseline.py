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
import json
import unittest
from unittest import mock

import capture_live_schedule_baseline as capture


def inputs():
    definitions = {
        'cases': [{
            'schedule_uid': 'uid',
            'schedule_name': 'nightly',
            'service_account': 'runner',
            'scenario': 'default',
            'expected_prediction': 'no_issue_detected',
            'expected_outcome': 'run_created'
        }]
    }
    report = {
        'findings': [{
            'rule': 'schedule.targetMainAccount',
            'resource': 'ScheduledWorkflow/team/nightly',
            'status': 'no_issue_detected'
        }]
    }
    return definitions, report


class BaselineTest(unittest.TestCase):

    def test_capture_keeps_only_baseline_identifiers(self):
        definitions, report = inputs()
        schedules = {
            'items': [{
                'metadata': {
                    'uid': 'uid',
                    'name': 'nightly',
                    'namespace': 'team'
                }
            }]
        }
        events = [{
            'metadata': {
                'uid': 'event'
            },
            'involvedObject': {
                'uid': 'uid'
            },
            'count': 2,
            'message': 'private message'
        }]
        with mock.patch.object(
                capture, 'kubectl_get',
                return_value=(schedules, None)), mock.patch.object(
                    capture, 'list_events',
                    return_value=events), mock.patch.object(
                        capture,
                        'list_runs',
                        return_value=[{
                            'run_id': 'run',
                            'parameters': 'private inputs'
                        }]):
            result = capture.capture(mock.Mock(), 'context', 'team',
                                     definitions, report)
        self.assertEqual(result['cases'][0]['baseline_run_ids'], ['run'])
        self.assertEqual(result['cases'][0]['baseline_event_counts'],
                         {'event': 2})
        self.assertNotIn('private', json.dumps(result))
        self.assertNotIn('baseline_run_ids', definitions['cases'][0])

    def test_identity_and_prediction_mismatch_fail(self):
        definitions, report = inputs()
        with mock.patch.object(
                capture, 'kubectl_get', return_value=({
                    'items': []
                }, None)):
            with self.assertRaisesRegex(ValueError,
                                        'schedule_identity_mismatch'):
                capture.capture(mock.Mock(), 'context', 'team', definitions,
                                report)
        report['findings'][0]['status'] = 'unknown'
        with mock.patch.object(capture, 'kubectl_get') as get:
            with self.assertRaises(ValueError):
                capture.capture(mock.Mock(), 'context', 'team', definitions,
                                report)
            get.assert_not_called()


if __name__ == '__main__':
    unittest.main()
