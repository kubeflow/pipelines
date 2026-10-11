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
"""Source compatibility never weakens target account evidence."""

import copy
import unittest
from unittest import mock

from live_schedule_check import run_evidence
import source_schedule_check as source


class SourceTests(unittest.TestCase):

    def setUp(self):
        self.start = source.timestamp('2026-10-05T00:00:00Z')
        self.case = dict(
            schedule_uid='schedule-uid',
            schedule_name='schedule',
            service_account='runner',
            baseline_run_ids=[])
        self.run = dict(
            run_id='run',
            display_name='execution',
            created_at='2026-10-05T00:00:01Z',
            state='SUCCEEDED')
        self.workflow = dict(
            metadata=dict(
                name='execution',
                namespace=source.NAMESPACE,
                uid='workflow-uid',
                labels={'pipeline/runid': 'run'},
                creationTimestamp='2026-10-05T00:00:01Z',
                ownerReferences=[
                    dict(
                        uid='schedule-uid',
                        name='schedule',
                        kind='ScheduledWorkflow',
                        controller=True)
                ]),
            spec=dict(serviceAccountName='runner'))

    def collect(self, workflow):
        with mock.patch.object(source, 'list_runs', return_value=[self.run]):
            return source.source_run_evidence(
                None,
                source.NAMESPACE,
                self.case,
                self.start,
                get=lambda *args: (dict(items=[workflow]), None))

    def test_missing_api_account_requires_correlated_workflow(self):
        result = self.collect(self.workflow)
        self.assertEqual(result, [
            dict(run_id='run', workflow_uid='workflow-uid', state='SUCCEEDED')
        ])
        with mock.patch(
                'live_schedule_check.list_runs', return_value=[self.run]):
            with self.assertRaisesRegex(source.CollectionError,
                                        'run_account_mismatch'):
                run_evidence(None, source.NAMESPACE, self.case, self.start)

    def test_rejects_wrong_account_owner_namespace_run_or_name(self):
        for change in (lambda w: w['spec'].update(serviceAccountName='other'),
                       lambda w: w['metadata']['ownerReferences'][0].update(
                           uid='other'),
                       lambda w: w['metadata'].update(namespace='other'),
                       lambda w: w['metadata'].update(name='other'),
                       lambda w: w['metadata']['labels'].update(
                           {'pipeline/runid': 'other'}), lambda w: w['metadata']
                       .update(creationTimestamp='2026-10-04T00:00:00Z')):
            workflow = copy.deepcopy(self.workflow)
            change(workflow)
            with self.assertRaises(source.CollectionError):
                self.collect(workflow)

    def test_run_snapshot_precedes_workflow_snapshot(self):
        order = []

        def records(*args):
            order.append('runs')
            return [self.run]

        def workflows(*args):
            order.append('workflows')
            self.assertEqual(order, ['runs', 'workflows'])
            return dict(items=[self.workflow]), None

        with mock.patch.object(source, 'list_runs', side_effect=records):
            result = source.source_run_evidence(
                None, source.NAMESPACE, self.case, self.start, get=workflows)
        self.assertEqual(result[0]['run_id'], 'run')

    def test_wrong_api_account_cannot_be_overridden(self):
        self.run['service_account'] = 'other'
        with self.assertRaisesRegex(source.CollectionError,
                                    'source_workflow_account_mismatch'):
            self.collect(self.workflow)

    def test_source_observation_retains_run_ids_across_poll_cycles(self):
        cases = [
            dict(self.case, scenario=str(i), schedule_uid=str(i))
            for i in range(3)
        ]
        fixture = dict(
            context=source.CONTEXT,
            namespace=source.NAMESPACE,
            activation_start='2026-10-05T00:00:00Z',
            schedules=cases)
        records = [[dict(run_id='early')], [], [], [dict(run_id='late')],
                   [dict(run_id='second')], [dict(run_id='third')]]
        args = [
            'source', '--state-dir', '/fixture', '--endpoint',
            'http://localhost', '--token-file', '/token', '--output', '/report'
        ]
        with mock.patch('sys.argv', args), mock.patch.object(
                source, 'read_object', return_value=fixture), mock.patch.object(
                    source, 'write_object') as write, mock.patch.object(
                        source, 'Client'), mock.patch.object(
                            source.time, 'sleep'), mock.patch.object(
                                source.time, 'monotonic',
                                side_effect=[0, 0, 1]), mock.patch.object(
                                    source,
                                    'source_run_evidence',
                                    side_effect=records):
            self.assertEqual(source.main(), 0)
        report = write.call_args.args[1]
        self.assertEqual({run['run_id'] for run in report['cases'][0]['runs']},
                         {'early', 'late'})

    def test_diagnostics_omit_raw_specs_messages_and_logs(self):
        result = source.diagnostics(get=lambda *args: (dict(items=[
            dict(
                metadata=dict(name='private-name'),
                spec=dict(secret='private-secret'),
                status=dict(phase='Failed', message='private-error'))
        ]), None))
        self.assertEqual(result['pods']['phases']['Failed'], 1)
        self.assertNotIn('private', str(result))


if __name__ == '__main__':
    unittest.main()
