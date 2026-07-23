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
"""Regression tests for Modelcar fixture build retry wiring."""

import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
E2E_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/e2e-test.yml'
CI_SCRIPTS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
MODEL_STEP_NAME = 'Build and upload the sample Modelcar image to Kind'


class ModelcarImageRetryTest(unittest.TestCase):

    def test_each_modelcar_build_retries_and_fails_closed(self):
        workflow = E2E_WORKFLOW.read_text(encoding='utf-8')
        step_marker = f'      - name: {MODEL_STEP_NAME}'
        step_blocks = [
            section.split('\n      - name:', 1)[0]
            for section in workflow.split(step_marker)[1:]
        ]

        self.assertEqual(len(step_blocks), 3)
        for step in step_blocks:
            with self.subTest(step=step):
                self.assertIn(
                    'source ./.github/resources/scripts/helper-functions.sh', step
                )
                self.assertIn(
                    'retry 3 20 env DOCKER_BUILDKIT=1 docker build', step
                )
                self.assertIn(
                    'kind --name kfp load docker-image '
                    'registry.domain.local/modelcar:test',
                    step,
                )
                self.assertNotIn('continue-on-error: true', step)

    def test_ci_runs_when_e2e_workflow_changes(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        self.assertIn("'.github/workflows/e2e-test.yml'", workflow)


if __name__ == '__main__':
    unittest.main()
