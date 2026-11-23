#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Guard the trust boundaries of the bot branch cleanup workflow."""

import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/bot-branch-cleanup.yml'


class BotBranchCleanupWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))
        cls.job = cls.workflow['jobs']['delete-branch']
        cls.steps = cls.job['steps']

    def step_using(self, action: str) -> dict:
        """Return the only step that uses the given action."""
        (step,) = [
            step
            for step in self.steps
            if step.get('uses', '').split('@')[0] == action
        ]
        return step

    def test_runs_only_for_closed_pull_requests(self):
        # PyYAML parses the bare `on` key as boolean True.
        self.assertEqual(
            self.workflow[True], {'pull_request_target': {'types': ['closed']}}
        )

    def test_job_filters_unmerged_bot_branches_in_upstream_repository(self):
        condition = ' '.join(self.job['if'].split())
        for clause in (
            "github.repository == 'kubeflow/pipelines'",
            '!github.event.pull_request.merged',
            'github.event.pull_request.head.repo.full_name == '
            'github.repository',
            "github.event.pull_request.user.login == 'dependabot[bot]' && "
            "startsWith(github.event.pull_request.head.ref, 'dependabot/')",
            "github.event.pull_request.user.login == 'copybara-service[bot]'"
            " && startsWith(github.event.pull_request.head.ref, 'test_')",
        ):
            with self.subTest(clause=clause):
                self.assertIn(clause, condition)
        # The author decides eligibility, not whoever closed the PR.
        self.assertNotIn('github.actor', condition)

    def test_only_the_trusted_script_is_checked_out(self):
        checkout = self.step_using('actions/checkout')['with']
        self.assertEqual(checkout['ref'], '${{ github.workflow_sha }}')
        self.assertIs(checkout['persist-credentials'], False)
        self.assertIs(checkout['sparse-checkout-cone-mode'], False)
        self.assertEqual(
            checkout['sparse-checkout'].split(),
            ['.github/scripts/bot_branch_cleanup.py'],
        )

    def test_token_has_only_required_scopes(self):
        self.assertEqual(self.workflow['permissions'], {})
        self.assertEqual(
            self.job['permissions'],
            {'contents': 'write', 'pull-requests': 'read'},
        )

    def test_no_script_interpolates_expressions(self):
        # Expressions in run scripts are expanded before the shell parses
        # them, which would let a branch name inject commands.
        for step in self.steps:
            with self.subTest(step=step.get('name')):
                self.assertNotIn('${{', step.get('run', ''))

    def test_event_data_and_token_reach_only_the_cleanup_step(self):
        (cleanup_step,) = [
            step
            for step in self.steps
            if 'bot_branch_cleanup.py' in step.get('run', '')
        ]
        self.assertEqual(
            cleanup_step['env']['HEAD_REF'],
            '${{ github.event.pull_request.head.ref }}',
        )
        self.assertEqual(
            cleanup_step['env']['GH_TOKEN'], '${{ secrets.GITHUB_TOKEN }}'
        )
        for step in self.steps:
            if step is cleanup_step:
                continue
            with self.subTest(step=step.get('name')):
                rendered = yaml.safe_dump(step)
                self.assertNotIn('github.event', rendered)
                self.assertNotIn('secrets.', rendered)
        self.assertNotIn('env', self.job)
        self.assertNotIn('env', self.workflow)


if __name__ == '__main__':
    unittest.main()
