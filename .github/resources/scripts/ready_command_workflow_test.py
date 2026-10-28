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
"""Guard the trust boundaries of the /ready issue command workflow."""

import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/ready-command.yml'


class ReadyCommandWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.safe_load(WORKFLOW.read_text(encoding='utf-8'))
        cls.job = cls.workflow['jobs']['ready']
        cls.steps = cls.job['steps']

    def step_using(self, action: str) -> dict:
        """Return the only step that uses the given action."""
        (step,) = [
            step
            for step in self.steps
            if step.get('uses', '').split('@')[0] == action
        ]
        return step

    def test_runs_only_for_issue_comments_in_upstream_repository(self):
        # PyYAML parses the bare `on` key as boolean True.
        self.assertEqual(
            self.workflow[True], {'issue_comment': {'types': ['created']}}
        )
        self.assertIn(
            "github.repository == 'kubeflow/pipelines'", self.job['if']
        )
        self.assertIn('!github.event.issue.pull_request', self.job['if'])

    def test_owners_and_scripts_come_from_trusted_workflow_revision(self):
        checkout = self.step_using('actions/checkout')['with']
        self.assertEqual(checkout['ref'], '${{ github.workflow_sha }}')
        self.assertIs(checkout['persist-credentials'], False)
        self.assertIs(checkout['sparse-checkout-cone-mode'], False)
        self.assertEqual(
            checkout['sparse-checkout'].split(),
            [
                'OWNERS',
                '.github/scripts/ready_command.py',
                '.github/scripts/kubeflow_membership.py',
                '.github/scripts/requirements.txt',
            ],
        )

    def test_token_has_only_required_scopes(self):
        self.assertEqual(self.workflow['permissions'], {})
        self.assertEqual(
            self.job['permissions'], {'contents': 'read', 'issues': 'write'}
        )

    def test_no_script_interpolates_expressions(self):
        # Expressions in run scripts are expanded before the shell parses
        # them, which would let comment text inject commands.
        for step in self.steps:
            with self.subTest(step=step.get('name')):
                self.assertNotIn('${{', step.get('run', ''))

    def test_event_data_and_token_reach_only_the_command_step(self):
        (command_step,) = [
            step
            for step in self.steps
            if 'ready_command.py' in step.get('run', '')
        ]
        self.assertEqual(
            command_step['env']['COMMENT_BODY'],
            '${{ github.event.comment.body }}',
        )
        for step in self.steps:
            if step is command_step:
                continue
            with self.subTest(step=step.get('name')):
                rendered = yaml.safe_dump(step)
                self.assertNotIn('github.event', rendered)
                self.assertNotIn('secrets.', rendered)
        self.assertNotIn('env', self.job)
        self.assertNotIn('env', self.workflow)


if __name__ == '__main__':
    unittest.main()
