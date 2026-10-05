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

    def test_real_alpine_report_uses_group_alias_and_upstream_fallback(self):
        # Reduced from OSV-Scanner 2.5.0 scanning the Alpine 3.18.0 ARM64 image.
        value = json.loads((Path(__file__).parent / "testdata" /
                            "osv_alpine_3_18.json").read_text())
        expected = [("/lib/apk/db/installed", "CVE-2023-2650", "Alpine:v3.18",
                     "openssl", "3.1.0-r4", "3.1.1-r0")]
        self.assertEqual(check_fixable_cves.find_blocking_cves(value), expected)
        without_upstream = copy.deepcopy(value)
        del package_entry(without_upstream)["vulnerabilities"][0]["upstream"]
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(without_upstream), expected)
        del package_entry(value)["groups"]
        self.assertEqual(check_fixable_cves.find_blocking_cves(value), expected)

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

    def test_unmatched_package_or_ecosystem_is_undetermined_not_clean(self):
        for name, ecosystem in (("other", "npm"), ("example", "PyPI")):
            with self.subTest(name=name, ecosystem=ecosystem):
                findings = check_fixable_cves.find_blocking_cves(
                    report(vulnerability(name=name, ecosystem=ecosystem)))
                self.assertEqual(findings[0][-1],
                                 "UNDETERMINED: no matching affected package")

    def test_versioned_ecosystem_matching_follows_osv_scanner(self):
        advisory = vulnerability(ecosystem="Debian:12")
        for ecosystem, matches in (("Debian:12", True), ("Debian", True),
                                   ("Debian:11", False), ("Ubuntu:12", False)):
            with self.subTest(ecosystem=ecosystem):
                findings = check_fixable_cves.find_blocking_cves(
                    report(advisory, ecosystem=ecosystem))
                expected = "2.0.0" if matches else "UNDETERMINED: no matching affected package"
                self.assertEqual(findings[0][-1], expected)
        findings = check_fixable_cves.find_blocking_cves(
            report(vulnerability(ecosystem="Debian"), ecosystem="Debian:12"))
        self.assertTrue(findings[0][-1].startswith("UNDETERMINED:"))

    def test_pypi_names_follow_pep503_normalization(self):
        for scanned, affected in (("Django_Rest.Framework",
                                   "django-rest-framework"),
                                  ("django-rest-framework",
                                   "Django...Rest__Framework")):
            with self.subTest(scanned=scanned, affected=affected):
                findings = check_fixable_cves.find_blocking_cves(
                    report(
                        vulnerability(name=affected, ecosystem="PyPI"),
                        name=scanned,
                        ecosystem="PyPI"))
                self.assertEqual(findings[0][-1], "2.0.0")

    def test_ubuntu_variants_match_without_crossing_release_boundaries(self):
        for scanned, affected, matches in (("Ubuntu:22.04", "Ubuntu:22.04:LTS",
                                            True), ("Ubuntu:22.04:LTS",
                                                    "Ubuntu:22.04", True),
                                           ("Ubuntu:Pro:22.04:LTS",
                                            "Ubuntu:22.04", True),
                                           ("Ubuntu:22.04", "Ubuntu:24.04:LTS",
                                            False)):
            with self.subTest(scanned=scanned, affected=affected):
                findings = check_fixable_cves.find_blocking_cves(
                    report(
                        vulnerability(ecosystem=affected), ecosystem=scanned))
                expected = "2.0.0" if matches else "UNDETERMINED: no matching affected package"
                self.assertEqual(findings[0][-1], expected)

    def test_purl_only_language_package_can_supply_a_fix(self):
        for purl, name, ecosystem in (("pkg:pypi/Example_Package",
                                       "example-package", "PyPI"),
                                      ("pkg:npm/%40example/package@1.0.0",
                                       "@example/package", "npm"),
                                      ("pkg:golang/example.com/module",
                                       "example.com/module", "Go")):
            with self.subTest(purl=purl):
                advisory = vulnerability()
                advisory["affected"][0]["package"] = {"purl": purl}
                findings = check_fixable_cves.find_blocking_cves(
                    report(advisory, name=name, ecosystem=ecosystem))
                self.assertEqual(findings[0][-1], "2.0.0")

    def test_unrelated_purl_package_does_not_abort_or_supply_a_fix(self):
        advisory = vulnerability(fixed_version="")
        unrelated = vulnerability()["affected"][0]
        unrelated["package"] = {"purl": "pkg:deb/debian/other?arch=source"}
        advisory["affected"].append(unrelated)
        self.assertEqual(
            check_fixable_cves.find_blocking_cves(report(advisory)), [])

    def test_unsupported_purl_only_cve_is_undetermined(self):
        advisory = vulnerability()
        advisory["affected"][0]["package"] = {"purl": "pkg:deb/debian/example"}
        findings = check_fixable_cves.find_blocking_cves(report(advisory))
        self.assertTrue(findings[0][-1].startswith("UNDETERMINED:"))

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
        self.assertIn("FAIL: found 1 blocking CVE finding(s)", stderr)
        self.assertIn("CVE-2026-12345 | npm | example | 1.0.0 | 2.0.0", stderr)

    def test_override_warns_and_lists_allowed_findings(self):
        status, _, stderr = self.run_policy(
            json.dumps(report(vulnerability())), True)
        self.assertEqual(status, 0)
        self.assertIn("WARNING: allowing 1 blocking CVE finding(s)", stderr)
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
                    self.assertIn("PASS: no blocking CVE findings", stdout)
                    self.assertEqual(stderr, "")

    def test_nullable_aliases_and_unmatched_identity_preserve_override(self):
        for name in ("example", "unmatched"):
            advisory = vulnerability(name=name)
            advisory["aliases"] = None
            for allow in (False, True):
                with self.subTest(name=name, allow=allow):
                    status, stdout, stderr = self.run_policy(
                        json.dumps(report(advisory)), allow)
                    self.assertEqual(status, 0 if allow else 1)
                    self.assertEqual(stdout, "")
                    self.assertIn("blocking CVE finding(s)", stderr)
                    if name == "unmatched":
                        self.assertIn(
                            "UNDETERMINED: no matching affected package",
                            stderr)

    def test_nullable_affected_is_an_overrideable_unresolved_finding(self):
        for affected in (None, []):
            advisory = vulnerability()
            advisory["affected"] = affected
            for allow in (False, True):
                with self.subTest(affected=affected, allow=allow):
                    status, stdout, stderr = self.run_policy(
                        json.dumps(report(advisory)), allow)
                    self.assertEqual(status, 0 if allow else 1)
                    self.assertEqual(stdout, "")
                    self.assertIn("UNDETERMINED: no matching affected package",
                                  stderr)

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
        paths.extend(vuln_path + (field,)
                     for field in ("id", "aliases", "upstream", "affected"))
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
