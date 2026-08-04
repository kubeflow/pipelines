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
WORKFLOW_PATH = ROOT / '.github/workflows/trivy.yml'
CI_SCRIPTS_WORKFLOW_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'


class TrivyWorkflowTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = WORKFLOW_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW_PATH.read_text(
            encoding='utf-8'
        )

    def test_scan_can_bootstrap_current_databases(self):
        self.assertIn(
            'TRIVY_DB_REPOSITORY: public.ecr.aws/aquasecurity/trivy-db:2',
            self.workflow,
        )
        self.assertIn(
            'TRIVY_JAVA_DB_REPOSITORY: public.ecr.aws/aquasecurity/trivy-java-db:1',
            self.workflow,
        )
        self.assertNotIn('TRIVY_SKIP_DB_UPDATE', self.workflow)
        self.assertNotIn('TRIVY_SKIP_JAVA_DB_UPDATE', self.workflow)

    def test_scan_uses_reviewed_pins_and_reports_all_vulnerabilities(self):
        self.assertIn(
            'aquasecurity/trivy-action@ed142fd0673e97e23eac54620cfb913e5ce36c25',
            self.workflow,
        )
        self.assertIn("version: 'v0.73.0'", self.workflow)
        self.assertIn("scanners: 'vuln'", self.workflow)
        self.assertIn(
            "severity: 'UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL'", self.workflow
        )
        self.assertNotIn('ignore-unfixed:', self.workflow)
        self.assertNotIn('limit-severities-for-sarif:', self.workflow)

    def test_scan_has_least_privilege_and_manual_recovery(self):
        self.assertIn('  workflow_dispatch:', self.workflow)
        self.assertIn('      contents: read', self.workflow)
        self.assertIn('      security-events: write', self.workflow)
        self.assertIn('          persist-credentials: false', self.workflow)

    def test_ci_scripts_tests_run_for_trivy_workflow_changes(self):
        self.assertIn(
            "      - '.github/workflows/trivy.yml'", self.ci_scripts_workflow
        )


if __name__ == '__main__':
    unittest.main()
