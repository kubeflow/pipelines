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
"""Guard the trusted queue admission workflow boundary."""

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

    def test_master_activation_retains_trusted_policy_guard(self):
        policy = json.loads(
            (ROOT / '.github/merge-queue-policy.json').read_text()
        )
        self.assertEqual(policy, {'version': 1, 'enabled_branches': ['master']})
        steps = self.job['steps']
        self.assertEqual(steps[0]['with']['ref'], '${{ github.workflow_sha }}')
        self.assertEqual(steps[0]['with']['persist-credentials'], 'false')
        self.assertEqual(steps[1]['id'], 'policy')
        for step in steps[2:]:
            self.assertEqual(
                step['if'], "steps.policy.outputs.enabled == 'true'"
            )
        self.assertNotIn('secrets.', json.dumps(self.job))
        self.assertNotIn('create-github-app-token', json.dumps(self.job))

    def test_builtin_token_owns_queue_dispatch_and_status_publication(self):
        self.assertEqual(
            self.job['permissions'],
            {
                'contents': 'write',
                'pull-requests': 'write',
                'actions': 'write',
                'statuses': 'write',
            },
        )
        script = self.job['steps'][-1]['with']['script']
        self.assertIn('await reconcileQueue({github, context, core})', script)
        self.assertNotIn('authorization', script)
        publisher = self.workflow['jobs']['check_merge_group_status']
        self.assertIn(
            "github.event.workflow_run.event == 'workflow_dispatch'",
            publisher['if'],
        )
        self.assertIn("'gh-readonly-queue/master/'", publisher['if'])
        self.assertIn("'gh-readonly-queue/main/'", publisher['if'])

    def test_scheduled_recovery_shares_serial_writer_and_needs_no_event(self):
        recovery = yaml.load(
            (ROOT / '.github/workflows/merge-queue-recovery.yml').read_text(),
            Loader=yaml.BaseLoader,
        )
        self.assertEqual(recovery['on']['schedule'], [{'cron': '*/5 * * * *'}])
        job = recovery['jobs']['queue_recovery']
        self.assertEqual(job['concurrency'], self.job['concurrency'])
        self.assertNotIn('needs', job)
        self.assertEqual(job['permissions'], self.job['permissions'])
        self.assertEqual(
            job['steps'][0]['with']['ref'], '${{ github.workflow_sha }}'
        )
        self.assertIn('queueOnly: true', job['steps'][-1]['with']['script'])

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
