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
import tempfile
import unittest


WORKFLOWS = Path(__file__).resolve().parents[2] / 'workflows'
REPOSITORY_ROOT = WORKFLOWS.parents[1]


def workflow(name: str) -> str:
    return (WORKFLOWS / name).read_text(encoding='utf-8')


class ReleaseMergeGroupWorkflowsTest(unittest.TestCase):

    def test_e2e_workflows_run_for_release_queue_with_read_only_tokens(self):
        for name in ('e2e-test.yml', 'e2e-test-frontend.yml'):
            with self.subTest(workflow=name):
                source = workflow(name)
                self.assertIn(
                    '  merge_group:\n'
                    '    types: [checks_requested]\n'
                    '    branches: [release-2.18]\n', source)
                self.assertIn('permissions:\n  contents: read\n', source)
                self.assertRegex(
                    source,
                    r'(?m)^  build:\n    permissions:\n'
                    r'      actions: read\n      contents: read\n'
                    r'    uses: ./\.github/workflows/image-builds\.yml$')

    def test_queue_workflows_use_read_only_tokens(self):
        for path in (*WORKFLOWS.glob('*.yml'), *WORKFLOWS.glob('*.yaml')):
            source = path.read_text(encoding='utf-8')
            if '\n  merge_group:\n' not in source:
                continue
            with self.subTest(workflow=path.name):
                self.assertRegex(source,
                                 r'(?m)^permissions:\n  contents: read(?:\n|$)')
                self.assertNotRegex(source, r'(?m)^\s+[\w-]+: write\s*$')

    def test_queue_only_image_builds_do_not_publish(self):
        for queue_name, original_name in (
                ('build-tools-images-merge-group.yml',
                 'build-tools-images.yml'),
                ('runtime-base-images-merge-group.yml',
                 'runtime-base-images.yml'),
        ):
            with self.subTest(workflow=queue_name):
                queue_source = workflow(queue_name)
                original_source = workflow(original_name)
                self.assertIn('  merge_group:\n', queue_source)
                self.assertNotIn('  merge_group:\n', original_source)
                self.assertNotRegex(queue_source, r'(?m)^  push:')
                self.assertNotIn('docker push', queue_source)
                self.assertNotIn('docker/login-action', queue_source)
                self.assertNotIn('actions/cache/save', queue_source)
                self.assertNotIn('gh api --method DELETE', queue_source)

    def test_queue_runtime_artifact_matches_image_consumer(self):
        queue_source = workflow('runtime-base-images-merge-group.yml')
        original_source = workflow('runtime-base-images.yml')
        image_builds_source = workflow('image-builds.yml')
        fingerprint_pattern = re.compile(
            r'archive_fingerprint=\$\(python3 \\\n'
            r'.*?\)', re.DOTALL)
        queue_fingerprint = fingerprint_pattern.search(queue_source)
        original_fingerprint = fingerprint_pattern.search(original_source)
        self.assertIsNotNone(queue_fingerprint)
        self.assertIsNotNone(original_fingerprint)
        self.assertEqual(queue_fingerprint.group(), original_fingerprint.group())
        self.assertIn(
            'name: ${{ steps.configure-runtime-base-images.outputs.artifact-name }}',
            queue_source)
        self.assertIn('SOURCE_SHA: ${{ github.event.pull_request.head.sha || github.sha }}',
                      image_builds_source)

    def test_sdk_tests_install_from_the_queued_commit(self):
        queued_sha = '0123456789abcdef0123456789abcdef01234567'
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            capture_path = temp_path / 'package-path'
            fake_python = temp_path / 'python'
            fake_python.write_text(
                '#!/bin/sh\nprintf "%s" "$KFP_PACKAGE_PATH" > "$KFP_PACKAGE_CAPTURE"\n',
                encoding='utf-8')
            fake_python.chmod(0o755)

            for script_name in ('presubmit-tests-sdk.sh',
                                'presubmit-tests-sdk-unit.sh'):
                for event_name, expected_ref in (
                        ('merge_group', queued_sha),
                        ('pull_request', 'refs/pull/123/merge'),
                ):
                    with self.subTest(script=script_name, event=event_name):
                        env = os.environ.copy()
                        env.update({
                            'SETUP_ENV': 'false',
                            'REPO_NAME': 'kubeflow/pipelines',
                            'PULL_NUMBER': '123',
                            'GITHUB_EVENT_NAME': event_name,
                            'GITHUB_SHA': queued_sha,
                            'KFP_PACKAGE_CAPTURE': str(capture_path),
                            'PATH': f'{temp_dir}{os.pathsep}{env["PATH"]}',
                        })
                        result = subprocess.run(
                            ['bash', str(REPOSITORY_ROOT / 'test' /
                                         script_name)],
                            cwd=REPOSITORY_ROOT,
                            env=env,
                            text=True,
                            capture_output=True,
                            check=False,
                        )
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual(
                            capture_path.read_text(encoding='utf-8'),
                            'git+https://github.com/kubeflow/pipelines@'
                            f'{expected_ref}#egg=kfp&subdirectory=sdk/python')


if __name__ == '__main__':
    unittest.main()
