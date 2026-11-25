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
import re
import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import evaluate

ROOT = Path(__file__).resolve().parents[3]


class AdmissionWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.load(
            (ROOT / '.github/workflows/ci-checks.yml').read_text(),
            Loader=yaml.BaseLoader,
        )
        cls.job = cls.workflow['jobs']['queue_admission']

    def test_default_branch_activation_retains_trusted_policy_guard(self):
        policy = json.loads(
            (ROOT / '.github/merge-queue-policy.json').read_text()
        )
        self.assertEqual(
            policy,
            {
                'version': 1,
                'enabled_branches': ['~DEFAULT_BRANCH', 'release-2.18'],
            },
        )
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
        self.assertIn(
            "queueOnly: context.eventName === 'schedule'",
            job['steps'][-1]['with']['script'],
        )
        self.assertIn('github.event.repository.default_branch', job['if'])
        self.assertEqual(
            set(recovery['on']['workflow_dispatch']['inputs']),
            {'pr_number', 'head_sha'},
        )

    def test_verified_publisher_wakes_admission_with_accounting(self) -> None:
        publisher = self.workflow['jobs']['check_ci_status']
        self.assertEqual(publisher['permissions']['actions'], 'write')
        script = publisher['steps'][-1]['with']['script']
        self.assertIn('const pr = await finalize(', script)
        self.assertIn(
            'await wakeAdmission({github, context, core, pr})', script
        )
        self.assertLess(
            script.index('withRequestAccounting({'),
            script.index('await wakeAdmission('),
        )

    def test_queue_writer_filters_completion_events_before_runner_start(self):
        upstream = 'kubeflow/pipelines'
        queue_ref = 'gh-readonly-queue/release-2.18/pr-7-abc'
        for event, action, branch, repo, expected in [
            ('pull_request', 'completed', 'feature', 'fork/repo', True),
            ('merge_group', 'completed', queue_ref, upstream, True),
            ('workflow_dispatch', 'completed', queue_ref, upstream, True),
            ('merge_group', 'in_progress', queue_ref, upstream, False),
            ('merge_group', 'completed', queue_ref, 'fork/repo', False),
            ('workflow_dispatch', 'completed', 'main', upstream, False),
            ('workflow_dispatch', 'completed', queue_ref, 'fork/repo', False),
            (
                'workflow_dispatch',
                'completed',
                'gh-readonly-queue/dev/pr-7-abc',
                upstream,
                False,
            ),
        ]:
            with self.subTest(
                event=event, action=action, branch=branch, repo=repo
            ):
                github = {
                    'repository': upstream,
                    'event_name': 'workflow_run',
                    'event': {
                        'action': action,
                        'workflow_run': {
                            'event': event,
                            'head_branch': branch,
                            'head_repository': {'full_name': repo},
                        },
                    },
                }
                self.assertEqual(
                    evaluate(
                        self.job['if'],
                        {'github': github},
                        {'always': lambda: True, 'cancelled': lambda: False},
                    ),
                    expected,
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
        self.assertNotIn('needs', self.job)
        self.assertEqual(self.job['concurrency']['cancel-in-progress'], 'false')
        self.assertIn('always()', self.job['if'])
        self.assertIn("github.event.action == 'completed'", self.job['if'])
        checker = (
            ROOT / '.github/resources/scripts/ci_check_runs.js'
        ).read_text()
        pattern = re.search(r'const EXCLUDED = /(.+)/;', checker).group(1)
        self.assertIsNotNone(re.fullmatch(pattern, 'queue_admission'))
        self.assertIsNone(re.fullmatch(pattern, 'queue_admission_unrelated'))
        for job, function in [
            ('check_ci_status', 'finalize'),
            ('recover_merged_release', 'finalizeMergedRecovery'),
        ]:
            finalizer = self.workflow['jobs'][job]['steps'][-1]
            self.assertIn(function + '({github', finalizer['with']['script'])
            self.assertFalse(
                any(
                    step.get('id') == 'poll'
                    for step in self.workflow['jobs'][job]['steps']
                )
            )


if __name__ == '__main__':
    unittest.main()
