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
import json
import unittest
from unittest.mock import Mock
from unittest.mock import patch

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

    def test_final_collection_covers_sleep_and_inflight_request(self):
        control = dict(
            self.case,
            schedule_uid='control',
            schedule_name='control',
            expected_prediction='no_issue_detected',
            expected_outcome='run_created')
        control_run = dict(
            self.run,
            recurring_run_id='control',
            created_at='2026-01-01T00:00:01Z')
        late_run_record = dict(self.run, created_at='2026-01-01T00:00:25Z')
        event = dict(self.event, lastTimestamp='2026-01-01T00:00:01Z')
        for slow_request in (False, True):
            for late_run in (False, True):
                with self.subTest(slow_request=slow_request, late_run=late_run):
                    clock = Mock(return_value=0)
                    client = Mock()
                    blocked_queries = []

                    def get(path, params=None):
                        if '/experiments/' in path:
                            return {'experiment_id': 'exp', 'namespace': 'team'}
                        if 'control' in params['filter']:
                            if slow_request:
                                clock.return_value += 11
                            return {'runs': [control_run]}
                        blocked_queries.append(clock.return_value)
                        if late_run and clock.return_value >= 25:
                            return {'runs': [late_run_record]}
                        return {'runs': []}

                    def sleep(seconds):
                        clock.return_value += seconds

                    client.get.side_effect = get
                    report = live.observe(
                        client,
                        'ctx',
                        'team', [self.case, control],
                        self.start,
                        30,
                        get=lambda *_: [event],
                        clock=clock,
                        sleep=sleep)
                    self.assertEqual(report['outcome'],
                                     'failed' if late_run else 'passed')
                    self.assertGreaterEqual(blocked_queries[-1], 30)
                    self.assertEqual(sum(t >= 30 for t in blocked_queries), 1)

    def test_final_collection_failure_is_not_ignored(self):
        case = dict(
            self.case,
            expected_prediction='no_issue_detected',
            expected_outcome='run_created')
        client = Mock()
        client.get.side_effect = lambda path, *args: ({
            'experiment_id': 'exp',
            'namespace': 'team'
        } if '/experiments/' in path else {
            'runs': [self.run]
        })
        events = Mock(side_effect=[[], CollectionError('request_failed')])
        with self.assertRaises(CollectionError):
            live.observe(
                client,
                'ctx',
                'team', [case],
                self.start,
                30,
                get=events,
                clock=Mock(side_effect=[0, 0, 30, 30]),
                sleep=lambda _: None)

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

    def completion_report(self,
                          states,
                          *,
                          blocked=False,
                          outcome='run_succeeded'):
        control = dict(
            self.case,
            schedule_uid='control',
            schedule_name='control',
            expected_prediction='no_issue_detected',
            expected_outcome=outcome)
        cases = [self.case, control] if blocked else [control]
        elapsed = [0]
        client = Mock()

        def get(path, params=None):
            if '/experiments/' in path:
                return {'experiment_id': 'exp', 'namespace': 'team'}
            if 'control' not in params['filter']:
                return {'runs': []}
            state = states[min(elapsed[0] // 10, len(states) - 1)]
            return {
                'runs': [
                    dict(
                        self.run,
                        recurring_run_id='control',
                        state=state,
                        pipeline_spec={'private': 'payload'},
                        error_message='secret')
                ]
            }

        def sleep(seconds):
            elapsed[0] += seconds

        client.get.side_effect = get
        report = live.observe(
            client,
            'ctx',
            'team',
            cases,
            self.start,
            30,
            get=lambda *_: [self.event],
            clock=lambda: elapsed[0],
            sleep=sleep)
        return report, elapsed[0]

    def test_completion_waits_for_success_in_final_interval(self):
        report, elapsed = self.completion_report(
            ['PENDING', 'RUNNING', 'RUNNING', 'SUCCEEDED'])
        self.assertEqual(report['outcome'], 'passed')
        self.assertEqual(report['scope'], 'schedule_run_completion')
        self.assertEqual(elapsed, 30)
        self.assertTrue(report['cases'][0]['success_observed'])
        self.assertEqual(report['cases'][0]['runs'], [{
            'run_id': 'new',
            'state': 'SUCCEEDED'
        }])
        self.assertNotIn('private', json.dumps(report))
        self.assertNotIn('secret', json.dumps(report))

    def test_unsuccessful_terminal_runs_fail_completion(self):
        for state in ('FAILED', 'CANCELED', 'SKIPPED'):
            with self.subTest(state=state):
                report, _ = self.completion_report(['RUNNING', state])
                self.assertEqual(report['outcome'], 'failed')
                self.assertFalse(report['cases'][0]['success_observed'])

    def test_unfinished_or_unknown_runs_cannot_pass_completion(self):
        for state in ('PENDING', 'RUNNING', 'CANCELING', 'PAUSED', None, {
                'private': 'payload'
        }, 'private-unrecognized-state'):
            with self.subTest(state=state):
                report, elapsed = self.completion_report([state])
                self.assertEqual(report['outcome'], 'inconclusive')
                self.assertEqual(elapsed, 30)
                self.assertNotIn('private', json.dumps(report))

    def test_creation_only_remains_compatible_with_failed_or_unknown_state(
            self):
        for state in ('FAILED', None):
            with self.subTest(state=state):
                report, _ = self.completion_report([state],
                                                   outcome='run_created')
                self.assertEqual(report['outcome'], 'passed')
                self.assertEqual(report['scope'], 'schedule_run_creation_only')

    def test_denial_requires_successful_control_in_completion_mode(self):
        for state, outcome in [('RUNNING', 'inconclusive'),
                               ('FAILED', 'failed'), ('SUCCEEDED', 'passed')]:
            with self.subTest(state=state):
                report, elapsed = self.completion_report([state], blocked=True)
                self.assertEqual(report['outcome'], outcome)
                if outcome == 'passed':
                    self.assertEqual(elapsed, 30)
                self.assertTrue(report['cases'][0]['denial_observed'])

    def test_success_does_not_hide_a_later_failure(self):
        report, elapsed = self.completion_report(
            ['SUCCEEDED', 'SUCCEEDED', 'SUCCEEDED', 'FAILED'])
        self.assertEqual(report['outcome'], 'failed')
        self.assertEqual(elapsed, 30)

    def test_explicit_success_expectation_validates(self):
        case = dict(
            self.case,
            expected_outcome='run_succeeded',
            expected_prediction='no_issue_detected')
        bundle = dict(
            namespace='team',
            observation_start='2026-01-01T00:00:00Z',
            cases=[case])
        report = {
            'findings': [
                dict(
                    rule='schedule.targetMainAccount',
                    resource='ScheduledWorkflow/team/schedule',
                    status='no_issue_detected')
            ]
        }
        self.assertEqual(live.validate(bundle, report, 'team')[0], [case])

    def test_cli_failures_remain_inconclusive_and_sanitized(self):
        args = [
            'check', '--context', 'ctx', '--namespace', 'team',
            '--kfp-endpoint', 'http://127.0.0.1', '--kfp-token-file', 'token',
            '--expectations', 'cases', '--prediction-report', 'report',
            '--not-before', '2026-01-01T00:00:00Z'
        ]
        failures = [
            (CollectionError('request_failed'), 'request_failed'),
            (CollectionError('run_account_mismatch'), 'run_account_mismatch'),
            (CollectionError('collection_timed_out'), 'collection_timed_out'),
            (ValueError('prediction_mismatch'), 'prediction_mismatch'),
            (ValueError('invalid_activation_start'),
             'invalid_activation_start'),
            (CollectionError('private response body'), None),
            (ValueError('private timestamp or payload'), None),
            (ValueError({'private': 'payload'}), None),
            (OSError('private token path'), None),
            (KeyError('private field'), None),
            (TypeError('private value'), None),
            (AttributeError('private object'), None),
        ]
        for error, reason in failures:
            with self.subTest(error=type(error), reason=reason):
                with patch('sys.argv', args), patch.object(
                        live, 'load',
                        side_effect=error), patch('builtins.print') as output:
                    self.assertEqual(live.main(), 1)
                self.assertEqual(
                    json.loads(output.call_args.args[0]), {
                        'outcome': 'inconclusive',
                        'reason': reason or 'invalid_or_incomplete_evidence'
                    })

    def test_cli_upgrades_positive_expectations_only(self):
        case = dict(
            self.case,
            schedule_uid='control',
            expected_outcome='run_created',
            expected_prediction='no_issue_detected')
        args = [
            'check', '--context', 'ctx', '--namespace', 'team',
            '--kfp-endpoint', 'http://127.0.0.1', '--kfp-token-file', 'token',
            '--expectations', 'cases', '--prediction-report', 'report',
            '--not-before', '2026-01-01T00:00:00Z', '--require-run-success'
        ]
        with patch('sys.argv', args), patch.object(live, 'load'), patch.object(
                live, 'validate',
                return_value=([self.case, case], self.start)), patch.object(
                    live, 'Client'), patch.object(
                        live, 'observe',
                        return_value={'outcome': 'passed'
                                     }) as observe, patch('builtins.print'):
            self.assertEqual(live.main(), 0)
        self.assertEqual(
            [c['expected_outcome'] for c in observe.call_args.args[3]],
            ['blocked', 'run_succeeded'])
        self.assertEqual(case['expected_outcome'], 'run_created')


if __name__ == '__main__':
    unittest.main()
