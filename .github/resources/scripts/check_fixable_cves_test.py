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
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest

import check_fixable_cves


def report(*vulnerabilities, name="example", ecosystem="npm", version="1.0.0"):
    return {
        "results": [{
            "source": {
                "path": "/app/package-lock.json",
                "type": "artifact"
            },
            "packages": [{
                "package": {
                    "name": name,
                    "ecosystem": ecosystem,
                    "version": version
                },
                "vulnerabilities": list(vulnerabilities),
            }],
        }],
        "experimental_config": {
            "licenses": {
                "summary": False,
                "allowlist": None
            }
        },
        "image_metadata": {
            "os": "Debian GNU/Linux 12 (bookworm)",
            "layer_metadata": [],
            "base_images": []
        },
    }


def vulnerability(advisory_id="CVE-2026-12345",
                  aliases=(),
                  fixed_version="2.0.0",
                  name="example",
                  ecosystem="npm"):
    events = [{"introduced": "0"}]
    if fixed_version:
        events.append({"fixed": fixed_version})
    return {
        "id":
            advisory_id,
        "aliases":
            list(aliases),
        "affected": [{
            "package": {
                "name": name,
                "ecosystem": ecosystem
            },
            "ranges": [{
                "type": "ECOSYSTEM",
                "events": events
            }],
        }],
    }


def package_entry(value):
    return value["results"][0]["packages"][0]


class FindBlockingCvesTest(unittest.TestCase):

    def test_osv_go_advisory_fixture_matches_cve_alias_and_fixed_version(self):
        # Consumed fields from OSV-Scanner v2.5.0's GO-2021-0053 fixture:
        # internal/sourceanalysis/testdata/go-integration/GO-2021-0053.json.
        advisory = {
            "id":
                "GO-2021-0053",
            "modified":
                "2023-02-10T16:51:38Z",
            "aliases": ["CVE-2021-3121", "GHSA-c3h9-896r-86jm"],
            "affected": [{
                "package": {
                    "name": "github.com/gogo/protobuf",
                    "ecosystem": "Go"
                },
                "ranges": [{
                    "type": "SEMVER",
                    "events": [{
                        "introduced": "0"
                    }, {
                        "fixed": "1.3.2"
                    }]
                }],
            }],
        }
        findings = check_fixable_cves.find_blocking_cves(
            report(
                advisory,
                name="github.com/gogo/protobuf",
                ecosystem="Go",
                version="1.3.1"))
        self.assertEqual(findings,
                         [("/app/package-lock.json", "CVE-2021-3121", "Go",
                           "github.com/gogo/protobuf", "1.3.1", "1.3.2")])

    def test_fixable_cve_id_blocks_without_severity(self):
        self.assertEqual(
            len(check_fixable_cves.find_blocking_cves(report(vulnerability()))),
            1)

    def test_severity_does_not_exclude_findings(self):
        for severity in ("LOW", "MEDIUM", "HIGH", "CRITICAL", "UNKNOWN"):
            with self.subTest(severity=severity):
                advisory = vulnerability()
                advisory["database_specific"] = {"severity": severity}
                self.assertEqual(
                    len(
                        check_fixable_cves.find_blocking_cves(
                            report(advisory))), 1)

    def test_unfixed_cve_does_not_block(self):
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(
                report(vulnerability(fixed_version=""))), [])

    def test_non_cve_advisory_does_not_block(self):
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(
                report(vulnerability(advisory_id="GHSA-abcd-1234-5678"))), [])

    def test_fixed_version_for_different_package_or_ecosystem_does_not_block(
            self):
        for name, ecosystem in (("other", "npm"), ("example", "PyPI")):
            with self.subTest(name=name, ecosystem=ecosystem):
                self.assertEqual(
                    check_fixable_cves.find_blocking_cves(
                        report(vulnerability(name=name, ecosystem=ecosystem))),
                    [])

    def test_versioned_ecosystem_matching_follows_osv_scanner(self):
        advisory = vulnerability(ecosystem="Debian:12")
        for ecosystem, blocks in (("Debian:12", True), ("Debian", True),
                                  ("Debian:11", False), ("Ubuntu:12", False)):
            with self.subTest(ecosystem=ecosystem):
                findings = check_fixable_cves.find_blocking_cves(
                    report(advisory, ecosystem=ecosystem))
                self.assertEqual(bool(findings), blocks)
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(
                report(
                    vulnerability(ecosystem="Debian"), ecosystem="Debian:12")),
            [])

    def test_os_package_name_is_not_used_to_match_source_package(self):
        value = report(
            vulnerability(name="openssl", ecosystem="Debian:12"),
            name="openssl",
            ecosystem="Debian:12")
        package_entry(value)["package"]["os_package_name"] = "libssl3"
        self.assertEqual(len(check_fixable_cves.find_blocking_cves(value)), 1)

    def test_multiple_affected_packages_do_not_borrow_another_packages_fix(
            self):
        advisory = vulnerability(fixed_version="")
        advisory["affected"].extend(vulnerability(name="other")["affected"])
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(report(advisory)), [])

    def test_duplicate_cve_aliases_merge_fixed_versions(self):
        first = vulnerability(
            advisory_id="GHSA-abcd-1234-5678", aliases=("CVE-2026-12345",))
        second = vulnerability(fixed_version="3.0.0")
        findings = check_fixable_cves.find_blocking_cves(
            report(first, first, second))
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0][-1], "2.0.0, 3.0.0")

    def test_transitive_group_aliases_link_cve_to_same_advisory_group(self):
        value = report(vulnerability(advisory_id="GO-2026-0001"))
        package_entry(value)["groups"] = [{
            "ids": ["GO-2026-0001", "GHSA-abcd-1234-5678"],
            "aliases": [
                "CVE-2026-12345", "GO-2026-0001", "GHSA-abcd-1234-5678"
            ],
            "max_severity": "",
        }]
        findings = check_fixable_cves.find_blocking_cves(value)
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0][1], "CVE-2026-12345")
        package_entry(value)["groups"][0]["ids"] = ["unrelated-advisory"]
        self.assertEqual(check_fixable_cves.find_blocking_cves(value), [])


class MainTest(unittest.TestCase):

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.report_path = Path(temporary.name) / "osv-results.json"

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
        self.assertIn("CVE-2026-12345 | npm | example | 1.0.0 | 2.0.0", stderr)

    def test_override_warns_and_lists_allowed_findings(self):
        status, _, stderr = self.run_policy(
            json.dumps(report(vulnerability())), True)
        self.assertEqual(status, 0)
        self.assertIn("WARNING: allowing 1 fixable CVE(s)", stderr)
        self.assertIn("--allow-fixable-cves was explicitly set", stderr)
        self.assertIn("CVE-2026-12345 | npm | example | 1.0.0 | 2.0.0", stderr)

    def test_clean_reports_pass_with_and_without_override(self):
        omitted_vulnerabilities = report()
        del package_entry(omitted_vulnerabilities)["vulnerabilities"]
        empty = report()
        empty["results"] = []
        for clean in (report(), empty, omitted_vulnerabilities):
            for allow in (False, True):
                with self.subTest(clean=clean, allow=allow):
                    status, stdout, stderr = self.run_policy(
                        json.dumps(clean), allow)
                    self.assertEqual(status, 0)
                    self.assertIn("PASS: no fixable CVEs found", stdout)
                    self.assertEqual(stderr, "")

    def test_missing_or_invalid_reports_fail_even_with_override(self):
        invalid_reports = [
            None, "not JSON", "null", "[]", "{}", '{"results": null}',
            '{"results": {}}', '{"results": [null]}', '{"results": [{}]}',
            json.dumps(report(None)),
            json.dumps(report({}))
        ]
        valid = report(vulnerability())
        paths = [
            ("results", 0, "source"),
            ("results", 0, "source", "path"),
            ("results", 0, "packages"),
            ("results", 0, "packages", 0, "package"),
            ("results", 0, "packages", 0, "package", "name"),
            ("results", 0, "packages", 0, "package", "ecosystem"),
            ("results", 0, "packages", 0, "package", "version"),
            ("results", 0, "packages", 0, "vulnerabilities"),
        ]
        vuln_path = ("results", 0, "packages", 0, "vulnerabilities", 0)
        paths.extend(
            vuln_path + (field,) for field in ("id", "aliases", "affected"))
        affected_path = vuln_path + ("affected", 0)
        paths.extend(affected_path + suffix
                     for suffix in (("package", "name"), ("package",
                                                          "ecosystem"),
                                    ("ranges",), ("ranges", 0, "events"),
                                    ("ranges", 0, "events", 1, "fixed")))
        for path in paths:
            malformed = copy.deepcopy(valid)
            target = malformed
            for key in path[:-1]:
                target = target[key]
            target[path[-1]] = 42
            invalid_reports.append(json.dumps(malformed))
        malformed = report(vulnerability())
        package_entry(malformed)["groups"] = [{
            "ids": ["CVE-2026-12345"],
            "aliases": {}
        }]
        invalid_reports.append(json.dumps(malformed))
        for contents in invalid_reports:
            for allow in (False, True):
                with self.subTest(contents=contents, allow=allow):
                    status, stdout, stderr = self.run_policy(contents, allow)
                    self.assertEqual(status, 2)
                    self.assertEqual(stdout, "")
                    self.assertIn("ERROR: Cannot read valid OSV-Scanner report",
                                  stderr)

    def test_non_utf8_report_fails_even_with_override(self):
        self.report_path.write_bytes(b"\xff")
        for allow in (False, True):
            with self.subTest(allow=allow):
                status, stdout, stderr = self.run_policy(None, allow)
                self.assertEqual(status, 2)
                self.assertEqual(stdout, "")
                self.assertIn("ERROR: Cannot read valid OSV-Scanner report",
                              stderr)

    def test_unreadable_report_fails_even_with_override(self):
        self.report_path.mkdir()
        for allow in (False, True):
            with self.subTest(allow=allow):
                status, stdout, stderr = self.run_policy(None, allow)
                self.assertEqual(status, 2)
                self.assertEqual(stdout, "")
                self.assertIn("ERROR: Cannot read valid OSV-Scanner report",
                              stderr)


if __name__ == "__main__":
    unittest.main()
