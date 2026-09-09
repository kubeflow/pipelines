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
"""Unit tests for synchronizing Argo CI and documentation versions."""

import importlib.util
from pathlib import Path
import tempfile
import unittest

SCRIPT_PATH = Path(__file__).with_name('sync_argo_versions.py')
SPEC = importlib.util.spec_from_file_location('sync_argo_versions', SCRIPT_PATH)
sync_argo_versions = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(sync_argo_versions)


class SyncArgoVersionsTest(unittest.TestCase):

    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.repo_root = Path(self.temporary_directory.name)
        files = {
            'third_party/argo/VERSION': 'v4.1.2\n',
            # Promote the previous current release into the compatibility slot
            # to verify replacements do not cascade into the new current one.
            'third_party/argo/COMPATIBILITY_VERSION': 'v4.0.5\n',
            '.github/workflows/e2e-test.yml':
                ('        argo_version: ["v3.7.14", "v4.0.5"]\n'),
            '.github/workflows/api-server-tests.yml': (
                '        argo_version: ["v3.7.14", "v4.0.5"]\n'
                "          ARGO_COMPATIBILITY_TESTS: ${{ matrix.argo_version == 'v4.0.5' }}\n"
            ),
            '.github/resources/runtime-base-images.txt':
                ('quay.io/argoproj/workflow-controller:v3.7.14\n'
                 'quay.io/argoproj/argoexec:v4.0.5\n'),
            'AGENTS.md': 'Argo v3.7.14 and v4.0.5\n',
        }
        for relative_path, contents in files.items():
            path = self.repo_root / relative_path
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(contents, encoding='utf-8')

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_sync_updates_compatibility_and_current_references(self):
        changed_paths = sync_argo_versions.sync(self.repo_root)

        self.assertEqual(len(changed_paths), 4)
        for relative_path in sync_argo_versions.CI_REFERENCE_PATHS:
            contents = (self.repo_root /
                        relative_path).read_text(encoding='utf-8')
            self.assertNotIn('v3.7.14', contents)
        workflow = (self.repo_root /
                    '.github/workflows/e2e-test.yml').read_text(
                        encoding='utf-8')
        self.assertIn('v4.0.5', workflow)
        self.assertIn('v4.1.2', workflow)
        self.assertIn(
            'Argo v4.0.5 and v4.1.2',
            (self.repo_root / 'AGENTS.md').read_text(encoding='utf-8'),
        )

        self.assertEqual(
            sync_argo_versions.sync(self.repo_root, check=True), [])

    def test_check_reports_changes_without_writing(self):
        changed_paths = sync_argo_versions.sync(self.repo_root, check=True)

        self.assertEqual(len(changed_paths), 4)
        self.assertIn(
            'v4.0.5',
            (self.repo_root /
             '.github/workflows/e2e-test.yml').read_text(encoding='utf-8'),
        )


if __name__ == '__main__':
    unittest.main()
