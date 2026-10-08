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
"""Scheduler evidence must retain useful state without raw payloads."""
import copy
import json
import unittest
from unittest.mock import patch

import schedule_diagnostics as diagnostics


class ScheduleDiagnosticsTests(unittest.TestCase):

    def setUp(self):
        self.cases = [dict(schedule_name='schedule', schedule_uid='swf-uid')]
        self.swf = dict(
            kind='ScheduledWorkflow',
            metadata=dict(name='schedule', uid='swf-uid', namespace='ns'),
            spec=dict(enabled=True, maxConcurrency=1, secret='sensitive'),
            status=dict(trigger=dict(lastWorkflowIndex=2)))
        self.wf = dict(
            metadata=dict(
                name='run-name',
                uid='wf-uid',
                namespace='ns',
                labels={
                    'pipeline/runid':
                        'run-id',
                    'scheduledworkflows.kubeflow.org/scheduledWorkflowName':
                        'schedule',
                    'scheduledworkflows.kubeflow.org/workflowIndex':
                        '1'
                },
                ownerReferences=[
                    dict(
                        name='schedule',
                        uid='swf-uid',
                        kind='ScheduledWorkflow',
                        controller=True)
                ]),
            status=dict(phase='Succeeded', message='sensitive'),
            spec=dict(secret='sensitive'))
        self.event = dict(
            involvedObject=dict(uid='swf-uid', name='schedule', namespace='ns'),
            reason='Failed',
            count=2,
            message='PermissionDenied sensitive')

    def collect(self, swf=None, wf=None, event=None):
        data = {
            'scheduledworkflows': [swf or self.swf],
            'workflows': [wf or self.wf],
            'events': [event or self.event]
        }
        with patch.object(
                diagnostics, 'items', side_effect=lambda c, n, r: data[r]):
            return diagnostics.snapshot('context', 'ns', self.cases)

    def test_retains_terminal_workflow_missing_completed_label_and_redacts(
            self):
        result = self.collect()
        self.assertNotIn('sensitive', json.dumps(result))
        case = result['cases'][0]
        self.assertTrue(case['enabled'])
        self.assertEqual(case['last_index'], 2)
        self.assertEqual(case['workflows'][0]['phase'], 'Succeeded')
        self.assertIsNone(case['workflows'][0]['completed'])
        self.assertEqual(case['events'][0]['classification'],
                         'authorization_denied')

    def test_missing_labels_are_retained_as_missing(self):
        wf = copy.deepcopy(self.wf)
        wf['metadata']['labels'] = {}
        row = self.collect(wf=wf)['cases'][0]['workflows'][0]
        self.assertIsNone(row['run_id'])
        self.assertIsNone(row['schedule_label'])
        self.assertIsNone(row['workflow_index'])

    def test_wrong_schedule_uid_fails_closed(self):
        swf = copy.deepcopy(self.swf)
        swf['metadata']['uid'] = 'replacement'
        with self.assertRaises(ValueError):
            self.collect(swf=swf)

    def test_malformed_status_fails_closed(self):
        for value in ('true', {}, 1):
            swf = copy.deepcopy(self.swf)
            swf['spec']['enabled'] = value
            with self.assertRaises(ValueError):
                self.collect(swf=swf)

    def test_wrong_owner_or_namespace_fails_closed(self):
        for field, value in [('namespace', 'other'),
                             ('ownerReferences', [
                                 dict(
                                     uid='swf-uid',
                                     name='wrong',
                                     kind='ScheduledWorkflow',
                                     controller=True)
                             ])]:
            wf = copy.deepcopy(self.wf)
            wf['metadata'][field] = value
            with self.assertRaises(ValueError):
                self.collect(wf=wf)

    def test_unknown_event_reason_is_not_emitted(self):
        event = copy.deepcopy(self.event)
        event['reason'] = 'sensitive'
        self.assertNotIn('sensitive', json.dumps(self.collect(event=event)))

    def test_empty_or_duplicate_schedule_fails_closed(self):
        for schedules in ([], [self.swf, self.swf]):
            with self.assertRaises(ValueError):
                diagnostics.schedule_evidence(self.cases, schedules, 'ns')

    def test_enable_fence_waits_and_times_out_closed(self):
        disabled = copy.deepcopy(self.swf)
        disabled['spec']['enabled'] = False
        with patch.object(
                diagnostics, 'items',
                side_effect=[[disabled],
                             [self.swf]]), patch.object(diagnostics.time,
                                                        'sleep') as sleep:
            diagnostics.wait_enabled('ctx', 'ns', self.cases)
            sleep.assert_called_once_with(2)
        with patch.object(
                diagnostics, 'items', return_value=[disabled]), patch.object(
                    diagnostics.time, 'monotonic', side_effect=[0, 61]):
            with self.assertRaisesRegex(ValueError, 'enable_not_observed'):
                diagnostics.wait_enabled('ctx', 'ns', self.cases)


if __name__ == '__main__':
    unittest.main()
