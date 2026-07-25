#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Unit tests for check_fixable_cves.py (stdlib only)."""

import unittest

import check_fixable_cves


def report(*vulnerabilities):
    return {
        "Results": [
            {
                "Target": "test-image",
                "Vulnerabilities": list(vulnerabilities),
            }
        ]
    }


def vulnerability(
    vulnerability_id="CVE-2026-12345",
    severity="HIGH",
    fixed_version="2.0.0",
):
    return {
        "VulnerabilityID": vulnerability_id,
        "PkgName": "example",
        "InstalledVersion": "1.0.0",
        "FixedVersion": fixed_version,
        "Severity": severity,
    }


class FindBlockingCvesTest(unittest.TestCase):
    def test_fixable_cves_of_every_severity_block(self):
        findings = check_fixable_cves.find_blocking_cves(
            report(
                vulnerability(severity="UNKNOWN"),
                vulnerability(
                    vulnerability_id="CVE-2026-23456",
                    severity="LOW",
                ),
                vulnerability(
                    vulnerability_id="CVE-2026-34567",
                    severity="MEDIUM",
                ),
                vulnerability(severity="HIGH"),
                vulnerability(
                    vulnerability_id="CVE-2026-67890",
                    severity="CRITICAL",
                ),
            )
        )

        self.assertEqual(len(findings), 5)

    def test_unfixed_cve_does_not_block(self):
        findings = check_fixable_cves.find_blocking_cves(
            report(vulnerability(fixed_version=""))
        )

        self.assertEqual(findings, [])

    def test_non_cve_advisory_does_not_block(self):
        findings = check_fixable_cves.find_blocking_cves(
            report(vulnerability(vulnerability_id="GHSA-abcd-1234-5678"))
        )

        self.assertEqual(findings, [])

    def test_duplicate_findings_are_reported_once(self):
        duplicate = vulnerability()
        findings = check_fixable_cves.find_blocking_cves(
            report(duplicate, duplicate.copy())
        )

        self.assertEqual(len(findings), 1)


if __name__ == "__main__":
    unittest.main()
