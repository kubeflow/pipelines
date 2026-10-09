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
import html
import json
from pathlib import Path
import re
import sys
from urllib.parse import unquote


def _records(value, field):
    if not isinstance(value, list) or any(
            not isinstance(item, dict) for item in value):
        raise ValueError(f"{field} must be an array of objects")
    return value


def _strings(value, field, nullable=False):
    if value is None and nullable:
        return []
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


def _normalized_package(package):
    name, ecosystem = package["name"], package["ecosystem"]
    # Match OSV's PyPI name and Ubuntu variant normalization while retaining
    # the distribution release (e.g. Ubuntu:22.04 must not match Ubuntu:24.04).
    if ecosystem == "PyPI":
        name = re.sub(r"[-_.]+", "-", name).lower()
    if ecosystem.startswith("Ubuntu:"):
        ecosystem = ":".join(
            part for part in ecosystem.split(":") if part not in {"Pro", "LTS"})
    return name, ecosystem


def _affected_package(value):
    if not isinstance(value, dict):
        raise ValueError("Affected package must be an object")
    if "name" in value or "ecosystem" in value:
        return _package(value)
    purl = value.get("purl")
    if not isinstance(purl, str) or not purl.startswith("pkg:"):
        raise ValueError("Affected package must contain an identity or PURL")
    # Handle only unambiguous language PURLs. A distribution PURL may omit
    # its release; unknown identities remain unresolved instead of guessing.
    path = purl[4:].split("#", 1)[0].split("?", 1)[0]
    kind, separator, name = path.partition("/")
    ecosystem = {"pypi": "PyPI", "npm": "npm", "golang": "Go"}.get(kind)
    if not separator or not name:
        raise ValueError("Affected package PURL must contain a type and name")
    if not ecosystem:
        return None
    if "@" in name.rsplit("/", 1)[-1]:
        name = name.rsplit("@", 1)[0]
    name = unquote(name)
    if ecosystem == "PyPI" and "/" in name:
        return None
    return {"name": name, "ecosystem": ecosystem}


def _fixed_versions(vulnerability, package):
    fixed_versions = set()
    matched_package = False
    package_name, package_ecosystem = _normalized_package(package)
    affected_entries = vulnerability.get("affected")
    for affected in _records(
        [] if affected_entries is None else affected_entries, "affected"):
        affected_package = affected.get("package")
        if affected_package is None:
            continue
        affected_package = _affected_package(affected_package)
        # Match OSV-Scanner's GetFixedVersions: versioned ecosystems also map
        # to their unversioned name, never to another distribution version.
        matches = False
        if affected_package is not None:
            affected_name, affected_ecosystem = _normalized_package(
                affected_package)
            ecosystems = {affected_ecosystem}
            ecosystems.add(affected_ecosystem.split(":", 1)[0])
            matches = (
                affected_name == package_name and
                package_ecosystem in ecosystems)
            matched_package |= matches
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
    return fixed_versions, matched_package


def find_blocking_cves(report):
    """Returns CVEs with fixes or unresolved affected-package identities."""
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
                aliases = _strings(
                    vulnerability.get("aliases", []), "aliases", nullable=True)
                upstream = _strings(
                    vulnerability.get("upstream", []), "upstream")
                ids = {
                    advisory_id, *aliases, *upstream,
                    *group_aliases.get(advisory_id, [])
                }
                fixed, matched_package = _fixed_versions(vulnerability, package)
                if not matched_package:
                    fixed = {"UNDETERMINED: no matching affected package"}
                for cve in ids:
                    if not cve.startswith("CVE-") or not fixed:
                        continue
                    finding = (source["path"], cve, package["ecosystem"],
                               package["name"], package["version"])
                    findings.setdefault(finding, set()).update(fixed)
    return sorted(
        key + (", ".join(sorted(fixed)),) for key, fixed in findings.items())


def markdown(value):
    """Escape scanner-controlled text before placing it in Markdown tables."""
    value = html.escape(str(value), quote=True)
    for character in ("\\", "`", "*", "_", "[", "]", "|", "~"):
        value = value.replace(character, "\\" + character)
    return value.replace("\r", " ").replace("\n", " ")


def remediation_hint(finding):
    """Give an owner-oriented next step without claiming a verified fix."""
    ecosystem, package = finding["ecosystem"], finding["package"]
    if any(
            version.startswith("UNDETERMINED:")
            for version in finding["fixed_versions"]):
        return "Resolve the advisory's affected-package identity before choosing an update."
    if ecosystem == "Go" and package == "stdlib":
        return (
            "Update the Go compiler and rebuild if this is a KFP binary; "
            "an inherited binary needs its upstream image or package updated.")
    if ecosystem == "Go":
        return "Update the owning Go module dependency and rebuild its binary."
    if ecosystem == "npm":
        return "Update the owning package-lock.json dependency, rebuild, and rescan."
    if ecosystem.split(":", 1)[0] in {"Alpine", "Debian", "Ubuntu"}:
        return "Refresh the owning Dockerfile's base image or distribution package and rebuild."
    return "Locate the owning manifest or upstream image, update it, rebuild, and rescan."


def validate_result(result):
    """Validate the artifact contract before a remediation job trusts it."""
    if (not isinstance(result, dict) or
            type(result.get("schema_version")) is not int or
            result["schema_version"] != 1):
        raise ValueError("Expected CVE result schema_version 1")
    image = result.get("image")
    if not isinstance(image, str) or not re.fullmatch(
            r"[a-z0-9]+(?:[._-][a-z0-9]+)*", image):
        raise ValueError("image must be a bare image name")
    if result.get("platform") not in ("linux/amd64", "linux/arm64"):
        raise ValueError("platform must be linux/amd64 or linux/arm64")
    reference = result.get("image_ref")
    if not isinstance(reference, str) or not re.fullmatch(
            rf"[a-z0-9.-]+(?::[0-9]+)?/(?:[a-z0-9._-]+/)*{re.escape(image)}@sha256:[0-9a-f]{{64}}",
            reference):
        raise ValueError(
            "image_ref must identify the image by its full digest reference")
    sha = result.get("source_sha")
    if not isinstance(sha, str) or not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("source_sha must be a full commit SHA")
    findings = _records(result.get("findings"), "findings")
    for finding in findings:
        for key in ("target", "cve", "ecosystem", "package", "installed"):
            if not isinstance(finding.get(key), str) or not finding[key]:
                raise ValueError(f"Finding {key} must be a nonempty string")
        if not re.fullmatch(r"CVE-[0-9]{4}-[0-9]{4,}", finding["cve"]):
            raise ValueError("Finding cve must be a CVE ID")
        if not _strings(finding.get("fixed_versions"), "fixed_versions"):
            raise ValueError("Finding fixed_versions cannot be empty")
    outcome = result.get("outcome")
    if outcome not in ("pass", "blocked", "overridden") or ((outcome == "pass")
                                                            != (not findings)):
        raise ValueError("Result outcome must agree with its findings")
    return result


def structured_result(findings,
                      image,
                      platform,
                      image_ref,
                      source_sha,
                      allow=False):
    fields = ("target", "cve", "ecosystem", "package", "installed")
    result = {
        "schema_version":
            1,
        "image":
            image,
        "platform":
            platform,
        "image_ref":
            image_ref,
        "source_sha":
            source_sha,
        "outcome":
            "overridden"
            if findings and allow else "blocked" if findings else "pass",
        "findings": [
            dict(
                zip(fields, finding[:5]), fixed_versions=finding[5].split(", "))
            for finding in findings
        ],
    }
    return validate_result(result)


def write_summary(path, findings, image, platform, allow):
    heading = "CVE scan: " + ("overridden" if findings and allow else
                              "blocked" if findings else "pass")
    lines = [
        "### " + heading, "",
        markdown(f"{image or 'Image'} ({platform or 'platform not supplied'})"),
        ""
    ]
    if findings:
        lines += [
            "| Target | CVE | Ecosystem / package | Installed | Fixed versions | Next step |",
            "| --- | --- | --- | --- | --- | --- |",
        ]
        for target, cve, ecosystem, package, installed, fixed in findings:
            hint = remediation_hint(
                dict(
                    ecosystem=ecosystem,
                    package=package,
                    fixed_versions=fixed.split(", ")))
            lines.append("| " + " | ".join(
                markdown(value)
                for value in (target, cve, f"{ecosystem} / {package}",
                              installed, fixed, hint)) + " |")
        lines += [
            "",
            "Rebuild and rescan after updating the owning source. Release publication stays blocked unless findings are explicitly overridden."
        ]
    else:
        lines.append("No blocking CVE findings.")
    with open(path, "a", encoding="utf-8") as summary:
        summary.write("\n".join(lines) + "\n\n")


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", help="Path to the OSV-Scanner JSON report")
    parser.add_argument(
        "--allow-fixable-cves",
        action="store_true",
        help="Allow fixable CVE findings; invalid reports still fail",
    )
    parser.add_argument(
        "--result-output", help="Write the validated CVE result artifact")
    parser.add_argument(
        "--summary-output", help="Append actionable Markdown to this file")
    parser.add_argument("--image")
    parser.add_argument("--platform")
    parser.add_argument("--image-ref")
    parser.add_argument("--source-sha")
    args = parser.parse_args(argv[1:])

    try:
        if args.result_output:
            # Never leave stale evidence from an earlier successful invocation.
            Path(args.result_output).unlink(missing_ok=True)
        with open(args.report, encoding="utf-8") as report_file:
            report = json.load(report_file)
        findings = find_blocking_cves(report)
        if args.result_output:
            result = structured_result(findings, args.image, args.platform,
                                       args.image_ref, args.source_sha,
                                       args.allow_fixable_cves)
            Path(args.result_output).write_text(
                json.dumps(result, indent=2) + "\n", encoding="utf-8")
        if args.summary_output:
            write_summary(args.summary_output, findings, args.image,
                          args.platform, args.allow_fixable_cves)
    except (OSError, ValueError) as error:
        print(
            f"ERROR: Cannot read valid OSV-Scanner report: {error}",
            file=sys.stderr)
        return 2

    if not findings:
        print("PASS: no blocking CVE findings")
        return 0

    if args.allow_fixable_cves:
        print(
            f"WARNING: allowing {len(findings)} blocking CVE finding(s) because "
            "--allow-fixable-cves was explicitly set.",
            file=sys.stderr,
        )
    else:
        print(
            f"FAIL: found {len(findings)} blocking CVE finding(s).",
            file=sys.stderr)
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
