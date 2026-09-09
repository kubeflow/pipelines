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
import re
import subprocess
import unittest

ROOT = Path(__file__).parents[3]
ARTIFACTS_SCRIPT = Path(__file__).with_name('ci-image-artifacts.sh')
IMAGE_BUILDS_WORKFLOW = ROOT / '.github' / 'workflows' / 'image-builds.yml'
FRONTEND_E2E_WORKFLOW = (
    ROOT / '.github' / 'workflows' / 'e2e-test-frontend.yml')
CI_SCRIPTS_WORKFLOW = ROOT / '.github' / 'workflows' / 'ci-scripts-tests.yml'


class CiImageArtifactsTest(unittest.TestCase):

    def test_artifact_list_matches_image_build_matrix(self):
        result = subprocess.run(
            [
                'bash',
                '-c',
                f'source "{ARTIFACTS_SCRIPT}"; '
                'printf "%s\\n" "${ALL_CI_IMAGE_ARTIFACTS[@]}"',
            ],
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        configured_artifacts = set(result.stdout.splitlines())

        workflow = IMAGE_BUILDS_WORKFLOW.read_text(encoding='utf-8')
        built_images = set(
            re.findall(r'^\s+- image:\s+([^\s]+)\s*$', workflow, re.MULTILINE))
        expected_artifacts = built_images | {'runtime-base-images'}

        self.assertEqual(configured_artifacts, expected_artifacts)

    def test_metadata_envoy_changes_trigger_its_image_build(self):
        frontend_e2e_workflow = FRONTEND_E2E_WORKFLOW.read_text(
            encoding='utf-8')
        image_builds_workflow = IMAGE_BUILDS_WORKFLOW.read_text(
            encoding='utf-8')
        ci_scripts_workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        pull_request_block = frontend_e2e_workflow.split(
            '  pull_request:\n', maxsplit=1)[1].split(
                '\n\nconcurrency:', maxsplit=1)[0]

        self.assertIn("      - 'third_party/metadata_envoy/**'",
                      pull_request_block)
        self.assertIn(
            '  build:\n    uses: ./.github/workflows/image-builds.yml',
            frontend_e2e_workflow,
        )
        self.assertIn(
            '          - image: metadata-envoy\n'
            '            dockerfile: third_party/metadata_envoy/Dockerfile\n'
            '            context: .',
            image_builds_workflow,
        )
        self.assertIn("      - '.github/workflows/e2e-test-frontend.yml'",
                      ci_scripts_workflow)


if __name__ == '__main__':
    unittest.main()
