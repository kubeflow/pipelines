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
"""Regression coverage for source ownership and complete release evidence."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

from arm64_smoke import IMAGES
from check_fixable_cves import find_blocking_cves
from check_fixable_cves import structured_result
import release_cve_plan as planner

SHA = "a" * 40
RUN_URL = "https://github.com/kubeflow/pipelines/actions/runs/12345"


class PlanTest(unittest.TestCase):

    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.source = self.root / "source"
        self.reports = self.root / "reports"
        self.source.mkdir()
        self.reports.mkdir()
        (self.source / "go.mod").write_text(
            "module example.com/test\n\ngo 1.26.0\ntoolchain go1.26.1\n")
        for _, path in planner.GO_IMAGES.values():
            target = self.source / path
            target.parent.mkdir(exist_ok=True)
            target.write_text("FROM golang:1.26.1\n")
        (self.source / "frontend/server").mkdir(parents=True)
        (self.source / "frontend/Dockerfile").write_text("FROM node:22\n")
        (self.source / "frontend/server/package-lock.json").write_text(
            json.dumps({
                "packages": {
                    "node_modules/example": {
                        "version": "1.0.0"
                    },
                    "node_modules/a/node_modules/@scope/nested": {
                        "version": "2.0.0"
                    }
                }
            }))
        for image in sorted(IMAGES):
            for platform in sorted(planner.PLATFORMS):
                self.write_report(image, platform, [])

    def write_report(self, image, platform, findings, allow=False):
        report = structured_result(
            findings, image, platform,
            f"ghcr.io/kubeflow/{image}@sha256:" + "b" * 64, SHA, allow)
        path = self.reports / f"cve-result-{image}-{platform.rsplit('/', 1)[1]}.json"
        path.write_text(json.dumps(report))
        return path

    def go_finding(self,
                   fixed="1.26.2, 1.27.0",
                   target="/bin/apiserver",
                   cve="CVE-2026-12345",
                   installed="1.26.1"):
        return (target, cve, "Go", "stdlib", installed, fixed)

    def plan(self):
        return planner.plan_remediation(self.reports, self.source, SHA,
                                        "release-3.0", RUN_URL)

    def test_complete_clean_inventory_has_no_automatic_changes(self):
        plan = self.plan()
        self.assertEqual(plan["go_version"], "")
        self.assertEqual(plan["npm_vulns"], [])
        self.assertEqual(plan["targets"], [])
        self.assertEqual(plan["verify_images"], [])
        self.assertIn("No supported automatic source update",
                      planner.remediation_markdown(plan))

    def test_go_uses_toolchain_and_smallest_sufficient_same_series_patch(self):
        self.write_report("kfp-api-server", "linux/amd64",
                          [self.go_finding(fixed="1.26.3, 1.26.9, 1.27.1")])
        self.write_report("kfp-driver", "linux/arm64", [
            self.go_finding(
                fixed="1.25.99, 1.26.4, 1.27.1",
                target="/bin/driver",
                cve="CVE-2026-12346")
        ])
        plan = self.plan()
        self.assertEqual(plan["go_version"], "1.26.4")
        self.assertEqual(len(plan["targets"]), 2)
        self.assertEqual({image["image"] for image in plan["verify_images"]},
                         set(planner.GO_IMAGES))
        self.assertIn("Root go.mod", planner.remediation_markdown(plan))

    def test_debian_merged_usr_api_binary_alias_maps_to_root_go(self):
        # OSV 2.5.0's real API-server report canonicalizes Dockerfile /bin/apiserver.
        self.write_report("kfp-api-server", "linux/amd64",
                          [self.go_finding(target="/usr/bin/apiserver")])
        plan = self.plan()
        self.assertEqual(plan["go_version"], "1.26.2")
        self.assertEqual(plan["targets"][0]["target"], "/usr/bin/apiserver")

    def test_go_never_automatically_moves_release_series_or_prerelease(self):
        for fixed in ("1.27.2", "1.26.2rc1", "1.26.0",
                      "UNDETERMINED: no matching affected package"):
            with self.subTest(fixed=fixed):
                self.write_report("kfp-api-server", "linux/amd64",
                                  [self.go_finding(fixed=fixed)])
                self.assertEqual(self.plan()["go_version"], "")

    def test_external_go_binaries_and_compiler_mismatches_remain_manual(self):
        for finding in (self.go_finding(target="/usr/bin/argoexec"),
                        self.go_finding(installed="1.25.10"),
                        ("/bin/apiserver", "CVE-2026-12345", "Go",
                         "example.com/module", "1.26.1", "1.26.2")):
            with self.subTest(finding=finding):
                self.write_report("kfp-api-server", "linux/amd64", [finding])
                plan = self.plan()
                self.assertEqual(plan["targets"], [])
                self.assertIn("Manual:", planner.remediation_markdown(plan))

    def test_same_cve_other_installed_version_remains_manual(self):
        self.write_report(
            "kfp-api-server", "linux/amd64",
            [self.go_finding(),
             self.go_finding(installed="1.25.10")])
        plan = self.plan()
        self.assertEqual(len(plan["targets"]), 1)
        self.assertEqual(plan["targets"][0]["installed"], "1.26.1")
        self.assertIn("Manual:", planner.remediation_markdown(plan))

    def test_overridden_findings_do_not_trigger_changes(self):
        self.write_report(
            "kfp-api-server", "linux/amd64", [self.go_finding()], allow=True)
        self.assertEqual(self.plan()["targets"], [])

    def test_npm_only_maps_matching_server_lockfile_packages(self):
        for path, package, installed in (
            ("/server/package-lock.json", "example",
             "1.0.0"), ("/server/node_modules/.package-lock.json",
                        "@scope/nested", "2.0.0")):
            with self.subTest(path=path):
                self.write_report(
                    "kfp-frontend", "linux/amd64",
                    [(path, "CVE-2026-12345", "npm", package, installed,
                      installed.split(".")[0] + ".1.0")])
                plan = self.plan()
                self.assertEqual(plan["npm_vulns"], ["CVE-2026-12345"])
                self.assertEqual(
                    [image["image"] for image in plan["verify_images"]],
                    ["kfp-frontend"])
        for image, path, package, installed in (
            ("kfp-api-server", "/server/package-lock.json", "example", "1.0.0"),
            ("kfp-frontend",
             "/usr/local/lib/node_modules/npm/package-lock.json", "example",
             "1.0.0"), ("kfp-frontend", "/server/package-lock.json", "unknown",
                        "1.0.0"), ("kfp-frontend", "/server/package-lock.json",
                                   "example", "0.9.0")):
            with self.subTest(image=image, path=path, package=package):
                self.write_report("kfp-frontend", "linux/amd64", [])
                self.write_report(
                    image, "linux/amd64",
                    [(path, "CVE-2026-12345", "npm", package, installed,
                      installed.split(".")[0] + ".1.0")])
                self.assertEqual(self.plan()["npm_vulns"], [])
                self.write_report(image, "linux/amd64", [])

    def test_npm_major_only_fix_does_not_block_supported_mixed_plan(self):
        lockfile = self.source / "frontend/server/package-lock.json"
        lock = json.loads(lockfile.read_text())
        lock["packages"]["node_modules/uuid"] = {"version": "9.0.1"}
        lockfile.write_text(json.dumps(lock))
        self.write_report(
            "kfp-frontend",
            "linux/amd64",
            [
                ("/server/node_modules/.package-lock.json", "CVE-2026-12345",
                 "npm", "example", "1.0.0", "1.1.0"),
                # Real report: uuid 9.0.1 fixes require a major upgrade.
                ("/server/node_modules/.package-lock.json", "CVE-2026-41907",
                 "npm", "uuid", "9.0.1", "11.1.1, 12.0.1, 13.0.1"),
            ])
        self.write_report(
            "kfp-driver", "linux/arm64",
            [self.go_finding(target="/bin/driver", cve="CVE-2026-12346")])
        plan = self.plan()
        self.assertEqual(plan["npm_vulns"], ["CVE-2026-12345"])
        self.assertEqual(plan["go_version"], "1.26.2")
        self.assertEqual({target["cve"] for target in plan["targets"]},
                         {"CVE-2026-12345", "CVE-2026-12346"})
        summary = planner.remediation_markdown(plan)
        uuid_row = next(
            line for line in summary.splitlines() if "CVE-2026-41907" in line)
        self.assertIn("Manual:", uuid_row)
        self.assertEqual({image["image"] for image in plan["verify_images"]},
                         IMAGES)

    def test_npm_requires_a_newer_stable_fix_without_major_upgrade(self):
        cases = [
            ("1.0.0", ["1.1.0"], True),
            ("1.0.0", ["0.9.9", "1.1.0", "2.0.0"], True),
            ("1.0.0", ["2.0.0"], False),
            ("1.0.0", ["1.0.0", "0.9.0"], False),
            ("1.0.0", ["1.1.0-rc.1"], False),
            ("1.0.0", ["UNDETERMINED: no matching affected package"], False),
            ("1.0.0-rc.1", ["1.0.0"], False),
            ("go1.0.0", ["1.1.0"], False),
        ]
        for installed, fixed, expected in cases:
            with self.subTest(installed=installed, fixed=fixed):
                self.assertEqual(
                    planner.has_same_major_npm_fix(installed, fixed), expected)

    def test_real_alpine_distro_advisory_produces_manual_base_image_action(
            self):
        report = json.loads((Path(__file__).parent /
                             "testdata/osv_alpine_3_18.json").read_text())
        self.write_report("kfp-api-server", "linux/arm64",
                          find_blocking_cves(report))
        plan = self.plan()
        self.assertEqual(plan["targets"], [])
        self.assertIn("base image or distribution package",
                      planner.remediation_markdown(plan))
        self.assertIn("CVE-2023-2650", planner.remediation_markdown(plan))

    def test_duplicates_are_grouped_across_architectures_and_images(self):
        for image in ("kfp-api-server", "kfp-driver"):
            for platform in planner.PLATFORMS:
                self.write_report(
                    image, platform,
                    [self.go_finding(target="/usr/local/bin/inherited")])
        summary = planner.remediation_markdown(self.plan())
        self.assertEqual(summary.count("CVE-2026-12345"), 1)
        self.assertIn("kfp-api-server (linux/amd64)", summary)
        self.assertIn("kfp-driver (linux/arm64)", summary)

    def test_retry_report_duplicates_must_be_identical(self):
        original = self.reports / "cve-result-kfp-api-server-amd64.json"
        duplicate = self.reports / "cve-result-retry.json"
        duplicate.write_bytes(original.read_bytes())
        self.assertEqual(len(self.plan()["reports"]), 14)
        changed = json.loads(duplicate.read_text())
        changed["image_ref"] = changed["image_ref"].replace("b" * 64, "c" * 64)
        duplicate.write_text(json.dumps(changed))
        with self.assertRaisesRegex(ValueError, "Conflicting duplicate"):
            self.plan()

    def test_missing_architecture_prevents_any_automatic_plan(self):
        self.write_report("kfp-api-server", "linux/amd64", [self.go_finding()])
        (self.reports / "cve-result-kfp-driver-arm64.json").unlink()
        with self.assertRaisesRegex(ValueError, "Missing CVE reports"):
            self.plan()

    def test_invalid_or_source_mismatched_reports_fail_closed(self):
        path = self.reports / "cve-result-kfp-api-server-amd64.json"
        original = json.loads(path.read_text())
        for key, value in (("source_sha", "c" * 40), ("schema_version", 2),
                           ("platform", "linux/s390x"),
                           ("image_ref",
                            "ghcr.io/kubeflow/kfp-api-server:latest"),
                           ("outcome", "blocked")):
            with self.subTest(key=key):
                changed = copy.deepcopy(original)
                changed[key] = value
                path.write_text(json.dumps(changed))
                with self.assertRaises(ValueError):
                    self.plan()
        path.write_text(json.dumps(original))

    def test_source_symlink_cannot_redirect_ownership_reads(self):
        original = self.source / "go.mod"
        external = self.root / "external-go.mod"
        original.rename(external)
        original.symlink_to(external)
        self.write_report("kfp-api-server", "linux/amd64", [self.go_finding()])
        with self.assertRaisesRegex(ValueError, "symlink"):
            self.plan()

    def test_report_symlink_cannot_supply_evidence(self):
        original = self.reports / "cve-result-kfp-api-server-amd64.json"
        external = self.root / "external.json"
        original.rename(external)
        original.symlink_to(external)
        with self.assertRaisesRegex(ValueError, "symlink"):
            self.plan()

    def test_metadata_cannot_inject_links_or_branch_options(self):
        for branch, url in (("--help", RUN_URL), ("release/../../main",
                                                  RUN_URL),
                            ("release-3.0", "https://example.com/untrusted")):
            with self.subTest(branch=branch, url=url):
                with self.assertRaises(ValueError):
                    planner.plan_remediation(self.reports, self.source, SHA,
                                             branch, url)


if __name__ == "__main__":
    unittest.main()
