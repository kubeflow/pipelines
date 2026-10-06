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
import copy
import json
import unittest
from unittest import mock

from prepare_schedule_transition import cases_for_transition
from prepare_schedule_transition import grant_patch
from prepare_schedule_transition import PREVIOUS
from prepare_schedule_transition import review_use
from prepare_schedule_transition import wait_for_use
from provision_live_schedules import fixture_rbac
from provision_live_schedules import NAMESPACE


class TransitionTests(unittest.TestCase):

    def fixture(self, phase):
        records = [
            dict(
                scenario=s, schedule_uid=s, schedule_name=s, service_account=a)
            for s, a in zip(('default', 'scoped',
                             'denied'), ('pipeline-runner', 'readiness-granted',
                                         'readiness-denied'))
        ]
        previous = dict(
            mode=PREVIOUS[phase],
            scope='fixture_run_completion',
            outcome='passed',
            all_expected_runs_succeeded=True,
            namespace=NAMESPACE,
            cases=copy.deepcopy(records))
        for case in previous['cases']:
            blocked = (case['scenario'] == 'denied' and phase
                       != 'audit-enforce') or (case['scenario'] == 'scoped' and
                                               phase == 'restored')
            case['runs'] = [] if blocked else [
                dict(run_id='old', state='SUCCEEDED')
            ]
        return dict(
            enabled=False, prepared=True, recreated=True,
            schedules=records), previous

    def test_each_transition_requires_previous_success_and_retains_positive_control(
            self):
        for phase in PREVIOUS:
            fixture, previous = self.fixture(phase)
            cases = cases_for_transition(fixture, previous, phase)['cases']
            self.assertEqual(cases[0]['expected_outcome'], 'run_created')
            self.assertEqual(cases[1]['expected_outcome'],
                             'blocked' if phase == 'revoked' else 'run_created')
            self.assertEqual(cases[2]['expected_outcome'], 'blocked')
            previous['cases'][0]['runs'] = []
            with self.assertRaises(ValueError):
                cases_for_transition(fixture, previous, phase)

    def test_inflight_failed_wrong_identity_and_stale_phase_fail_closed(self):
        for change in ('running', 'failed', 'identity', 'phase', 'enabled',
                       'extra_run'):
            fixture, previous = self.fixture('revoked')
            if change in ('running', 'failed'):
                previous['cases'][0]['runs'][0]['state'] = change.upper()
            elif change == 'identity':
                previous['cases'][0]['schedule_uid'] = 'other'
            elif change == 'phase':
                previous['mode'] = 'enforce'
            elif change == 'enabled':
                fixture['enabled'] = True
            else:
                previous['cases'][2]['runs'] = [dict(state='SUCCEEDED')]
            with self.subTest(change=change), self.assertRaises(ValueError):
                cases_for_transition(fixture, previous, 'revoked')

    def test_revoke_and_restore_change_only_the_scoped_controller_grant(self):
        role = next(
            r for r in fixture_rbac([]) if r['kind'] == 'Role' and
            r['metadata']['name'] == 'fixture-controller')
        role['metadata']['resourceVersion'] = '123'
        patch = grant_patch(role, False)
        self.assertEqual(
            patch[0],
            dict(op='test', path='/metadata/resourceVersion', value='123'))
        self.assertEqual(patch[1]['value'], role['rules'])
        original = copy.deepcopy(role['rules'])
        role['rules'] = patch[2]['value']
        self.assertEqual(len(role['rules']), 1)
        self.assertEqual(grant_patch(role, True)[2]['value'], original)
        role['rules'].append(dict(verbs=['*']))
        with self.assertRaises(ValueError):
            grant_patch(role, True)

    def test_live_authorization_is_confirmed_before_cutover(self):
        with mock.patch(
                'prepare_schedule_transition.review_use',
                side_effect=[ValueError(), None]) as review, mock.patch(
                    'prepare_schedule_transition.time.sleep') as sleep:
            wait_for_use('context', 'controller', False)
            self.assertEqual(review.call_count, 2)
            sleep.assert_called_once_with(2)
        with mock.patch(
                'prepare_schedule_transition.review_use',
                side_effect=ValueError()), mock.patch(
                    'prepare_schedule_transition.time.sleep') as sleep:
            with self.assertRaises(ValueError):
                wait_for_use('context', 'controller', False)
            self.assertEqual(sleep.call_count, 29)

    def test_all_namespace_collection_is_explicit_and_exclusive(self):
        from kubectl_inventory import kubectl_get
        for all_namespaces in (False, True):
            with mock.patch(
                    'kubectl_inventory.subprocess.Popen',
                    side_effect=OSError()) as popen:
                kubectl_get(
                    'context', None, 'roles', all_namespaces=all_namespaces)
            self.assertEqual('--all-namespaces' in popen.call_args.args[0],
                             all_namespaces)
        with self.assertRaises(ValueError):
            kubectl_get('context', 'namespace', 'roles', all_namespaces=True)

    def test_live_sar_rejects_opposite_unknown_or_evaluation_error(self):
        for expected in (False, True):
            for status in ({
                    'allowed': expected
            }, {
                    'allowed': not expected
            }, {}, {
                    'allowed': expected,
                    'evaluationError': 'failed'
            }):
                with mock.patch(
                        'prepare_schedule_transition.subprocess.run',
                        return_value=mock.Mock(
                            stdout=json.dumps({'status': status}))) as run:
                    if status == {'allowed': expected}:
                        review_use('test', 'controller', expected)
                    else:
                        with self.assertRaises(ValueError):
                            review_use('test', 'controller', expected)
                    attrs = json.loads(run.call_args.kwargs['input']
                                      )['spec']['resourceAttributes']
                    self.assertEqual(attrs['namespace'], NAMESPACE)
                    self.assertEqual(attrs['name'], 'readiness-granted')
                    self.assertEqual(attrs['verb'], 'use')


if __name__ == '__main__':
    unittest.main()
