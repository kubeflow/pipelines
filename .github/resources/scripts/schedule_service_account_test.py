#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Ensure custom schedule authorization is exercised, rather than skipped."""

import json
import unittest
from pathlib import Path
from unittest.mock import patch

import yaml
from prepare_schedule_service_account import ACCOUNT, fixture, main
from verify_migrated_specs import verify

ROOT = Path(__file__).resolve().parents[3]


class ScheduleQualificationTest(unittest.TestCase):
    def test_setup_keeps_enforcement_and_waits_for_rollout(self):
        with (
            patch('sys.argv', ['fixture', '--namespace', 'profile-a']),
            patch(
                'prepare_schedule_service_account.subprocess.check_output',
                return_value='actual-controller',
            ),
            patch('prepare_schedule_service_account.subprocess.run') as run,
        ):
            main()
        calls = run.call_args_list
        self.assertEqual(len(calls), 3)
        document = json.loads(calls[0].kwargs['input'])
        self.assertIn('actual-controller', json.dumps(document))
        self.assertIn('ALLOWEDSERVICEACCOUNTS=' + ACCOUNT, calls[1].args[0])
        self.assertIn(
            'KFP_SECURITY_SERVICE_ACCOUNT_MODE=enforce', calls[1].args[0]
        )
        self.assertIn('rollout', calls[2].args[0])
        self.assertTrue(all(call.kwargs['check'] for call in calls))

    def test_fixture_grants_only_named_accounts_in_one_namespace(self):
        items = fixture('profile-a', 'system', 'controller-sa')['items']
        self.assertTrue(
            all(item['metadata']['namespace'] == 'profile-a' for item in items)
        )
        roles = {
            item['metadata']['name']: item
            for item in items
            if item['kind'] == 'Role'
        }
        bindings = {
            item['metadata']['name']: item
            for item in items
            if item['kind'] == 'RoleBinding'
        }
        use = roles['kfp-schedule-test-use']['rules']
        self.assertEqual(
            use,
            [
                {
                    'apiGroups': [''],
                    'resources': ['serviceaccounts'],
                    'verbs': ['use'],
                    'resourceNames': [ACCOUNT],
                }
            ],
        )
        self.assertEqual(
            bindings['kfp-schedule-test-use']['subjects'],
            [
                {
                    'kind': 'ServiceAccount',
                    'name': 'default-editor',
                    'namespace': 'profile-a',
                },
                {
                    'kind': 'ServiceAccount',
                    'name': 'controller-sa',
                    'namespace': 'system',
                },
            ],
        )
        token = roles['kfp-schedule-test-token']['rules'][0]
        self.assertEqual(token['resourceNames'], [ACCOUNT])
        self.assertEqual(token['resources'], ['serviceaccounts/token'])
        self.assertEqual(token['verbs'], ['create'])
        self.assertEqual(
            bindings['kfp-schedule-test-token']['subjects'],
            [
                {
                    'kind': 'ServiceAccount',
                    'name': 'ml-pipeline-driver-agent-executor-plugin',
                    'namespace': 'profile-a',
                },
            ],
        )
        for role in roles.values():
            for rule in role['rules']:
                self.assertNotIn('*', rule['resources'])
                self.assertNotIn('*', rule['verbs'])
        self.assertEqual(
            bindings['kfp-schedule-test-runtime']['subjects'][0]['name'],
            ACCOUNT,
        )

    def test_required_cases_reject_skips_and_missing_reports(self):
        labels = json.loads(
            (
                ROOT / '.github/resources/legacy-v2-service-account-specs.json'
            ).read_text()
        )
        self.assertEqual(
            set(labels),
            {
                'LegacyRecurringCustomAccountPinnedtrue',
                'LegacyRecurringCustomAccountPinnedfalse',
            },
        )
        self.assertTrue(verify([], labels))
        for state in ('skipped', 'pending', 'failed'):
            self.assertTrue(
                verify(
                    [
                        {
                            'SpecReports': [
                                {'LeafNodeLabels': [label], 'State': state}
                                for label in labels
                            ]
                        }
                    ],
                    labels,
                )
            )
        self.assertFalse(
            verify(
                [
                    {
                        'SpecReports': [
                            {'LeafNodeLabels': [label], 'State': 'passed'}
                            for label in labels
                        ]
                    }
                ],
                labels,
            )
        )

    def test_automatic_cached_multiuser_shard_requires_fixture_and_manifest(
        self,
    ):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/api-server-tests.yml').read_text()
        )
        steps = workflow['jobs']['api-test-multi-user']['steps']
        fixture = next(
            step for step in steps if step.get('id') == 'schedule-account'
        )
        self.assertIn("matrix.test_shard == 'B'", fixture['if'])
        self.assertIn("matrix.cache_enabled == 'true'", fixture['if'])
        self.assertIn("github.event_name != 'workflow_dispatch'", fixture['if'])
        run = next(step for step in steps if step.get('id') == 'test-run')
        self.assertIn(
            'schedule-account', run['with']['required_service_account_specs']
        )
        self.assertIn(
            'schedule-account', run['env']['KFP_SCHEDULE_TEST_SERVICE_ACCOUNT']
        )
        self.assertLess(steps.index(fixture), steps.index(run))


if __name__ == '__main__':
    unittest.main()
