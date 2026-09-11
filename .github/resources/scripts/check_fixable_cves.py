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
"""Fails when a Trivy JSON report contains a fixable CVE."""

import json
import sys

BLOCKING_SEVERITIES = {"UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL"}


def find_blocking_cves(report):
    """Returns unique fixable CVE findings."""
    findings = {}
    for result in report.get("Results") or []:
        target = result.get("Target", "unknown")
        for vulnerability in result.get("Vulnerabilities") or []:
            vulnerability_id = vulnerability.get("VulnerabilityID", "")
            fixed_version = vulnerability.get("FixedVersion", "")
            severity = vulnerability.get("Severity", "").upper()
            if (
                not vulnerability_id.startswith("CVE-")
                or not fixed_version
                or severity not in BLOCKING_SEVERITIES
            ):
                continue

            finding = (
                target,
                vulnerability_id,
                vulnerability.get("PkgName", "unknown"),
                vulnerability.get("InstalledVersion", "unknown"),
                fixed_version,
                severity,
            )
            findings[finding] = finding
    return sorted(findings.values())


def main(argv):
    if len(argv) != 2:
        print(f"Usage: {argv[0]} <trivy-results.json>", file=sys.stderr)
        return 2

    try:
        with open(argv[1], encoding="utf-8") as report_file:
            report = json.load(report_file)
    except (OSError, json.JSONDecodeError) as error:
        print(f"ERROR: Cannot read Trivy report: {error}", file=sys.stderr)
        return 2

    findings = find_blocking_cves(report)
    if not findings:
        print("PASS: no fixable CVEs found")
        return 0

    print(
        f"FAIL: found {len(findings)} fixable CVE(s).",
        file=sys.stderr,
    )
    print(
        "Target | CVE | Package | Installed | Fixed | Severity",
        file=sys.stderr,
    )
    for target, cve, package, installed, fixed, severity in findings:
        print(
            f"{target} | {cve} | {package} | {installed} | {fixed} | {severity}",
            file=sys.stderr,
        )
    return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
