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

import contextlib
import copy
import io
import json
from pathlib import Path
import unittest

import readiness
import schedule_policy


def evidence():
    return dict(
        policy_contract=schedule_policy.CONTRACT,
        target_revision='a' * 40,
        multi_user=True,
        mode='enforce',
        default_service_account='pipeline-runner',
        controller_user='system:serviceaccount:kubeflow:controller',
        allowed_service_accounts=['custom'],
        rbac_complete=True,
        rbac_only=True,
        rbac=[],
        recurring_runs=[
            dict(
                recurring_run_id='id',
                experiment_id='exp',
                service_account='custom')
        ],
        experiments=[dict(experiment_id='exp', namespace='team-a')])


def schedule():
    return dict(
        kind='ScheduledWorkflow',
        metadata=dict(name='nightly', namespace='team-a', uid='id'),
        spec=dict(enabled=False, serviceAccount='untrusted-cr-account'))


class SchedulePolicyTest(unittest.TestCase):

    def test_cli_policy_example(self):
        examples = Path(__file__).parent / 'examples'
        with contextlib.redirect_stdout(io.StringIO()) as output:
            code = readiness.main([
                '--inventory',
                str(examples / 'schedules.json'), '--schedule-policy',
                str(examples / 'target-policy.json'), '--include-schedules',
                '--system-namespace', 'kubeflow', '--namespace', 'team-a',
                '--source-version', '2.17.2', '--format', 'json'
            ])
        self.assertEqual(code, 2)
        result = json.loads(output.getvalue())
        self.assertTrue(
            any(f['status'] == 'policy_rejection' for f in result['findings']))
        self.assertIn('a' * 40, readiness.markdown(result))
        self.assertNotIn('example-controller', output.getvalue())

    def test_persisted_account_not_mutable_cr(self):
        result = schedule_policy.assess(schedule(), evidence())
        self.assertEqual(result[0], 'policy_rejection')
        self.assertIn('team-a/custom', result[1])
        self.assertNotIn('untrusted-cr-account', result[1])

    def test_template_default_requires_empty_target_patch(self):
        bundle = evidence()
        bundle['recurring_runs'][0]['service_account'] = None
        bundle['recurring_runs'][0]['_readiness_v2_default'] = True
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'unknown')
        bundle['compiled_pipeline_spec_patch'] = {'serviceAccountName': 'other'}
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'unknown')
        bundle['compiled_pipeline_spec_patch'] = {}
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'no_issue_detected')

    def test_default_exemption(self):
        bundle = evidence()
        bundle['recurring_runs'][0]['service_account'] = 'pipeline-runner'
        bundle['allowed_service_accounts'] = []
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'no_issue_detected')

    def test_empty_and_wildcard_allowlists_reject_custom(self):
        for allow in ([], ['*']):
            bundle = evidence()
            bundle['allowed_service_accounts'] = allow
            bundle['rbac_complete'] = False
            self.assertEqual(
                schedule_policy.assess(schedule(), bundle)[0],
                'policy_rejection')
        bundle['mode'] = 'audit'
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'operational_impact')

    def test_incomplete_rbac_and_identity_are_unknown(self):
        mutations = [
            lambda b: b.update(rbac_complete=False),
            lambda b: b.update(rbac_only=False),
            lambda b: b.update(recurring_runs=[]),
            lambda b: b['recurring_runs'].append(
                copy.deepcopy(b['recurring_runs'][0])),
            lambda b: b['experiments'].append(
                copy.deepcopy(b['experiments'][0])),
            lambda b: b['experiments'][0].update(namespace='other'),
            lambda b: b['recurring_runs'][0].update(service_account=''),
            lambda b: b['recurring_runs'][0].update(namespace='other'),
        ]
        for mutate in mutations:
            bundle = evidence()
            mutate(bundle)
            self.assertEqual(
                schedule_policy.assess(schedule(), bundle)[0], 'unknown')
        obj = schedule()
        obj['metadata']['uid'] = 'different'
        self.assertEqual(schedule_policy.assess(obj, evidence())[0], 'unknown')
        obj = schedule()
        obj['spec']['workflow'] = {'spec': 'secret pipeline'}
        self.assertEqual(schedule_policy.assess(obj, evidence())[0], 'unknown')

    def test_no_group_expansion(self):
        bundle = evidence()
        bundle['rbac'] = [
            dict(
                kind='ClusterRole',
                metadata=dict(name='use'),
                rules=[
                    dict(
                        apiGroups=[''],
                        resources=['serviceaccounts'],
                        verbs=['use'])
                ]),
            dict(
                kind='ClusterRoleBinding',
                metadata=dict(name='use'),
                roleRef=dict(
                    apiGroup='rbac.authorization.k8s.io',
                    kind='ClusterRole',
                    name='use'),
                subjects=[
                    dict(
                        kind='Group',
                        apiGroup='rbac.authorization.k8s.io',
                        name='system:authenticated')
                ])
        ]
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'policy_rejection')
        bundle['rbac'][1]['subjects'] = [
            dict(
                kind='User',
                apiGroup='rbac.authorization.k8s.io',
                name=bundle['controller_user'])
        ]
        self.assertEqual(
            schedule_policy.assess(schedule(), bundle)[0], 'no_issue_detected')

    def test_contract_and_report_remain_incomplete(self):
        bundle = evidence()
        schedule_policy.validate(bundle)
        result = readiness.analyze({'items': [schedule()]},
                                   'kubeflow', ['team-a'],
                                   'ui',
                                   'cache',
                                   '2.17.2',
                                   include_schedules=True,
                                   policy=bundle)
        self.assertEqual(result['assessment'], 'incomplete')
        self.assertEqual(result['target']['schedule_policy_revision'], 'a' * 40)
        self.assertTrue(
            any(f['status'] == 'policy_rejection' for f in result['findings']))
        for key, value in [('policy_contract', 'unknown'),
                           ('target_revision', 'master'),
                           ('multi_user', 'false'),
                           ('allowed_service_accounts', '*')]:
            broken = dict(bundle, **{key: value})
            with self.assertRaises((ValueError, TypeError)):
                schedule_policy.validate(broken)


if __name__ == '__main__':
    unittest.main()
