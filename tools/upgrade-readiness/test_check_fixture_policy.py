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
"""Target fixture checks must not be presented as source predictions."""

import unittest
from unittest import mock

from build_live_policy import build_policy
import check_fixture_policy as check


class FixturePolicyTests(unittest.TestCase):

    def setUp(self):
        self.fixture = dict(
            context=check.CONTEXT,
            namespace=check.NAMESPACE,
            recreated=True,
            prepared=True,
            enabled=False,
            schedules=[
                dict(
                    scenario=str(i),
                    schedule_uid='uid-' + str(i),
                    schedule_name='schedule-' + str(i)) for i in range(3)
            ])
        self.policy = build_policy(
            dict(items=[]), dict(items=[]), 'a' * 40, 'enforce')
        self.records = dict(
            recurring_runs=[
                dict(
                    recurring_run_id='uid-' + str(i),
                    experiment_id='experiment',
                    service_account='pipeline-runner') for i in range(3)
            ],
            experiments=[
                dict(experiment_id='experiment', namespace=check.NAMESPACE)
            ])
        self.schedules = dict(items=[
            dict(
                metadata=dict(
                    uid=c['schedule_uid'],
                    name=c['schedule_name'],
                    namespace=check.NAMESPACE),
                spec={}) for c in self.fixture['schedules']
        ])
        self.coverage = dict(list_completed_namespaces=[check.NAMESPACE])

    def test_collects_only_fixture_namespace_and_marks_target_scope(self):
        with mock.patch.object(
                check, 'collect',
                return_value=(self.records, [],
                              self.coverage)) as collect, mock.patch.object(
                                  check,
                                  'kubectl_get',
                                  return_value=(self.schedules, None)):
            result = check.assess(None, check.CONTEXT, self.fixture,
                                  self.policy)
        collect.assert_called_once_with(None, [check.NAMESPACE])
        self.assertEqual(result['scope'], 'post_recreation_target_policy_check')
        self.assertFalse(result['pre_upgrade_prediction_validated'])
        self.assertEqual([f['status'] for f in result['findings']],
                         ['no_issue_detected'] * 3)

    def test_legacy_embedded_workflow_remains_unresolved(self):
        self.schedules['items'][0]['spec'] = dict(
            workflow=dict(spec={'serviceAccountName': 'pipeline-runner'}))
        with mock.patch.object(
                check, 'collect',
                return_value=(self.records, [],
                              self.coverage)), mock.patch.object(
                                  check,
                                  'kubectl_get',
                                  return_value=(self.schedules, None)):
            with self.assertRaisesRegex(ValueError,
                                        'fixture_policy_unresolved'):
                check.assess(None, check.CONTEXT, self.fixture, self.policy)

    def test_recreated_target_identity_mismatch_fails(self):
        for field in ('uid', 'name', 'namespace'):
            original = self.schedules['items'][0]['metadata'][field]
            self.schedules['items'][0]['metadata'][field] = 'wrong'
            with mock.patch.object(
                    check,
                    'collect',
                    return_value=(self.records, [],
                                  self.coverage)), mock.patch.object(
                                      check,
                                      'kubectl_get',
                                      return_value=(self.schedules, None)):
                with self.assertRaisesRegex(
                        ValueError, 'fixture_schedule_identity_mismatch'):
                    check.assess(None, check.CONTEXT, self.fixture, self.policy)
            self.schedules['items'][0]['metadata'][field] = original

    def test_failed_collection_or_unrecreated_fixture_cannot_pass(self):
        with mock.patch.object(
                check,
                'collect',
                return_value=(self.records, ['failure'], self.coverage)):
            with self.assertRaisesRegex(
                    ValueError, 'complete_target_fixture_evidence_required'):
                check.assess(None, check.CONTEXT, self.fixture, self.policy)
        self.fixture['recreated'] = False
        with mock.patch.object(check, 'collect') as collect:
            with self.assertRaisesRegex(ValueError,
                                        'disabled_recreated_fixtures_required'):
                check.assess(None, check.CONTEXT, self.fixture, self.policy)
            collect.assert_not_called()


if __name__ == '__main__':
    unittest.main()
