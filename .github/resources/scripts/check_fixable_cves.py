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
"""Checks fixable CVEs in a Trivy report with an explicit findings override."""

import argparse
import json
import sys

BLOCKING_SEVERITIES = {"UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL"}


def find_blocking_cves(report):
    """Returns unique fixable CVE findings."""
    if not isinstance(report, dict) or report.get("SchemaVersion") != 2:
        raise ValueError("Expected a Trivy JSON report with SchemaVersion 2")

    results = report.get("Results")
    if results is not None and not isinstance(results, list):
        raise ValueError("Results must be an array or null")
    findings = {}
    for result in results or []:
        if not isinstance(result, dict):
            raise ValueError("Each result must be an object")
        target = result.get("Target", "unknown")
        if not isinstance(target, str):
            raise ValueError("Result Target must be a string")
        vulnerabilities = result.get("Vulnerabilities")
        if vulnerabilities is not None and not isinstance(
                vulnerabilities, list):
            raise ValueError("Vulnerabilities must be an array or null")
        for vulnerability in vulnerabilities or []:
            if not isinstance(vulnerability, dict):
                raise ValueError("Each vulnerability must be an object")
            for field in ("VulnerabilityID", "FixedVersion", "Severity",
                          "PkgName", "InstalledVersion"):
                if not isinstance(vulnerability.get(field, ""), str):
                    raise ValueError(f"Vulnerability {field} must be a string")
            vulnerability_id = vulnerability.get("VulnerabilityID", "")
            fixed_version = vulnerability.get("FixedVersion", "")
            severity = vulnerability.get("Severity", "").upper()
            if not vulnerability_id or severity not in BLOCKING_SEVERITIES:
                raise ValueError(
                    "Vulnerability must have an ID and valid severity")
            if not vulnerability_id.startswith("CVE-") or not fixed_version:
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
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", help="Path to the Trivy JSON report")
    parser.add_argument(
        "--allow-fixable-cves",
        action="store_true",
        help="Allow fixable CVE findings; invalid reports still fail",
    )
    args = parser.parse_args(argv[1:])

    try:
        with open(args.report, encoding="utf-8") as report_file:
            report = json.load(report_file)
        findings = find_blocking_cves(report)
    except (OSError, ValueError) as error:
        print(
            f"ERROR: Cannot read valid Trivy report: {error}", file=sys.stderr)
        return 2

    if not findings:
        print("PASS: no fixable CVEs found")
        return 0

    if args.allow_fixable_cves:
        print(
            f"WARNING: allowing {len(findings)} fixable CVE(s) because "
            "--allow-fixable-cves was explicitly set.",
            file=sys.stderr,
        )
    else:
        print(f"FAIL: found {len(findings)} fixable CVE(s).", file=sys.stderr)
    print(
        "Target | CVE | Package | Installed | Fixed | Severity",
        file=sys.stderr,
    )
    for target, cve, package, installed, fixed, severity in findings:
        print(
            f"{target} | {cve} | {package} | {installed} | {fixed} | {severity}",
            file=sys.stderr,
        )
    return 0 if args.allow_fixable_cves else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
