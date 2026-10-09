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
"""Map complete release CVE evidence to conservative, verifiable source
fixes."""

import argparse
import json
from pathlib import Path
import re

from arm64_smoke import IMAGES
from check_fixable_cves import markdown
from check_fixable_cves import remediation_hint
from check_fixable_cves import validate_result

# These binaries are built from the root Go module by these Dockerfiles.
GO_IMAGES = {
    "kfp-api-server": ("/bin/apiserver", "backend/Dockerfile"),
    "kfp-driver": ("/bin/driver", "backend/Dockerfile.driver"),
    "kfp-launcher": ("/bin/launcher-v2", "backend/Dockerfile.launcher"),
    "kfp-persistence-agent":
        ("/bin/persistence_agent", "backend/Dockerfile.persistenceagent"),
    "kfp-scheduled-workflow-controller":
        ("/bin/controller", "backend/Dockerfile.scheduledworkflow"),
    "kfp-viewer-crd-controller":
        ("/bin/controller", "backend/Dockerfile.viewercontroller"),
}
NPM_TARGETS = {
    "/server/package-lock.json", "/server/node_modules/.package-lock.json"
}
PLATFORMS = {"linux/amd64", "linux/arm64"}


def source_file(root, relative):
    """Do not follow source-controlled symlinks when reading ownership data."""
    root = Path(root).absolute()
    path = root / relative
    if not path.is_relative_to(root):
        raise ValueError("Source path must remain inside the checkout")
    if any(part.is_symlink() for part in (path, *path.parents)):
        raise ValueError(f"Source path must not be a symlink: {relative}")
    if not path.is_file():
        raise ValueError(f"Missing source file: {relative}")
    return path


def load_reports(directory, source_sha):
    """Require current-source evidence for both architectures of every
    image."""
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha):
        raise ValueError("source_sha must be a full commit SHA")
    root = Path(directory).absolute()
    reports = {}
    for path in sorted(root.glob("cve-result-*.json")):
        if any(part.is_symlink() for part in (path, *path.parents)):
            raise ValueError("CVE reports must not be symlinks")
        report = validate_result(json.loads(path.read_text(encoding="utf-8")))
        if report["source_sha"] != source_sha:
            raise ValueError(
                "CVE report source SHA does not match the release source")
        key = report["image"], report["platform"]
        if report["image"] not in IMAGES:
            raise ValueError(f"Unexpected image: {report['image']}")
        if key in reports and reports[key] != report:
            raise ValueError(f"Conflicting duplicate CVE reports for {key}")
        reports[key] = report
    expected = {(image, platform) for image in IMAGES for platform in PLATFORMS}
    if set(reports) != expected:
        raise ValueError(
            f"Missing CVE reports: {sorted(expected - set(reports))}")
    return [reports[key] for key in sorted(reports)]


def stable_go_version(value):
    match = re.fullmatch(r"(?:go)?([0-9]+)\.([0-9]+)\.([0-9]+)", value)
    return tuple(map(int, match.groups())) if match else None


def has_same_major_npm_fix(installed, fixed_versions):
    """Match the preparer's no-major-upgrades boundary conservatively."""

    def stable_version(value):
        match = re.fullmatch(
            r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", value)
        return tuple(map(int, match.groups())) if match else None

    current = stable_version(installed)
    if current is None:
        return False
    fixed = (stable_version(value) for value in fixed_versions)
    return any(version and version[0] == current[0] and version > current
               for version in fixed)


def compiler_version(source_root):
    contents = source_file(source_root, "go.mod").read_text(encoding="utf-8")
    go = re.search(r"(?m)^go ([0-9]+\.[0-9]+(?:\.[0-9]+)?)\s*$", contents)
    toolchain = re.search(r"(?m)^toolchain (\S+)\s*$", contents)
    if not go:
        raise ValueError("Cannot determine the root Go compiler version")
    value = go[1] if not toolchain or toolchain[1] == "default" else toolchain[1]
    if value.count(".") == 1:
        value += ".0"
    version = stable_go_version(value)
    if not version:
        raise ValueError("Automatic Go fixes require a stable compiler version")
    return version


def npm_packages(source_root):
    lock = json.loads(
        source_file(
            source_root,
            "frontend/server/package-lock.json").read_text(encoding="utf-8"))
    packages = lock.get("packages")
    if not isinstance(packages, dict):
        raise ValueError(
            "Automatic npm fixes require a package-lock packages inventory")
    return {(path.rsplit("node_modules/", 1)[-1], package.get("version"))
            for path, package in packages.items()
            if "node_modules/" in path and isinstance(package, dict)}


def finding_key(image, finding):
    return (image, *(finding[field]
                     for field in ("target", "cve", "ecosystem", "package")))


def plan_remediation(reports_dir, source_root, source_sha, source_branch,
                     run_url):
    """Select only supported fixes; preserve all other findings for review."""
    if (not isinstance(source_branch, str) or
            not re.fullmatch(r"[A-Za-z0-9_./-]+", source_branch) or
            source_branch.startswith(("-", "/")) or source_branch.endswith(
                ("/", ".", ".lock")) or
            any(part in source_branch for part in ("..", "//"))):
        raise ValueError("source_branch must be a branch name")
    if not re.fullmatch(
            r"https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/actions/runs/[0-9]+(?:/attempts/[0-9]+)?",
            run_url):
        raise ValueError("run_url must be a GitHub Actions run URL")
    reports = load_reports(reports_dir, source_sha)
    selected = {}
    go_fixes = []
    npm_vulns = set()
    current_go, packages = None, None
    for report in reports:
        if report["outcome"] != "blocked":
            continue
        image = report["image"]
        for finding in report["findings"]:
            automatic = False
            owner = GO_IMAGES.get(image)
            # Debian's merged /usr canonicalizes the API server's /bin copy.
            owned_target = owner and finding["target"] == owner[0]
            if image == "kfp-api-server" and finding[
                    "target"] == "/usr/bin/apiserver":
                owned_target = True
            if (owned_target and finding["ecosystem"] == "Go" and
                    finding["package"] == "stdlib"):
                if current_go is None:
                    current_go = compiler_version(source_root)
                fixed = [
                    stable_go_version(value)
                    for value in finding["fixed_versions"]
                ]
                fixed = [
                    version for version in fixed if version and
                    version[:2] == current_go[:2] and version > current_go
                ]
                if stable_go_version(
                        finding["installed"]) == current_go and fixed:
                    go_fixes.append(min(fixed))
                    automatic = True
            elif (image == "kfp-frontend" and
                  finding["target"] in NPM_TARGETS and
                  finding["ecosystem"] == "npm"):
                if packages is None:
                    packages = npm_packages(source_root)
                if ((finding["package"], finding["installed"]) in packages and
                        has_same_major_npm_fix(finding["installed"],
                                               finding["fixed_versions"]) and
                        not any(
                            value.startswith("UNDETERMINED:")
                            for value in finding["fixed_versions"])):
                    npm_vulns.add(finding["cve"])
                    automatic = True
            if automatic:
                selected[finding_key(image, finding)] = dict(
                    image=image,
                    **{
                        field: finding[field]
                        for field in ("target", "cve", "ecosystem", "package")
                    })
    verify_images = []
    if go_fixes:
        for image, (_, dockerfile) in sorted(GO_IMAGES.items()):
            source_file(source_root, dockerfile)
            verify_images.append(
                dict(image=image, dockerfile=dockerfile, context="."))
    if npm_vulns:
        source_file(source_root, "frontend/Dockerfile")
        verify_images.append(
            dict(
                image="kfp-frontend",
                dockerfile="frontend/Dockerfile",
                context="."))
    return {
        "schema_version":
            1,
        "source_sha":
            source_sha,
        "source_branch":
            source_branch,
        "run_url":
            run_url,
        "go_version":
            ".".join(map(str, max(go_fixes))) if go_fixes else "",
        "npm_vulns":
            sorted(npm_vulns),
        "targets": [selected[key] for key in sorted(selected)],
        "verify_images":
            sorted(verify_images, key=lambda value: value["image"]),
        "reports":
            reports,
    }


def remediation_markdown(plan):
    lines = [
        "# Release CVE remediation", "",
        f"Source branch: {markdown(plan['source_branch'])}; commit: {markdown(plan['source_sha'])}.",
        f"Release run: {plan['run_url']}", "",
        "Findings are grouped across images and architectures. A proposed update still requires rebuilds and rescans; release publication remains blocked until fixed or explicitly overridden.",
        ""
    ]
    if plan["go_version"]:
        lines.append(
            f"- Proposed Go compiler patch: {markdown(plan['go_version'])}. Rebuild all six KFP Go images."
        )
    if plan["npm_vulns"]:
        lines.append("- Proposed frontend server lockfile remediation: " +
                     ", ".join(plan["npm_vulns"]) + ".")
    if not plan["targets"]:
        lines.append(
            "No supported automatic source update was identified. Use the owner-specific instructions below."
        )
    lines += [
        "",
        "| CVE | Ecosystem / package | Installed → fixed | Images / platforms | Target | Source owner / action |",
        "| --- | --- | --- | --- | --- | --- |"
    ]
    grouped = {}
    selected = {finding_key(item["image"], item) for item in plan["targets"]}
    for report in plan["reports"]:
        if report["outcome"] != "blocked":
            continue
        for finding in report["findings"]:
            key = tuple(finding[field]
                        for field in ("target", "cve", "ecosystem", "package",
                                      "installed"))
            group = grouped.setdefault(
                key, {
                    "finding": finding,
                    "images": set(),
                    "fixes": set(),
                    "actions": set()
                })
            group["images"].add(f"{report['image']} ({report['platform']})")
            group["fixes"].update(finding["fixed_versions"])
            if finding_key(report["image"], finding) in selected:
                action = "Root go.mod and Go builder pins: apply a compiler patch and rebuild." if finding[
                    "ecosystem"] == "Go" else "frontend/server/package-lock.json: apply OSV npm remediation and rebuild frontend."
            else:
                action = "Manual: " + remediation_hint(finding)
            group["actions"].add(action)
    for key in sorted(grouped):
        group = grouped[key]
        finding = group["finding"]
        values = (finding["cve"],
                  f"{finding['ecosystem']} / {finding['package']}",
                  finding["installed"] + " → " +
                  ", ".join(sorted(group["fixes"])),
                  ", ".join(sorted(group["images"])), finding["target"],
                  " ".join(sorted(group["actions"])))
        lines.append("| " + " | ".join(markdown(value) for value in values) +
                     " |")
    if not grouped:
        lines.append("No unoverridden blocking findings.")
    return "\n".join(lines) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for option in ("reports", "source", "source-sha", "source-branch",
                   "run-url", "output-dir"):
        parser.add_argument("--" + option, required=True)
    args = parser.parse_args()
    plan = plan_remediation(args.reports, args.source, args.source_sha,
                            args.source_branch, args.run_url)
    output = Path(args.output_dir)
    output.mkdir(parents=True, exist_ok=True)
    (output / "plan.json").write_text(
        json.dumps(plan, indent=2) + "\n", encoding="utf-8")
    (output / "remediation.md").write_text(
        remediation_markdown(plan), encoding="utf-8")


if __name__ == "__main__":
    main()
