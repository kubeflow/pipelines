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
"""Verify frontend image timeouts leave room for recovery and artifact
upload."""

import itertools
import json
import os
import subprocess
import sys
import unittest
from pathlib import Path
from unittest import mock

import build_frontend_image
import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/image-builds.yml'


class FrontendImageRetryTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.job = yaml.safe_load(WORKFLOW.read_text())['jobs']['image-build']
        cls.steps = cls.job['steps']
        cls.frontend = next(
            s for s in cls.steps if s.get('id') == 'frontend-image'
        )
        cls.first = next(s for s in cls.steps if s.get('id') == 'save-image')
        cls.retry = next(s for s in cls.steps if s.get('id') == 'rebuild')

    def test_frontend_owns_its_recovery_and_other_images_keep_action_retry(
        self,
    ):
        self.assertEqual(
            self.frontend['if'], "${{ matrix.image == 'frontend' }}"
        )
        self.assertNotIn('continue-on-error', self.frontend)
        setup = next(s for s in self.steps if s.get('id') == 'buildx')
        for step in (setup, self.first):
            self.assertEqual(step['if'], "${{ matrix.image != 'frontend' }}")
        self.assertTrue(self.first['continue-on-error'])
        self.assertEqual(
            self.retry['if'], "${{ steps.save-image.outcome == 'failure' }}"
        )
        self.assertFalse(self.retry.get('continue-on-error', False))
        self.assertTrue(self.retry['with']['no-cache'])
        self.assertFalse(self.job['strategy']['fail-fast'])

    def test_workflow_arguments_reach_helper_with_native_platform_and_tags(
        self,
    ):
        # Execute shell expansion so read-only callers must omit --cache-to.
        for architecture, cache_write in itertools.product(
            ('amd64', 'arm64'), (False, True)
        ):
            artifact = 'frontend' + (
                '-arm64' if architecture == 'arm64' else ''
            )
            cache_to = (
                (
                    f'type=gha,version=2,scope=kfp-image-frontend-{architecture},'
                    'mode=max,timeout=120s,ignore-error=true'
                )
                if cache_write
                else ''
            )
            values = {
                **os.environ,
                'REGISTRY': 'kind-registry:5000',
                'IMAGE_TAG': 'latest',
                'TARGET_ARCH': architecture,
                'NODE_VERSION': '24',
                'ARTIFACTS_PATH': 'images_123',
                'ARTIFACT_NAME': artifact,
                'IMAGE_CACHE_TO': cache_to,
                'TEST_PYTHON': sys.executable,
            }
            command = self.frontend['run'].replace(
                '${{ inputs.architecture }}', architecture
            )
            capture = """python3() {
  "$TEST_PYTHON" -c 'import json, sys; print(json.dumps(sys.argv[1:]))' "$@"
}
"""
            result = subprocess.run(
                ['bash', '-eu', '-c', capture + command],
                env=values,
                check=True,
                capture_output=True,
                text=True,
            )
            arguments = json.loads(result.stdout)
            self.assertEqual(
                arguments[0],
                '.github/resources/scripts/build_frontend_image.py',
            )
            self.assertEqual('--cache-to' in arguments, cache_write)
            with (
                mock.patch('sys.argv', arguments),
                mock.patch.object(
                    build_frontend_image, 'build', return_value=0
                ) as build,
                mock.patch('signal.signal'),
            ):
                self.assertEqual(build_frontend_image.main(), 0)
            build.assert_called_once_with(
                '.',
                'frontend/Dockerfile',
                'kind-registry:5000/frontend:latest',
                '24',
                f'images_123/{artifact}.tar',
                platform=f'linux/{architecture}',
                additional_tags=['kind-registry:5000/frontend:ci'],
                cache_from=(
                    f'type=gha,version=2,scope=kfp-image-frontend-{architecture},'
                    'timeout=60s'
                ),
                cache_to=cache_to or None,
            )

    def test_only_successful_builds_publish_and_record_arm_identity(self):
        success = "steps.frontend-image.outcome == 'success' || steps.save-image.outcome == 'success' || steps.rebuild.outcome == 'success'"
        upload = next(s for s in self.steps if s['name'] == 'Upload artifact')
        identity = next(
            s for s in self.steps if s['name'] == 'Record ARM image identity'
        )
        self.assertEqual(upload['if'], '${{ ' + success + ' }}')
        self.assertEqual(
            identity['if'],
            "${{ inputs.architecture == 'arm64' && (" + success + ') }}',
        )
        self.assertEqual(upload['with']['if-no-files-found'], 'error')
        self.assertLess(
            self.steps.index(self.frontend), self.steps.index(identity)
        )
        self.assertLess(self.steps.index(identity), self.steps.index(upload))
        self.assertIn('${{ env.ARTIFACT_NAME }}.tar', upload['with']['path'])
        self.assertIn('${{ env.ARTIFACT_NAME }}.json', upload['with']['path'])

    def test_bounded_attempts_and_cleanup_leave_setup_and_upload_budget(self):
        import inspect

        defaults = inspect.signature(build_frontend_image.build).parameters
        attempt = defaults['attempt_seconds'].default
        maintenance = defaults['maintenance_seconds'].default
        grace = defaults['grace'].default
        # Worst case per attempt: setup/build termination plus two diagnostics,
        # builder removal, container removal, and an absence query.
        worst_case = 2 * (attempt + grace + 5 * (maintenance + grace))
        self.assertGreaterEqual(
            self.job['timeout-minutes'] * 60 - worst_case, 540
        )


if __name__ == '__main__':
    unittest.main()
