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

import os
from pathlib import Path
import re
import subprocess
import unittest

import yaml

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

    def test_visualization_runtime_has_native_arm64_coverage(self):
        job = yaml.safe_load(
            IMAGE_BUILDS_WORKFLOW.read_text())['jobs']['image-build']
        entries = job['strategy']['matrix']['include']
        visualization = [
            entry for entry in entries
            if entry['image'] == 'visualization-server'
        ]
        self.assertEqual(len(visualization), 2)
        self.assertEqual(
            {entry.get('runs_on', 'ubuntu-latest') for entry in visualization},
            {'ubuntu-latest', 'ubuntu-24.04-arm'})
        self.assertIn('matrix.runs_on', job['runs-on'])
        self.assertIn('matrix.artifact_name', job['env']['ARTIFACT_NAME'])
        artifacts = [
            entry.get('artifact_name', entry['image']) for entry in entries
        ]
        self.assertEqual(len(artifacts), len(set(artifacts)))
        for entry in visualization:
            self.assertEqual(entry['dockerfile'],
                             'backend/Dockerfile.visualization')
        self.assertEqual(
            sum(
                entry.get('runs_on') == 'ubuntu-24.04-arm'
                for entry in entries), 1)

    def test_runtime_verification_propagates_container_failure(self):
        steps = yaml.safe_load(
            IMAGE_BUILDS_WORKFLOW.read_text())['jobs']['image-build']['steps']
        verify = next(step for step in steps
                      if step['name'] == 'Verify Python runtime images')
        upload = next(
            step for step in steps if step['name'] == 'Upload artifact')
        self.assertLess(steps.index(verify), steps.index(upload))
        self.assertIn('success()', upload['if'])
        stub = """docker() {
          printf '%s\\n' "$*"
          case "$*" in
            *test_runtime_image.py*|*unittest*) return "$SMOKE_EXIT" ;;
          esac
        }
"""
        for image in ('visualization-server', 'metadata-writer'):
            self.assertIn(image, verify['if'])
            for status in (0, 31):
                with self.subTest(image=image, status=status):
                    result = subprocess.run(
                        ['bash', '-eu', '-c', stub + verify['run']],
                        env={
                            **os.environ, 'RUNTIME_IMAGE': image,
                            'SMOKE_EXIT': str(status),
                            'ARTIFACTS_PATH': '/tmp/images',
                            'ARTIFACT_NAME': image,
                            'REGISTRY': 'test',
                            'IMAGE_TAG': 'test'
                        },
                        text=True,
                        capture_output=True)
                    self.assertEqual(result.returncode, status, result.stderr)
                    self.assertIn('load --input', result.stdout)
                    self.assertIn('ensurepip', result.stdout)
                    self.assertIn(
                        'test_runtime_image.py' if image
                        == 'visualization-server' else 'unittest discover',
                        result.stdout)

    def test_merge_group_requires_exact_runtime_base_image_producer_sha(self):
        workflow = IMAGE_BUILDS_WORKFLOW.read_text(encoding='utf-8')
        self.assertIn(
            "REQUIRE_SOURCE_SHA: ${{ github.event_name == 'merge_group' }}",
            workflow)
        self.assertIn('producer_args+=(--require-source-sha)', workflow)
        self.assertIn(
            'actions/workflows/runtime-base-images-merge-group.yml/runs',
            workflow)
        self.assertIn('producer_args+=(--queue-runs-file "${queue_runs_file}")',
                      workflow)

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
            '  build:\n'
            '    permissions:\n'
            '      actions: read\n'
            '      contents: read\n'
            '    uses: ./.github/workflows/image-builds.yml',
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
