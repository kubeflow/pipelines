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
"""Unit tests for ci_health_report.py (stdlib only).

Run with:  python3 -m unittest .github/resources/scripts/ci_health_report_test.py
       or: python3 .github/resources/scripts/ci_health_report_test.py
"""

import io
import unittest
import zipfile
from unittest import mock

import ci_health_report as chr_mod


def make_run(run_id, conclusion="failure", attempts=1, created="2026-07-13T00:00:00Z"):
    return {
        "id": run_id,
        "status": "completed",
        "conclusion": conclusion,
        "run_attempt": attempts,
        "created_at": created,
        "name": "WF",
    }


def make_job(name, conclusion, minutes=10):
    return {
        "name": name,
        "conclusion": conclusion,
        "started_at": "2026-07-13T00:00:00Z",
        "completed_at": f"2026-07-13T00:{minutes:02d}:00Z",
    }


def junit_zip(*xml_bodies):
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w") as archive:
        for index, body in enumerate(xml_bodies):
            archive.writestr(f"report{index}.xml", body)
    return buffer.getvalue()


class ConclusionHandlingTest(unittest.TestCase):
    """Cancelled/skipped are non-results; other non-success are failures."""

    def collect(self, jobs):
        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                return [make_run(1)] if "e2e-test.yml" in url else []
            return jobs

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            return chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 40)

    def test_cancelled_and_skipped_excluded_from_denominator(self):
        lanes, _, _, _ = self.collect(
            [
                make_job("lane", "success"),
                make_job("lane", "cancelled"),
                make_job("lane", "skipped"),
            ]
        )
        self.assertEqual(lanes[("WF", "lane")]["total"], 1)
        self.assertEqual(lanes[("WF", "lane")]["failed"], 0)

    def test_timed_out_and_stale_count_as_failures(self):
        lanes, failed_runs, _, _ = self.collect(
            [make_job("lane", "timed_out"), make_job("lane", "stale")]
        )
        self.assertEqual(lanes[("WF", "lane")]["total"], 2)
        self.assertEqual(lanes[("WF", "lane")]["failed"], 2)
        self.assertEqual([run_id for _, run_id in failed_runs], [1])


class RerunHandlingTest(unittest.TestCase):
    """Every attempt is counted, so re-run-then-green still shows the flake."""

    def test_rerun_attempts_all_counted_and_run_marked_failed(self):
        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                # Overall conclusion success: the re-run went green.
                return (
                    [make_run(7, conclusion="success", attempts=2)]
                    if "e2e-test.yml" in url
                    else []
                )
            if "/attempts/1/" in url:
                return [make_job("lane", "failure")]
            if "/attempts/2/" in url:
                return [make_job("lane", "success")]
            raise AssertionError(f"unexpected jobs url {url}")

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            lanes, failed_runs, reruns, _ = chr_mod.collect_lane_stats(
                "t", "o/r", "2026-07-01", 40
            )
        self.assertEqual(lanes[("WF", "lane")]["total"], 2)
        self.assertEqual(lanes[("WF", "lane")]["failed"], 1)
        self.assertEqual(reruns, 1)
        # The run is a junit candidate even though its final conclusion is green.
        self.assertEqual([run_id for _, run_id in failed_runs], [7])


class TruncationFlagTest(unittest.TestCase):
    def test_run_cap_is_surfaced_in_notes(self):
        runs = [make_run(i) for i in range(3)]

        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                return runs if "e2e-test.yml" in url else []
            return [make_job("lane", "success")]

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            _, _, _, notes = chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 2)
        self.assertTrue(any("run cap applied" in note for note in notes))

    def test_rate_limit_is_surfaced_in_notes(self):
        def fake_paginate(token, url, key, max_pages=3):
            raise chr_mod.RateLimited(url)

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            lanes, _, _, notes = chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 2)
        self.assertEqual(len(lanes), 0)
        self.assertTrue(any("rate-limited" in note for note in notes))


class JunitIngestionTest(unittest.TestCase):
    GOOD_XML = (
        '<testsuite><testcase classname="SuiteA" name="flaky test">'
        "<failure>boom</failure></testcase>"
        '<testcase classname="SuiteB" name="flaky test"><error>err</error></testcase>'
        '<testcase classname="SuiteA" name="green test"/></testsuite>'
    )

    def test_classname_disambiguates_and_malformed_counts_as_error(self):
        payload = junit_zip(self.GOOD_XML, "<not-closed")
        artifacts = [
            {"name": "junit-xml - lane", "archive_download_url": "u", "expired": False}
        ]

        def fake_paginate(token, url, key, max_pages=3):
            return artifacts

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate), \
                mock.patch.object(chr_mod, "api_request", return_value=payload):
            tests, parsed, errors = chr_mod.collect_failed_tests(
                "t", "o/r", [("2026-07-13T00:00:00Z", 1)], 5
            )
        self.assertEqual(tests["SuiteA :: flaky test"], 1)
        self.assertEqual(tests["SuiteB :: flaky test"], 1)
        self.assertNotIn("SuiteA :: green test", tests)
        self.assertEqual(parsed, 1)
        self.assertEqual(errors, 1)  # the malformed member

    def test_newest_failed_runs_win_the_budget_across_workflows(self):
        seen = []

        def fake_paginate(token, url, key, max_pages=3):
            seen.append(url)
            return []

        failed_runs = [
            ("2026-07-01T00:00:00Z", 111),  # oldest (from the first workflow)
            ("2026-07-13T00:00:00Z", 999),  # newest (from a later workflow)
        ]
        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            chr_mod.collect_failed_tests("t", "o/r", failed_runs, 1)
        self.assertEqual(len(seen), 1)
        self.assertIn("/runs/999/", seen[0])

    def test_artifact_api_error_is_reported_not_silent(self):
        def fake_paginate(token, url, key, max_pages=3):
            raise chr_mod.RateLimited(url)

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            tests, parsed, errors = chr_mod.collect_failed_tests(
                "t", "o/r", [("2026-07-13T00:00:00Z", 1)], 5
            )
        self.assertEqual(len(tests), 0)
        self.assertEqual(parsed, 0)
        self.assertEqual(errors, 1)


class RenderTest(unittest.TestCase):
    def test_notes_and_ingestion_errors_are_visible(self):
        lanes = {("WF", "lane"): {"total": 4, "failed": 2, "durations": [10.0, 20.0]}}
        report = chr_mod.render_report(
            lanes, {}, 0, 3, reruns=1, days=7, notes=["run cap applied: `e2e` (50→40)"]
        )
        self.assertIn("Data completeness", report)
        self.assertIn("run cap applied", report)
        self.assertIn("3 artifact/XML ingestion error(s)", report)
        self.assertIn("| WF | lane | 4 | 2 | 50% | 15 |", report)

    def test_zero_artifacts_message_distinct_from_errors(self):
        clean = chr_mod.render_report({}, {}, 0, 0, 0, 7, [])
        self.assertIn("No `junit-xml - *` artifacts found", clean)
        errored = chr_mod.render_report({}, {}, 0, 2, 0, 7, [])
        self.assertNotIn("No `junit-xml - *` artifacts found", errored)
        self.assertIn("No junit artifacts could be ingested", errored)


class UpsertIssueTest(unittest.TestCase):
    def test_updates_existing_issue(self):
        calls = []

        def fake_api(token, url, method="GET", body=None, raw=False):
            calls.append((method, url))
            if method == "POST" and url.endswith("/labels"):
                return {}
            if "issues?labels=" in url:
                return [{"number": 42, "html_url": "issue-url"}]
            if method == "PATCH":
                return {}
            raise AssertionError(f"unexpected call {method} {url}")

        with mock.patch.object(chr_mod, "api_request", side_effect=fake_api):
            url = chr_mod.upsert_issue("t", "o/r", "report")
        self.assertEqual(url, "issue-url")
        self.assertIn(("PATCH", f"{chr_mod.API_ROOT}/repos/o/r/issues/42"), calls)

    def test_creates_issue_when_none_open(self):
        def fake_api(token, url, method="GET", body=None, raw=False):
            if method == "POST" and url.endswith("/labels"):
                return {}
            if "issues?labels=" in url:
                return []
            if method == "POST" and url.endswith("/issues"):
                return {"html_url": "new-issue-url"}
            raise AssertionError(f"unexpected call {method} {url}")

        with mock.patch.object(chr_mod, "api_request", side_effect=fake_api):
            url = chr_mod.upsert_issue("t", "o/r", "report")
        self.assertEqual(url, "new-issue-url")


if __name__ == "__main__":
    unittest.main()
