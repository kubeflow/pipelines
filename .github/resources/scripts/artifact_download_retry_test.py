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
"""Regression tests for artifact-download retries and image preloading."""

import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
RETRY_ACTION = (
    REPOSITORY_ROOT / '.github/actions/download-artifact-with-retry/action.yml'
)
DEPLOY_ACTION = REPOSITORY_ROOT / '.github/actions/deploy/action.yml'
IMAGE_BUILDS_WORKFLOW = (
    REPOSITORY_ROOT / '.github/workflows/image-builds.yml'
)
CREATE_MANIFEST_WORKFLOW = (
    REPOSITORY_ROOT / '.github/workflows/create-manifest.yml'
)
CI_SCRIPTS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
RUNTIME_BASE_IMAGES = (
    REPOSITORY_ROOT / '.github/resources/runtime-base-images.txt'
)


class ArtifactDownloadRetryTest(unittest.TestCase):

    def test_retry_action_preserves_inputs_and_output(self):
        action = RETRY_ACTION.read_text(encoding='utf-8')

        self.assertEqual(action.count('uses: actions/download-artifact@v7'), 2)
        self.assertIn("steps.primary.outcome == 'failure'", action)
        self.assertIn('sleep "$RETRY_DELAY_SECONDS"', action)
        self.assertIn('pattern: ${{ inputs.pattern }}', action)
        self.assertIn('merge-multiple: ${{ inputs.merge-multiple }}', action)
        self.assertIn('github-token: ${{ inputs.github-token }}', action)
        self.assertIn('run-id: ${{ inputs.run-id }}', action)
        self.assertIn(
            'steps.primary.outputs.download-path || '
            'steps.retry.outputs.download-path',
            action,
        )
        self.assertIn(
            '[[ "$PRIMARY_OUTCOME" == "success" || '
            '"$RETRY_OUTCOME" == "success" ]]',
            action,
        )

    def test_every_artifact_download_uses_retry_action(self):
        callers = (
            DEPLOY_ACTION,
            IMAGE_BUILDS_WORKFLOW,
            CREATE_MANIFEST_WORKFLOW,
        )

        for caller in callers:
            with self.subTest(caller=caller.relative_to(REPOSITORY_ROOT)):
                contents = caller.read_text(encoding='utf-8')
                self.assertIn(
                    'uses: ./.github/actions/download-artifact-with-retry',
                    contents,
                )
                self.assertNotIn(
                    'uses: actions/download-artifact@',
                    contents,
                )

    def test_runtime_archive_contains_external_deployment_images(self):
        images = {
            line.strip()
            for line in RUNTIME_BASE_IMAGES.read_text(
                encoding='utf-8'
            ).splitlines()
            if line.strip() and not line.startswith('#')
        }

        self.assertTrue(
            {
                'docker.io/library/mysql:8.4',
                'quay.io/argoproj/workflow-controller:v3.7.14',
                'quay.io/argoproj/argoexec:v3.7.14',
                'quay.io/argoproj/workflow-controller:v4.0.5',
                'quay.io/argoproj/argoexec:v4.0.5',
            }.issubset(images)
        )

    def test_ci_runs_for_retry_wiring_and_runtime_image_changes(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        for path_filter in (
            "'.github/actions/download-artifact-with-retry/**'",
            "'.github/actions/deploy/**'",
            "'.github/resources/runtime-base-images.txt'",
            "'.github/workflows/create-manifest.yml'",
            "'.github/workflows/image-builds.yml'",
        ):
            with self.subTest(path_filter=path_filter):
                self.assertIn(path_filter, workflow)


if __name__ == '__main__':
    unittest.main()
