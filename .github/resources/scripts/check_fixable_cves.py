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
"""Checks fixable CVEs in an OSV-Scanner report with a findings override."""

import argparse
import json
import sys


def _records(value, field):
    if not isinstance(value, list) or any(
            not isinstance(item, dict) for item in value):
        raise ValueError(f"{field} must be an array of objects")
    return value


def _strings(value, field):
    if not isinstance(value, list) or any(
            not isinstance(item, str) or not item for item in value):
        raise ValueError(f"{field} must be an array of nonempty strings")
    return value


def _package(value):
    if not isinstance(value, dict) or any(not isinstance(value.get(field), str)
                                          for field in ("name", "ecosystem")):
        raise ValueError(
            "Package must contain string name and ecosystem fields")
    return value


def _fixed_versions(vulnerability, package):
    fixed_versions = set()
    for affected in _records(vulnerability.get("affected", []), "affected"):
        affected_package = affected.get("package")
        if affected_package is None:
            continue
        _package(affected_package)
        # Match OSV-Scanner's GetFixedVersions: versioned ecosystems also map
        # to their unversioned name, never to another distribution version.
        ecosystems = {affected_package["ecosystem"]}
        ecosystems.add(affected_package["ecosystem"].split(":", 1)[0])
        matches = (
            affected_package["name"] == package["name"] and
            package["ecosystem"] in ecosystems)
        for affected_range in _records(affected.get("ranges", []), "ranges"):
            for event in _records(affected_range.get("events"), "events"):
                if not event or any(not isinstance(value, str) or not value
                                    for value in event.values()):
                    raise ValueError(
                        "Range events must contain nonempty strings")
                fixed = event.get("fixed", "")
                if not isinstance(fixed, str):
                    raise ValueError("Fixed version must be a string")
                if matches and fixed:
                    fixed_versions.add(fixed)
    return fixed_versions


def find_blocking_cves(report):
    """Returns unique fixable CVE findings."""
    if not isinstance(report, dict):
        raise ValueError("Expected an OSV-Scanner JSON report object")
    results = _records(report.get("results"), "results")
    findings = {}
    for result in results:
        source = result.get("source")
        if not isinstance(source, dict) or not isinstance(
                source.get("path"), str):
            raise ValueError("Source must contain a string path")
        for entry in _records(result.get("packages"), "packages"):
            package = _package(entry.get("package"))
            if not isinstance(package.get("version"), str):
                raise ValueError("Package version must be a string")
            group_aliases = {}
            for group in _records(entry.get("groups", []), "groups"):
                ids = _strings(group.get("ids"), "group ids")
                aliases = group.get("aliases", [])
                aliases = _strings([] if aliases is None else aliases,
                                   "group aliases")
                for advisory_id in ids:
                    group_aliases.setdefault(advisory_id,
                                             set()).update(ids + aliases)
            for vulnerability in _records(
                    entry.get("vulnerabilities", []), "vulnerabilities"):
                advisory_id = vulnerability.get("id")
                if not isinstance(advisory_id, str) or not advisory_id:
                    raise ValueError(
                        "Vulnerability ID must be a nonempty string")
                aliases = _strings(vulnerability.get("aliases", []), "aliases")
                ids = {
                    advisory_id, *aliases, *group_aliases.get(advisory_id, [])
                }
                fixed = _fixed_versions(vulnerability, package)
                for cve in ids:
                    if not cve.startswith("CVE-") or not fixed:
                        continue
                    finding = (source["path"], cve, package["ecosystem"],
                               package["name"], package["version"])
                    findings.setdefault(finding, set()).update(fixed)
    return sorted(
        key + (", ".join(sorted(fixed)),) for key, fixed in findings.items())


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", help="Path to the OSV-Scanner JSON report")
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
            f"ERROR: Cannot read valid OSV-Scanner report: {error}",
            file=sys.stderr)
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
        "Target | CVE | Ecosystem | Package | Installed | Fixed",
        file=sys.stderr,
    )
    for target, cve, ecosystem, package, installed, fixed in findings:
        print(
            f"{target} | {cve} | {ecosystem} | {package} | {installed} | {fixed}",
            file=sys.stderr,
        )
    return 0 if args.allow_fixable_cves else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
