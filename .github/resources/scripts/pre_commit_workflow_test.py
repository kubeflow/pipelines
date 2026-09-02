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

from pathlib import Path
import unittest

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
PRE_COMMIT_WORKFLOW = (
    REPOSITORY_ROOT / '.github' / 'workflows' / 'pre-commit.yml')
PRE_COMMIT_CONFIG = REPOSITORY_ROOT / '.pre-commit-config.yaml'
CI_SCRIPTS_WORKFLOW = (
    REPOSITORY_ROOT / '.github' / 'workflows' / 'ci-scripts-tests.yml')


class PreCommitWorkflowTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = PRE_COMMIT_WORKFLOW.read_text(encoding='utf-8')
        cls.config = PRE_COMMIT_CONFIG.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW.read_text(
            encoding='utf-8')

    def test_workflow_uses_the_pinned_pre_commit_hooks(self):
        self.assertIn('uses: pre-commit/action@', self.workflow)
        self.assertNotIn('golangci/golangci-lint-action', self.workflow)
        self.assertIn('persist-credentials: false', self.workflow)
        self.assertIn('permissions:\n  contents: read', self.workflow)
        self.assertIn(
            '--from-ref ${{ steps.pre-commit-range.outputs.base-sha }}',
            self.workflow)
        self.assertIn('PRE_COMMIT_FROM_REF', self.config)
        self.assertIn('git diff --quiet "${base}" -- "*.go"', self.config)
        self.assertIn('golangci-lint config verify', self.config)
        self.assertIn(
            'echo "PRE_COMMIT_FROM_REF=${base_sha}" >> "${GITHUB_ENV}"',
            self.workflow,
        )

    def test_workflow_fails_closed_when_the_event_base_is_unavailable(self):
        self.assertIn('Unable to resolve the event base commit', self.workflow)
        self.assertNotIn('git rev-parse HEAD^', self.workflow)

    def test_config_changes_execute_each_applicable_hook_family(self):
        self.assertIn(
            "if: steps.pre-commit-range.outputs.config-changed == 'true'",
            self.workflow)
        for representative_file in (
                '.pre-commit-config.yaml',
                '.github/workflows/pre-commit.yml',
                '.golangci.yaml',
                'frontend/package.json',
                'sdk/python/kfp/cli/__init__.py',
                'backend/src/common/types.go',
        ):
            with self.subTest(representative_file=representative_file):
                self.assertIn(representative_file, self.workflow)

        self.assertIn('id: golangci-lint-fmt', self.config)
        self.assertIn('id: golangci-lint-config-verify', self.config)

    def test_workflow_changes_run_ci_script_tests(self):
        self.assertIn("      - '.pre-commit-config.yaml'",
                      self.ci_scripts_workflow)
        self.assertIn("      - '.github/workflows/pre-commit.yml'",
                      self.ci_scripts_workflow)


if __name__ == '__main__':
    unittest.main()
