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

import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest

import check_fixable_cves


def report(*vulnerabilities):
    return {
        "SchemaVersion":
            2,
        "Results": [{
            "Target": "test-image",
            "Vulnerabilities": list(vulnerabilities),
        }]
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
            ))

        self.assertEqual(len(findings), 5)

    def test_unfixed_cve_does_not_block(self):
        findings = check_fixable_cves.find_blocking_cves(
            report(vulnerability(fixed_version="")))

        self.assertEqual(findings, [])

    def test_non_cve_advisory_does_not_block(self):
        findings = check_fixable_cves.find_blocking_cves(
            report(vulnerability(vulnerability_id="GHSA-abcd-1234-5678")))

        self.assertEqual(findings, [])

    def test_duplicate_findings_are_reported_once(self):
        duplicate = vulnerability()
        findings = check_fixable_cves.find_blocking_cves(
            report(duplicate, duplicate.copy()))

        self.assertEqual(len(findings), 1)


class MainTest(unittest.TestCase):

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.report_path = Path(temporary.name) / "trivy-results.json"

    def run_policy(self, contents, allow=False):
        if contents is not None:
            self.report_path.write_text(contents, encoding="utf-8")
        argv = ["check_fixable_cves.py", str(self.report_path)]
        if allow:
            argv.append("--allow-fixable-cves")
        stdout, stderr = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(
                stderr):
            status = check_fixable_cves.main(argv)
        return status, stdout.getvalue(), stderr.getvalue()

    def test_fixable_cves_fail_by_default(self):
        status, _, stderr = self.run_policy(json.dumps(report(vulnerability())))
        self.assertEqual(status, 1)
        self.assertIn("FAIL: found 1 fixable CVE(s)", stderr)
        self.assertIn("CVE-2026-12345 | example | 1.0.0 | 2.0.0 | HIGH", stderr)

    def test_override_warns_and_lists_allowed_findings(self):
        status, _, stderr = self.run_policy(
            json.dumps(report(vulnerability())), allow=True)
        self.assertEqual(status, 0)
        self.assertIn("WARNING: allowing 1 fixable CVE(s)", stderr)
        self.assertIn("--allow-fixable-cves was explicitly set", stderr)
        self.assertIn("CVE-2026-12345 | example | 1.0.0 | 2.0.0 | HIGH", stderr)

    def test_clean_reports_pass_with_and_without_override(self):
        for clean in (report(), {
                "SchemaVersion": 2
        }, {
                "SchemaVersion": 2,
                "Results": None,
        }, {
                "SchemaVersion": 2,
                "Results": [{
                    "Target": "test-image"
                }],
        }):
            for allow in (False, True):
                with self.subTest(clean=clean, allow=allow):
                    status, stdout, stderr = self.run_policy(
                        json.dumps(clean), allow=allow)
                    self.assertEqual(status, 0)
                    self.assertIn("PASS: no fixable CVEs found", stdout)
                    self.assertEqual(stderr, "")

    def test_missing_or_invalid_reports_fail_even_with_override(self):
        invalid_reports = [
            None,
            "not JSON",
            "null",
            "[]",
            "{}",
            '{"SchemaVersion": 1}',
            '{"SchemaVersion": 2, "Results": {}}',
            '{"SchemaVersion": 2, "Results": [null]}',
            '{"SchemaVersion": 2, "Results": [{"Vulnerabilities": {}}]}',
            json.dumps(report(None)),
            json.dumps(report({})),
        ]
        for field in ("VulnerabilityID", "FixedVersion", "Severity", "PkgName",
                      "InstalledVersion"):
            malformed = vulnerability()
            malformed[field] = ["invalid"]
            invalid_reports.append(json.dumps(report(malformed)))
        for contents in invalid_reports:
            for allow in (False, True):
                with self.subTest(contents=contents, allow=allow):
                    status, stdout, stderr = self.run_policy(contents, allow)
                    self.assertEqual(status, 2)
                    self.assertEqual(stdout, "")
                    self.assertIn("ERROR: Cannot read valid Trivy report",
                                  stderr)

    def test_non_utf8_report_fails_even_with_override(self):
        self.report_path.write_bytes(b"\xff")
        for allow in (False, True):
            with self.subTest(allow=allow):
                status, stdout, stderr = self.run_policy(None, allow)
                self.assertEqual(status, 2)
                self.assertEqual(stdout, "")
                self.assertIn("ERROR: Cannot read valid Trivy report", stderr)

    def test_unreadable_report_fails_even_with_override(self):
        self.report_path.mkdir()
        for allow in (False, True):
            with self.subTest(allow=allow):
                status, stdout, stderr = self.run_policy(None, allow)
                self.assertEqual(status, 2)
                self.assertEqual(stdout, "")
                self.assertIn("ERROR: Cannot read valid Trivy report", stderr)


if __name__ == "__main__":
    unittest.main()
