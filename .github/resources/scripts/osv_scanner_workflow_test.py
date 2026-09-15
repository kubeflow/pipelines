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
        self.assertEqual(self.workflow.count("OSV_SCANNER_VERSION: '2.5.0'"),
                         2)
        scanner_checksum = (
            "OSV_SCANNER_SHA256: 'edcfc41d257db36148f065055655fe3f"
            "cfc434b0b423ea67468a84c207524e0c'")
        self.assertEqual(self.workflow.count(scanner_checksum), 2)
        self.assertIn('sha256sum --check --strict', self.workflow)

    def test_scan_recursively_reports_all_supported_dependencies(self):
        self.assertIn('./osv-scanner scan source', self.workflow)
        self.assertIn('            --recursive', self.workflow)
        self.assertIn('            --no-resolve', self.workflow)
        self.assertIn(
            '            --experimental-exclude backend/api/v1beta1/python_http_client',
            self.workflow,
        )
        self.assertIn(
            '            --experimental-exclude backend/api/v2beta1/python_http_client',
            self.workflow,
        )
        self.assertIn('            --format sarif', self.workflow)
        self.assertIn('            --output-file osv-results.sarif',
                      self.workflow)
        self.assertIn('            . || scan_exit_code=$?', self.workflow)
        self.assertIn(
            '"${scan_exit_code}" -ne 0 && "${scan_exit_code}" -ne 1',
            self.workflow,
        )

    def test_scan_has_least_privilege_and_manual_dispatch(self):
        self.assertIn('  push:', self.workflow)
        self.assertIn('      - master', self.workflow)
        self.assertIn('  workflow_dispatch:', self.workflow)
        self.assertIn('  contents: read', self.workflow)
        self.assertIn('  security-events: write', self.workflow)
        self.assertEqual(self.workflow.count('      security-events: write'), 2)
        self.assertIn('          persist-credentials: false', self.workflow)
        self.assertNotIn('contents: write', self.workflow)
        self.assertNotIn('pull-requests: write', self.workflow)

    def test_deployed_images_are_discovered_and_scanned(self):
        self.assertIn("KUSTOMIZE_VERSION: '5.8.1'", self.workflow)
        self.assertIn(
            "KUSTOMIZE_SHA256: '029a7f0f4e1932c52a0476cf02a0fd855c0bb85694b82c338fc648dcb53a819d'",
            self.workflow,
        )
        self.assertIn('osv_manifest_images.py', self.workflow)
        self.assertIn('--overlay manifests/kustomize/env/platform-agnostic',
                      self.workflow)
        self.assertIn(
            '--overlay manifests/kustomize/env/platform-agnostic-multi-user',
            self.workflow,
        )
        self.assertIn('./osv-scanner scan image "${IMAGE}"', self.workflow)
        self.assertIn('docker-pull-with-retry.sh "${IMAGE}"', self.workflow)
        self.assertIn(
            'category: kubeflow-pipelines-osv-image-${{ matrix.category }}',
            self.workflow)
        self.assertIn('      fail-fast: false', self.workflow)

    def test_exact_duplicate_results_are_removed_before_upload(self):
        self.assertEqual(self.workflow.count('deduplicate_sarif.py'), 2)
        self.assertIn('            --input osv-results.sarif', self.workflow)
        self.assertIn('            --output osv-results-deduplicated.sarif',
                      self.workflow)
        self.assertIn('            --input osv-image-results.sarif',
                      self.workflow)
        self.assertIn(
            '            --output osv-image-results-deduplicated.sarif',
            self.workflow,
        )
        self.assertIn('          sarif_file: osv-results-deduplicated.sarif',
                      self.workflow)
        self.assertIn(
            '          sarif_file: osv-image-results-deduplicated.sarif',
            self.workflow,
        )

    def test_ci_scripts_tests_run_for_osv_workflow_changes(self):
        self.assertIn(
            "      - '.github/workflows/osv-scanner.yml'",
            self.ci_scripts_workflow,
        )


if __name__ == '__main__':
    unittest.main()
