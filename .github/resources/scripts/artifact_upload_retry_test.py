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
"""Regression tests for artifact-upload retry wiring."""

import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
RETRY_ACTION = (
    REPOSITORY_ROOT / '.github/actions/upload-artifact-with-retry/action.yml'
)
TEST_AND_REPORT_ACTION = (
    REPOSITORY_ROOT / '.github/actions/test-and-report/action.yml'
)
CI_SCRIPTS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'


class ArtifactUploadRetryTest(unittest.TestCase):

    def test_retry_action_preserves_outputs_and_avoids_name_collisions(self):
        action = RETRY_ACTION.read_text(encoding='utf-8')

        self.assertEqual(action.count('uses: actions/upload-artifact@v7'), 2)
        self.assertIn("steps.primary.outcome == 'failure'", action)
        self.assertIn('sleep "$RETRY_DELAY_SECONDS"', action)
        self.assertIn(
            'name: ${{ inputs.name }} - retry-${{ github.run_attempt }}', action
        )
        self.assertIn(
            'steps.primary.outputs.artifact-url || steps.retry.outputs.artifact-url',
            action,
        )
        self.assertIn(
            '[[ "$PRIMARY_OUTCOME" == "success" || "$RETRY_OUTCOME" == "success" ]]',
            action,
        )

    def test_test_and_report_uses_retry_for_each_artifact_upload(self):
        action = TEST_AND_REPORT_ACTION.read_text(encoding='utf-8')

        self.assertEqual(
            action.count('uses: ./.github/actions/upload-artifact-with-retry'),
            3,
        )
        self.assertNotIn('uses: actions/upload-artifact@v7', action)

    def test_ci_runs_when_either_action_changes(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        self.assertIn("'.github/actions/upload-artifact-with-retry/**'", workflow)
        self.assertIn("'.github/actions/test-and-report/**'", workflow)


if __name__ == '__main__':
    unittest.main()
