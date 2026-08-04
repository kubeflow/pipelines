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

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_PATH = ROOT / '.github/workflows/osv-scanner.yml'
CI_SCRIPTS_WORKFLOW_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'


class OsvScannerWorkflowTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = WORKFLOW_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW_PATH.read_text(
            encoding='utf-8')

    def test_scanner_release_is_pinned_and_checksum_verified(self):
        self.assertIn("OSV_SCANNER_VERSION: '2.3.8'", self.workflow)
        self.assertIn(
            "OSV_SCANNER_SHA256: 'bc98e15319ed0d515e3f9235287ba53cdc5535d576d24fd573978ecfe9ab92dc'",
            self.workflow,
        )
        self.assertIn('sha256sum --check --strict', self.workflow)

    def test_scan_recursively_reports_all_supported_dependencies(self):
        self.assertIn('./osv-scanner scan source', self.workflow)
        self.assertIn('            --recursive', self.workflow)
        self.assertIn('            --no-resolve', self.workflow)
        self.assertIn('            --format sarif', self.workflow)
        self.assertIn('            --output-file osv-results.sarif',
                      self.workflow)
        self.assertIn('            . || scan_exit_code=$?', self.workflow)
        self.assertIn(
            '"${scan_exit_code}" -ne 0 && "${scan_exit_code}" -ne 1',
            self.workflow,
        )

    def test_scan_has_least_privilege_and_manual_dispatch(self):
        self.assertIn('  workflow_dispatch:', self.workflow)
        self.assertIn('  contents: read', self.workflow)
        self.assertIn('  security-events: write', self.workflow)
        self.assertIn('          persist-credentials: false', self.workflow)
        self.assertNotIn('contents: write', self.workflow)
        self.assertNotIn('pull-requests: write', self.workflow)

    def test_ci_scripts_tests_run_for_osv_workflow_changes(self):
        self.assertIn(
            "      - '.github/workflows/osv-scanner.yml'",
            self.ci_scripts_workflow,
        )


if __name__ == '__main__':
    unittest.main()
