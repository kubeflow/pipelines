#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Guard the trusted, disabled-by-default queue admission workflow boundary."""

import json
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class AdmissionWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.load(
            (ROOT / '.github/workflows/ci-checks.yml').read_text(),
            Loader=yaml.BaseLoader,
        )
        cls.job = cls.workflow['jobs']['queue_admission']

    def test_default_policy_does_not_create_app_token_or_run_admission(self):
        policy = json.loads(
            (ROOT / '.github/merge-queue-policy.json').read_text()
        )
        self.assertEqual(policy, {'version': 1, 'enabled_branches': []})
        steps = self.job['steps']
        self.assertEqual(steps[0]['with']['ref'], '${{ github.workflow_sha }}')
        self.assertEqual(steps[0]['with']['persist-credentials'], 'false')
        self.assertEqual(steps[1]['id'], 'policy')
        self.assertIn(
            'loadPolicy().enabled_branches.length > 0',
            steps[1]['with']['script'],
        )
        self.assertEqual(
            steps[2]['if'], "steps.policy.outputs.enabled == 'true'"
        )
        self.assertEqual(
            steps[3]['if'],
            "always() && !cancelled() && steps.policy.outputs.enabled == 'true'",
        )

    def test_app_only_mutates_queue_and_actions_retains_status_binding(self):
        self.assertEqual(
            self.job['permissions'],
            {'contents': 'read', 'pull-requests': 'read', 'statuses': 'write'},
        )
        app = self.job['steps'][2]
        self.assertEqual(app['with']['owner'], '${{ github.repository_owner }}')
        self.assertEqual(
            app['with']['repositories'], '${{ github.event.repository.name }}'
        )
        self.assertEqual(
            {
                key: value
                for key, value in app['with'].items()
                if key.startswith('permission-')
            },
            {
                'permission-contents': 'write',
                'permission-pull-requests': 'write',
            },
        )
        script = self.job['steps'][3]['with']['script']
        self.assertIn('mutationGithub', script)
        self.assertIn(
            'if (!process.env.QUEUE_APP_TOKEN) throw new Error', script
        )
        self.assertIn(
            'await sweep({github, mutationGithub, context, core})', script
        )

    def test_revocation_events_and_sweep_do_not_deadlock_ci(self):
        events = self.workflow['on']['pull_request_target']['types']
        for event in [
            'labeled',
            'unlabeled',
            'synchronize',
            'edited',
            'closed',
            'ready_for_review',
            'converted_to_draft',
        ]:
            self.assertIn(event, events)
        self.assertIn('schedule', self.workflow['on'])
        self.assertEqual(self.job['needs'], 'check_ci_status')
        self.assertEqual(self.job['concurrency']['cancel-in-progress'], 'false')
        self.assertIn('always()', self.job['if'])
        self.assertIn("github.event.action == 'completed'", self.job['if'])
        for job in ['check_ci_status', 'recover_merged_release']:
            poll = next(
                step
                for step in self.workflow['jobs'][job]['steps']
                if step.get('id') == 'poll'
            )
            self.assertIn('^queue_admission$', poll['with']['checks_exclude'])


if __name__ == '__main__':
    unittest.main()
