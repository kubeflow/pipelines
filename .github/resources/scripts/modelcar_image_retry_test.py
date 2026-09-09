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
"""Regression tests for centralized Modelcar fixture image wiring."""

import unittest
from pathlib import Path

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
E2E_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/e2e-test.yml'
DEPLOY_ACTION = REPOSITORY_ROOT / '.github/actions/deploy/action.yml'
IMAGE_BUILDS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/image-builds.yml'
RUNTIME_IMAGES_WORKFLOW = (
    REPOSITORY_ROOT / '.github/workflows/runtime-base-images.yml')
CI_SCRIPTS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
MODELCAR_DOCKERFILE = (
    'test_data/sdk_compiled_pipelines/valid/critical/modelcar/Dockerfile')
MODELCAR_IMAGE = 'registry.domain.local/modelcar:test'


class ModelcarImageArtifactTest(unittest.TestCase):

    def test_producer_builds_and_archives_modelcar_once(self):
        workflow = RUNTIME_IMAGES_WORKFLOW.read_text(encoding='utf-8')
        build_step = workflow.split(
            '- name: Build and save the sample Modelcar fixture image',
            maxsplit=1,
        )[1].split(
            '\n      - name:', maxsplit=1)[0]

        self.assertIn(
            "if: steps.cache-runtime-base-images.outputs.cache-hit != 'true'",
            build_step)
        self.assertIn('retry 3 30 env DOCKER_BUILDKIT=1 docker build',
                      build_step)
        self.assertIn(f'--file {MODELCAR_DOCKERFILE}', build_step)
        self.assertIn(f'--tag {MODELCAR_IMAGE}', build_step)
        self.assertIn(f'docker save {MODELCAR_IMAGE}', build_step)
        self.assertIn('${ARTIFACTS_PATH}/${MODELCAR_IMAGE_ARCHIVE}', build_step)

    def test_e2e_consumes_prebuilt_modelcar_and_fails_closed(self):
        workflow = E2E_WORKFLOW.read_text(encoding='utf-8')
        deploy_action = DEPLOY_ACTION.read_text(encoding='utf-8')

        self.assertEqual(workflow.count("load_modelcar_fixture: 'true'"), 3)
        self.assertNotIn('Build and upload the sample Modelcar image', workflow)
        self.assertNotIn(f'--tag {MODELCAR_IMAGE}', workflow)

        self.assertIn("if: ${{ inputs.load_modelcar_fixture == 'true' }}",
                      deploy_action)
        self.assertIn('runtime-base-images/modelcar.tar', deploy_action)
        self.assertIn('Modelcar fixture archive not found', deploy_action)
        self.assertIn('retry 3 10 kind --name', deploy_action)
        self.assertIn('load image-archive "${MODELCAR_IMAGE_ARCHIVE}"',
                      deploy_action)
        self.assertNotIn('docker build', deploy_action)

    def test_modelcar_inputs_are_fingerprinted_and_forwarded(self):
        producer = RUNTIME_IMAGES_WORKFLOW.read_text(encoding='utf-8')
        consumer = IMAGE_BUILDS_WORKFLOW.read_text(encoding='utf-8')

        self.assertGreaterEqual(producer.count(MODELCAR_DOCKERFILE), 3)
        self.assertIn(MODELCAR_DOCKERFILE, consumer)
        self.assertIn('path: ${{ env.ARTIFACTS_PATH }}', producer)
        self.assertIn('path: ${{ env.ARTIFACTS_PATH }}', consumer)

    def test_ci_runs_when_modelcar_wiring_changes(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        for path in (
                "'.github/actions/deploy/**'",
                "'.github/workflows/e2e-test.yml'",
                "'.github/workflows/image-builds.yml'",
                "'.github/workflows/runtime-base-images.yml'",
                f"'{MODELCAR_DOCKERFILE}'",
        ):
            with self.subTest(path=path):
                self.assertIn(path, workflow)


if __name__ == '__main__':
    unittest.main()
