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
"""Acceptance verifier rejects missing or unrelated live evidence."""

import copy
from datetime import datetime
from datetime import timezone
import unittest
from unittest.mock import Mock

from kfp_http import CollectionError
import live_schedule_check as live


class LiveTests(unittest.TestCase):

    def setUp(self):
        self.start = datetime(2026, 1, 1, tzinfo=timezone.utc)
        self.case = dict(
            schedule_uid='uid',
            schedule_name='schedule',
            scenario='denied',
            service_account='custom',
            expected_prediction='policy_rejection',
            expected_outcome='blocked',
            baseline_run_ids=[],
            baseline_event_counts={})
        self.event = dict(
            metadata=dict(uid='event'),
            involvedObject=dict(
                uid='uid',
                name='schedule',
                namespace='team',
                kind='ScheduledWorkflow'),
            type='Warning',
            reason='Failed',
            source=dict(component='scheduled-workflow-controller'),
            count=1,
            lastTimestamp='2026-01-01T00:01:00Z',
            message='code = PermissionDenied service account authorization error '
            'Namespace:team,Verb:use,Group:,Resource:serviceaccounts,Subresource:,Name:custom,'
        )
        self.run = dict(
            run_id='new',
            recurring_run_id='uid',
            experiment_id='exp',
            service_account='custom',
            created_at='2026-01-01T00:01:00Z')

    def test_activation_boundary_excludes_source_runs(self):
        with self.assertRaises(ValueError):
            live.activation_start(self.start, '2025-12-31T00:00:00Z')
        activation = live.activation_start(self.start, '2026-01-02T00:00:00Z')
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        self.assertEqual(live.runs(client, 'team', self.case, activation), [])

    def test_fresh_exact_denial(self):
        self.assertTrue(
            live.denied([self.event], 'team', self.case, self.start))

    def test_unrelated_or_stale_denials(self):
        for path, value in [
            (('involvedObject', 'uid'), 'other'),
            (('involvedObject', 'namespace'), 'other'),
            (('source', 'component'), 'other'),
            (('lastTimestamp',), '2025-01-01T00:00:00Z'),
            (('message',), 'code = PermissionDenied pipeline access denied'),
            (('message',),
             self.event['message'].replace('Name:custom,',
                                           'Name:custom-other,'))
        ]:
            event = copy.deepcopy(self.event)
            target = event
            for key in path[:-1]:
                target = target[key]
            target[path[-1]] = value
            self.assertFalse(
                live.denied([event], 'team', self.case, self.start))

    def test_baseline_event_must_increment(self):
        self.case['baseline_event_counts'] = {'event': 1}
        self.assertFalse(
            live.denied([self.event], 'team', self.case, self.start))
        self.event['count'] = 2
        self.assertTrue(
            live.denied([self.event], 'team', self.case, self.start))

    def test_run_identity_and_account(self):
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        self.assertEqual(
            live.runs(client, 'team', self.case, self.start), ['new'])
        for key in ('recurring_run_id', 'service_account'):
            changed = dict(self.run, **{key: 'other'})
            client.get.side_effect = lambda path, *args: ({
                'experiment_id': 'exp',
                'namespace': 'team'
            } if '/experiments/' in path else {
                'runs': [changed]
            })
            with self.assertRaises(CollectionError):
                live.runs(client, 'team', self.case, self.start)

    def test_baseline_run_does_not_count(self):
        self.case['baseline_run_ids'] = ['new']
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        self.assertEqual(live.runs(client, 'team', self.case, self.start), [])

    def test_pagination_loop_not_success(self):
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run],
            'next_page_token': 'same'
        })
        with self.assertRaises(CollectionError):
            live.runs(client, 'team', self.case, self.start)

    def test_timeout_is_inconclusive(self):
        client = Mock()
        client.get.return_value = {'runs': []}
        report = live.observe(
            client,
            'ctx',
            'team', [self.case],
            self.start,
            30,
            get=lambda *_: [],
            clock=Mock(side_effect=[0, 0, 30, 30]),
            sleep=lambda _: None)
        self.assertEqual(report['outcome'], 'inconclusive')

    def test_new_run_fails_expected_block(self):
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        report = live.observe(
            client,
            'ctx',
            'team', [self.case],
            self.start,
            30,
            get=lambda *_: [self.event],
            clock=Mock(side_effect=[0, 0]))
        self.assertEqual(report['outcome'], 'failed')
        self.assertNotIn('message', str(report))

    def test_denial_requires_positive_control(self):
        client = Mock()
        client.get.return_value = {'runs': []}
        report = live.observe(
            client,
            'ctx',
            'team', [self.case],
            self.start,
            30,
            get=lambda *_: [self.event],
            clock=Mock(side_effect=[0, 0, 30, 30]),
            sleep=lambda _: None)
        self.assertEqual(report['outcome'], 'inconclusive')

    def test_denial_with_fresh_control_passes(self):
        control = dict(
            self.case,
            schedule_uid='control',
            schedule_name='control',
            expected_prediction='no_issue_detected',
            expected_outcome='run_created')
        client = Mock()

        def get(path, params=None):
            if '/experiments/' in path:
                return {'experiment_id': 'exp', 'namespace': 'team'}
            if 'control' in params['filter']:
                return {'runs': [dict(self.run, recurring_run_id='control')]}
            return {'runs': []}

        client.get.side_effect = get
        report = live.observe(
            client,
            'ctx',
            'team', [self.case, control],
            self.start,
            30,
            get=lambda *_: [self.event],
            clock=Mock(side_effect=[0, 0, 30, 30]),
            sleep=lambda _: None)
        self.assertEqual(report['outcome'], 'passed')

    def test_experiment_namespace_mismatch(self):
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'other'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        with self.assertRaises(CollectionError):
            live.list_runs(client, 'team', 'uid')

    def test_prediction_binding(self):
        control = dict(
            self.case,
            schedule_uid='control',
            schedule_name='control',
            expected_prediction='no_issue_detected',
            expected_outcome='run_created')
        bundle = dict(
            namespace='team',
            observation_start='2026-01-01T00:00:00Z',
            cases=[self.case, control])
        report = {
            'findings': [
                dict(
                    rule='schedule.targetMainAccount',
                    resource='ScheduledWorkflow/team/' + c['schedule_name'],
                    status=c['expected_prediction']) for c in bundle['cases']
            ]
        }
        self.assertEqual(len(live.validate(bundle, report, 'team')[0]), 2)
        report['findings'][0]['status'] = 'unknown'
        with self.assertRaises(ValueError):
            live.validate(bundle, report, 'team')


if __name__ == '__main__':
    unittest.main()
