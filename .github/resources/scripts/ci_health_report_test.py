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

Run with:  cd .github/resources/scripts && python3 -m unittest -v ci_health_report_test
(the leading-dot directory prevents unittest path-to-module conversion from the
repo root, and default `unittest discover` only matches test*.py). CI runs this
via .github/workflows/ci-scripts-tests.yml on changes to these scripts.
"""

import io
import json
import os
import re
import tempfile
import unittest
import zipfile
from pathlib import Path
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
                return ([make_run(1)], False) if "e2e-test.yml" in url else ([], False)
            return jobs, False

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
                    ([make_run(7, conclusion="success", attempts=2)], False)
                    if "e2e-test.yml" in url
                    else ([], False)
                )
            if "/attempts/1/" in url:
                return [make_job("lane", "failure")], False
            if "/attempts/2/" in url:
                return [make_job("lane", "success")], False
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
                return (runs, False) if "e2e-test.yml" in url else ([], False)
            return [make_job("lane", "success")], False

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


class JobFetchErrorTest(unittest.TestCase):
    def test_job_listing_http_error_is_surfaced_not_healthy(self):
        import urllib.error

        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                return ([make_run(1)], False) if "e2e-test.yml" in url else ([], False)
            raise urllib.error.HTTPError(url, 500, "boom", {}, io.BytesIO(b""))

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            lanes, _, _, notes = chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 40)
        self.assertEqual(len(lanes), 0)
        self.assertTrue(
            any("job listing(s) failed" in note for note in notes),
            f"expected job-fetch note in {notes}",
        )


class GitHubStatusCorrelationTest(unittest.TestCase):
    def incident_payload(self):
        return {
            "incidents": [
                {
                    "id": "incident-1",
                    "name": "Disruption with some GitHub services",
                    "impact": "major",
                    "shortlink": "https://stspg.io/example",
                    "created_at": "2026-07-24T16:17:00Z",
                    "started_at": "2026-07-24T16:17:00Z",
                    "resolved_at": "2026-07-24T17:36:00Z",
                    "components": [{"name": "Actions"}],
                    "incident_updates": [{
                        "display_at": "2026-07-24T16:19:00Z",
                        "affected_components": [{"name": "API Requests"}],
                    }],
                },
                {
                    "id": "copilot-only",
                    "name": "Copilot unavailable",
                    "created_at": "2026-07-24T12:00:00Z",
                    "resolved_at": "2026-07-24T13:00:00Z",
                    "components": [{"name": "Copilot"}],
                },
            ]
        }

    def test_status_request_never_forwards_github_credentials(self):
        response = mock.MagicMock()
        response.read.return_value = b'{"incidents":[]}'
        response.__enter__.return_value = response
        with mock.patch.object(
            chr_mod.urllib.request, "urlopen", return_value=response
        ) as urlopen:
            chr_mod.status_request()

        request = urlopen.call_args.args[0]
        self.assertEqual(request.full_url, chr_mod.GITHUB_STATUS_INCIDENTS_URL)
        self.assertIsNone(request.get_header("Authorization"))

    def test_filters_components_and_preserves_official_incident_evidence(self):
        incidents = chr_mod.github_status_incidents(
            self.incident_payload(),
            "2026-07-24",
            now=chr_mod.parse_timestamp("2026-07-24T18:00:00Z"),
        )

        self.assertEqual(len(incidents), 1)
        self.assertEqual(incidents[0]["id"], "incident-1")
        self.assertEqual(incidents[0]["components"], ["API Requests", "Actions"])
        self.assertEqual(incidents[0]["url"], "https://stspg.io/example")

    def test_strict_and_late_reporting_matches_are_distinct(self):
        incidents = chr_mod.github_status_incidents(
            self.incident_payload(), "2026-07-24"
        )
        observations = [
            {
                "id": "nearby",
                "failed": True,
                "started": "2026-07-24T16:00:00Z",
                "completed": "2026-07-24T16:10:00Z",
            },
            {
                "id": "overlap",
                "failed": True,
                "started": "2026-07-24T16:30:00Z",
                "completed": "2026-07-24T16:40:00Z",
            },
            {
                "id": "success",
                "failed": False,
                "started": "2026-07-24T16:30:00Z",
                "completed": "2026-07-24T16:40:00Z",
            },
        ]

        correlated = chr_mod.correlate_github_incidents(
            observations,
            incidents,
            now=chr_mod.parse_timestamp("2026-07-24T18:00:00Z"),
        )

        self.assertEqual(
            correlated[0]["github_incidents"],
            [{"id": "incident-1", "match": "nearby"}],
        )
        self.assertEqual(
            correlated[1]["github_incidents"],
            [{"id": "incident-1", "match": "overlap"}],
        )
        self.assertEqual(correlated[2]["github_incidents"], [])

    def test_github_service_error_signature_requires_target_and_error(self):
        self.assertTrue(
            chr_mod.github_failure_signature(
                "Error: failed to download action from https://github.com: "
                "504 Gateway Timeout"
            )
        )
        self.assertFalse(
            chr_mod.github_failure_signature(
                "dial tcp 10.96.0.5:9000: i/o timeout"
            )
        )

    def test_correlated_job_log_adds_signature_evidence(self):
        observations = [{
            "id": "one",
            "job_id": 42,
            "github_incidents": [{"id": "incident-1", "match": "overlap"}],
        }]
        log = (
            b"actions/download-artifact failed: 403 Forbidden from "
            b"results-receiver.actions.githubusercontent.com"
        )
        with mock.patch.object(chr_mod, "api_request", return_value=log) as request:
            enriched, errors = chr_mod.add_github_log_evidence(
                "token", "o/r", observations
            )

        self.assertEqual(errors, 0)
        self.assertTrue(enriched[0]["github_incidents"][0]["signature"])
        self.assertIn("/actions/jobs/42/logs", request.call_args.args[1])


class StaleWorkflowTest(unittest.TestCase):
    def test_missing_workflow_404_is_surfaced(self):
        import urllib.error

        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                raise urllib.error.HTTPError(url, 404, "gone", {}, io.BytesIO(b""))
            raise AssertionError("no jobs should be fetched")

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            _, _, _, notes = chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 40)
        self.assertTrue(
            any("not found" in note and "TARGET_WORKFLOWS" in note for note in notes),
            f"expected missing-workflow note in {notes}",
        )


class PageBudgetTest(unittest.TestCase):
    def test_exact_full_page_uses_total_count_to_avoid_false_truncation(self):
        payload = {"total_count": 100, "artifacts": [{"id": i} for i in range(100)]}
        with mock.patch.object(chr_mod, "api_request", return_value=payload):
            items, may_have_more = chr_mod.paginate("t", "u", "artifacts", max_pages=1)
        self.assertEqual(len(items), 100)
        self.assertFalse(may_have_more)

    def test_full_page_below_total_count_is_truncated(self):
        payload = {"total_count": 101, "artifacts": [{"id": i} for i in range(100)]}
        with mock.patch.object(chr_mod, "api_request", return_value=payload):
            items, may_have_more = chr_mod.paginate("t", "u", "artifacts", max_pages=1)
        self.assertEqual(len(items), 100)
        self.assertTrue(may_have_more)

    def test_job_page_budget_truncation_is_surfaced(self):
        def fake_paginate(token, url, key, max_pages=3):
            if "workflows/" in url:
                return ([make_run(1)], False) if "e2e-test.yml" in url else ([], False)
            return [make_job("lane", "success")], True  # budget exhausted

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            _, _, _, notes = chr_mod.collect_lane_stats("t", "o/r", "2026-07-01", 40)
        self.assertTrue(
            any("page budget" in note for note in notes),
            f"expected page-budget note in {notes}",
        )

    def test_artifact_page_budget_counts_as_ingestion_gap(self):
        def fake_paginate(token, url, key, max_pages=3):
            return [], True  # more artifacts exist beyond page 1

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate):
            _, _, errors, _ = chr_mod.collect_failed_tests(
                "t", "o/r", [("2026-07-13T00:00:00Z", 1)], 5
            )
        self.assertEqual(errors, 1)


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
            return artifacts, False

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate), \
                mock.patch.object(chr_mod, "api_request", return_value=payload):
            tests, parsed, errors, scanned = chr_mod.collect_failed_tests(
                "t", "o/r", [("2026-07-13T00:00:00Z", 1)], 5
            )
        self.assertEqual(scanned, 1)
        self.assertEqual(tests["SuiteA :: flaky test"], 1)
        self.assertEqual(tests["SuiteB :: flaky test"], 1)
        self.assertNotIn("SuiteA :: green test", tests)
        self.assertEqual(parsed, 1)
        self.assertEqual(errors, 1)  # the malformed member

    def test_retry_artifact_replaces_unsuffixed_duplicate(self):
        payload = junit_zip(self.GOOD_XML)
        artifacts = [
            {
                "name": "junit-xml - lane",
                "archive_download_url": "primary",
                "expired": False,
            },
            {
                "name": "junit-xml - lane - retry-1",
                "archive_download_url": "retry",
                "expired": False,
            },
        ]

        def fake_paginate(token, url, key, max_pages=3):
            return artifacts, False

        with mock.patch.object(chr_mod, "paginate", side_effect=fake_paginate), \
                mock.patch.object(chr_mod, "api_request", return_value=payload) as request:
            tests, parsed, errors, scanned = chr_mod.collect_failed_tests(
                "t", "o/r", [("2026-07-13T00:00:00Z", 1)], 5
            )

        self.assertEqual(scanned, 1)
        self.assertEqual(tests["SuiteA :: flaky test"], 1)
        self.assertEqual(tests["SuiteB :: flaky test"], 1)
        self.assertEqual(parsed, 1)
        self.assertEqual(errors, 0)
        request.assert_called_once_with("t", "retry", raw=True)

    def test_highest_retry_attempt_wins_for_each_artifact(self):
        artifacts = [
            {
                "name": "junit-xml - lane",
                "archive_download_url": "primary",
                "expired": False,
            },
            {
                "name": "junit-xml - lane - retry-2",
                "archive_download_url": "retry-2",
                "expired": False,
            },
            {
                "name": "junit-xml - lane - retry-10",
                "archive_download_url": "retry-10",
                "expired": False,
            },
            {
                "name": "junit-xml - lane - retry-11",
                "archive_download_url": "expired-retry",
                "expired": True,
            },
            {
                "name": "junit-xml - primary-only",
                "archive_download_url": "primary-only",
                "expired": False,
            },
            {
                "name": "kind-logs - lane - retry-12",
                "archive_download_url": "unrelated",
                "expired": False,
            },
        ]

        selected = chr_mod.select_junit_artifacts(artifacts)

        self.assertEqual(
            [artifact["archive_download_url"] for artifact in selected],
            ["retry-10", "primary-only"],
        )

    def test_newest_failed_runs_win_the_budget_across_workflows(self):
        seen = []

        def fake_paginate(token, url, key, max_pages=3):
            seen.append(url)
            return [], False

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
            tests, parsed, errors, scanned = chr_mod.collect_failed_tests(
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
        self.assertIn("3 artifact ingestion gap(s)", report)
        self.assertIn("| WF | lane | 4 | 2 | 50% | 15 |", report)

    def test_zero_artifacts_message_distinct_from_errors(self):
        clean = chr_mod.render_report({}, {}, 0, 0, 0, 7, [])
        self.assertIn("No `junit-xml - *` artifacts found", clean)
        errored = chr_mod.render_report({}, {}, 0, 2, 0, 7, [])
        self.assertNotIn("No `junit-xml - *` artifacts found", errored)
        self.assertIn("No junit artifacts could be ingested", errored)

    def test_junit_scan_scope_is_named_not_whole_window(self):
        capped = chr_mod.render_report(
            {}, {}, 0, 0, 0, 7, [], junit_scanned=15, junit_total=20
        )
        self.assertIn("newest 15 of 20 failed runs", capped)
        # the zero-artifact message must not claim the whole window
        self.assertIn("on the scanned failed runs", capped)
        uncapped = chr_mod.render_report(
            {}, {}, 0, 0, 0, 7, [], junit_scanned=3, junit_total=3
        )
        self.assertIn("Scanned 3 failed run(s)", uncapped)


class MissingImageArtifactPatternTest(unittest.TestCase):
    """The barrier's failure text is a contract; parsing it silently degrades."""

    def test_parses_every_artifact_list_the_barrier_emits(self):
        barrier = (
            Path(__file__).resolve().parent / 'wait-for-image-artifacts.sh'
        ).read_text(encoding='utf-8')
        messages = re.findall(
            r'fail_setup "(Missing branch image artifacts[^"]*)"', barrier)

        self.assertEqual(len(messages), 2, barrier)
        for message in messages:
            rendered = message.replace('${missing_artifacts[*]}', 'apiserver')
            with self.subTest(message=rendered):
                match = chr_mod.MISSING_IMAGE_ARTIFACTS.search(rendered)
                self.assertIsNotNone(match, rendered)
                self.assertEqual(match.group('artifacts'), 'apiserver')


class TrendAggregationTest(unittest.TestCase):
    def test_wilson_interval_is_bounded_and_tightens_with_sample_size(self):
        small = chr_mod.wilson_interval(1, 2)
        large = chr_mod.wilson_interval(50, 100)
        self.assertTrue(0 <= small[0] < small[1] <= 100)
        self.assertLess(large[1] - large[0], small[1] - small[0])

    def test_job_phases_derive_queue_setup_test_and_report(self):
        run = {"created_at": "2026-07-13T00:00:00Z"}
        job = {
            "started_at": "2026-07-13T00:02:00Z",
            "completed_at": "2026-07-13T00:30:00Z",
            "steps": [{
                "name": "Run Tests / Run Tests",
                "started_at": "2026-07-13T00:10:00Z",
                "completed_at": "2026-07-13T00:25:00Z",
            }],
        }
        self.assertEqual(
            chr_mod.job_phase_minutes(run, job),
            {
                "queue": 2.0, "setup": 8.0, "bootstrap": None,
                "build": None, "deploy": None, "test": 15.0, "report": 5.0,
            },
        )

    def test_daily_snapshot_has_true_test_rate_classes_and_rerun_rescue(self):
        observations = [
            {
                "id": "7:1:1", "date": "2026-07-13", "workflow": "WF",
                "lane": "lane", "run_id": 7, "attempt": 1, "sha": "a",
                "conclusion": "failure", "failed": True, "duration": 20.0,
                "phases": {"queue": 1.0, "setup": 3.0, "test": 12.0, "report": 4.0},
            },
            {
                "id": "7:2:2", "date": "2026-07-13", "workflow": "WF",
                "lane": "lane", "run_id": 7, "attempt": 2, "sha": "a",
                "conclusion": "success", "failed": False, "duration": 10.0,
                "phases": {"queue": 1.0, "setup": 2.0, "test": 6.0, "report": 1.0},
            },
        ]
        results = [{
            "generated_at": "2026-07-13T00:30:00Z",
            "workflow": "WF", "report_name": "lane", "result": "test_failure",
            "dimensions": {"cache_enabled": "true"},
            "signatures": {"client_timeout": 2},
            "tests": [
                {"id": "Suite :: flaky", "executions": 1, "failures": 1, "skipped": 0},
                {"id": "Suite :: green", "executions": 1, "failures": 0, "skipped": 0},
            ],
        }]
        snapshot = chr_mod.aggregate_daily(observations, results, {7: 2})[0]

        self.assertEqual(snapshot["totals"]["reruns"], 1)
        self.assertEqual(snapshot["totals"]["rerun_rescues"], 1)
        self.assertEqual(snapshot["failure_classes"]["test_failure"], 1)
        self.assertNotIn("unclassified_failure", snapshot["failure_classes"])
        self.assertEqual(snapshot["signatures"]["client_timeout"], 2)
        self.assertEqual(snapshot["tests"][0]["executions"], 1)
        self.assertEqual(snapshot["lanes"][0]["duration"], {"p50": 15.0, "p95": 19.5})

    def test_daily_snapshot_persists_github_incident_correlation(self):
        incident = {
            "id": "incident-1",
            "name": "Actions disruption",
            "url": "https://stspg.io/example",
            "impact": "major",
            "started_at": "2026-07-13T00:05:00Z",
            "resolved_at": "2026-07-13T00:30:00Z",
            "components": ["Actions"],
        }
        observations = [
            {
                "id": "7:1:1", "date": "2026-07-13", "workflow": "WF",
                "lane": "lane", "run_id": 7, "attempt": 1, "sha": "a",
                "conclusion": "failure", "failed": True, "duration": 20.0,
                "phases": {},
                "github_incidents": [{
                    "id": "incident-1",
                    "match": "overlap",
                    "signature": "GitHub returned 504",
                }],
            },
            {
                "id": "8:1:1", "date": "2026-07-13", "workflow": "WF",
                "lane": "lane", "run_id": 8, "attempt": 1, "sha": "b",
                "conclusion": "failure", "failed": True, "duration": 10.0,
                "phases": {},
                "github_incidents": [{"id": "incident-1", "match": "nearby"}],
            },
        ]

        snapshot = chr_mod.aggregate_daily(
            observations, [], {}, [incident]
        )[0]

        self.assertEqual(snapshot["totals"]["failures"], 2)
        self.assertEqual(snapshot["totals"]["github_correlated_failures"], 2)
        self.assertEqual(snapshot["totals"]["github_signature_matches"], 1)
        self.assertEqual(snapshot["totals"]["github_strict_overlaps"], 1)
        self.assertEqual(snapshot["totals"]["github_nearby_matches"], 1)
        self.assertEqual(snapshot["lanes"][0]["github_correlated_failures"], 2)
        self.assertEqual(snapshot["github_incidents"][0]["url"], incident["url"])
        self.assertEqual(snapshot["github_incidents"][0]["signature_matches"], 1)
        self.assertEqual(snapshot["github_incidents"][0]["strict_overlaps"], 1)
        self.assertEqual(snapshot["github_incidents"][0]["nearby_matches"], 1)

    def test_groups_registry_producer_with_missing_artifact_lanes(self):
        observations = [
            {
                "id": "9:1:101", "job_id": 101, "date": "2026-07-13",
                "workflow": "Legacy", "lane": (
                    "build / image-build (apiserver, backend/Dockerfile, .)"
                ),
                "run_id": 9, "attempt": 1, "sha": "a",
                "conclusion": "failure", "failed": True, "duration": 5.0,
                "phases": {},
            },
            {
                "id": "9:1:102", "job_id": 102, "date": "2026-07-13",
                "workflow": "Legacy", "lane": "database main",
                "run_id": 9, "attempt": 1, "sha": "a",
                "conclusion": "failure", "failed": True, "duration": 20.0,
                "phases": {},
            },
        ]
        logs = {
            101: (
                'Get "https://registry-1.docker.io/v2/": '
                'Client.Timeout exceeded while awaiting headers\n'
            ),
            102: 'Missing branch image artifacts after producer completion grace: apiserver\n',
        }

        def fake_api_request(_token, url, raw=False):
            self.assertTrue(raw)
            return logs[int(url.split('/')[-2])].encode()

        with mock.patch.object(chr_mod, 'api_request', side_effect=fake_api_request):
            enriched, events, errors = chr_mod.group_image_producer_failures(
                'token', 'kubeflow/pipelines', observations
            )

        self.assertEqual(errors, 0)
        self.assertEqual(
            enriched[0]["api_result_class"], "infrastructure_failure"
        )
        self.assertEqual(
            enriched[0]["api_signatures"], {"external_registry_timeout": 1}
        )
        self.assertEqual(events[0]["artifact"], "apiserver")
        self.assertEqual(events[0]["affected_lanes"], ["database main"])
        self.assertEqual(events[0]["impacted_failures"], 2)
        self.assertEqual(events[0]["status_correlation"], "none_reported")

        results = [{
            "generated_at": "2026-07-13T00:20:00Z",
            "workflow": "Legacy", "report_name": "database main",
            "result": "infrastructure_failure",
            "dimensions": {}, "signatures": {"missing_image_artifact": 1},
            "tests": [],
        }]
        snapshot = chr_mod.aggregate_daily(
            enriched, results, {}, infrastructure_events=events
        )[0]
        self.assertEqual(snapshot["totals"]["failures"], 2)
        self.assertEqual(snapshot["failure_classes"]["infrastructure_failure"], 2)
        self.assertNotIn("unclassified_failure", snapshot["failure_classes"])
        self.assertEqual(snapshot["signatures"]["external_registry_timeout"], 1)
        self.assertEqual(snapshot["signatures"]["missing_image_artifact"], 1)
        self.assertEqual(snapshot["infrastructure_events"], events)

    def test_rerun_rescue_is_attributed_only_to_latest_attempt_day(self):
        observations = [
            {
                "id": "8:1:1", "date": "2026-07-12", "workflow": "WF",
                "lane": "lane", "run_id": 8, "attempt": 1, "sha": "a",
                "run_created": "2026-07-12T23:00:00Z",
                "completed": "2026-07-12T23:20:00Z",
                "conclusion": "failure", "failed": True, "duration": 20.0,
                "phases": {},
            },
            {
                "id": "8:2:2", "date": "2026-07-13", "workflow": "WF",
                "lane": "lane", "run_id": 8, "attempt": 2, "sha": "a",
                "run_created": "2026-07-12T23:00:00Z",
                "completed": "2026-07-13T00:10:00Z",
                "conclusion": "success", "failed": False, "duration": 10.0,
                "phases": {},
            },
        ]
        snapshots = chr_mod.aggregate_daily(observations, [], {8: 2})

        self.assertEqual(snapshots[0]["totals"]["reruns"], 0)
        self.assertEqual(snapshots[0]["totals"]["rerun_rescues"], 0)
        self.assertEqual(snapshots[1]["totals"]["reruns"], 1)
        self.assertEqual(snapshots[1]["totals"]["rerun_rescues"], 1)
        self.assertEqual(snapshots[1]["totals"]["time_to_green"]["p50"], 70.0)

    def test_merge_replaces_overlap_and_never_prunes_older_days(self):
        history = {
            "schema_version": 1,
            "days": [
                {"date": "2020-01-01", "totals": {"lane_runs": 1}},
                {"date": "2026-07-13", "totals": {"lane_runs": 2}},
            ],
        }
        merged = chr_mod.merge_history(
            history, [{"date": "2026-07-13", "totals": {"lane_runs": 3}}],
        )
        self.assertEqual(
            [(day["date"], day["totals"]["lane_runs"]) for day in merged["days"]],
            [("2020-01-01", 1), ("2026-07-13", 3)],
        )

    def test_load_history_reconstructs_sorted_daily_snapshots(self):
        with tempfile.TemporaryDirectory() as directory:
            for date, lane_runs in (
                ("2026-07-13", 2),
                ("2026-07-12", 1),
            ):
                with open(
                    os.path.join(directory, f"{date}.json"),
                    "w",
                    encoding="utf-8",
                ) as output:
                    json.dump({
                        "date": date,
                        "totals": {"lane_runs": lane_runs},
                    }, output)

            history = chr_mod.load_history(directory)

        self.assertEqual(
            [(day["date"], day["totals"]["lane_runs"]) for day in history["days"]],
            [("2026-07-12", 1), ("2026-07-13", 2)],
        )

    def test_load_history_accepts_legacy_combined_file_in_directory(self):
        with tempfile.TemporaryDirectory() as directory:
            with open(
                os.path.join(directory, "history.json"),
                "w",
                encoding="utf-8",
            ) as output:
                json.dump({
                    "schema_version": chr_mod.HISTORY_SCHEMA_VERSION,
                    "days": [{"date": "2026-07-12"}],
                }, output)

            history = chr_mod.load_history(directory)

        self.assertEqual(history["days"], [{"date": "2026-07-12"}])

    def test_ci_result_artifact_highest_retry_wins(self):
        artifacts = [
            {"name": "ci-result - lane", "archive_download_url": "base"},
            {"name": "ci-result - lane - retry-2", "archive_download_url": "retry"},
            {"name": "junit-xml - lane", "archive_download_url": "junit"},
        ]
        selected = chr_mod.select_ci_result_artifacts(artifacts)
        self.assertEqual([artifact["archive_download_url"] for artifact in selected], ["retry"])

    def test_missing_result_runner_loss_is_classified_from_annotation(self):
        run = make_run(9, conclusion="failure")
        jobs = {
            1: [
                {
                    **make_job("End to End A Tests", "success"),
                    "id": 1,
                    "check_run_url": "checks/1",
                },
                {
                    **make_job("End to End B Tests", "failure"),
                    "id": 2,
                    "check_run_url": "checks/2",
                },
            ]
        }
        results = [{
            "schema_version": 1,
            "generated_at": "2026-07-13T00:10:00Z",
            "workflow": "WF",
            "report_name": "A",
            "run_id": 9,
            "run_attempt": 1,
            "result": "success",
        }]

        with mock.patch.object(
            chr_mod,
            "api_request",
            return_value=[{
                "message": "The hosted runner lost communication with the server."
            }],
        ):
            fallbacks, missing, observations = chr_mod.missing_result_fallbacks(
                "t", "e2e-test.yml", run, jobs, results
            )

        self.assertEqual(missing, 1)
        self.assertEqual([result["result"] for result in fallbacks], ["runner_lost"])
        self.assertEqual(observations, [])

    def test_cancelled_job_timeout_becomes_failure_observation(self):
        run = make_run(10, conclusion="cancelled")
        jobs = {
            1: [
                {
                    **make_job("KFP Webhooks - K8s v1.36.1", "success"),
                    "id": 1,
                    "check_run_url": "checks/1",
                },
                {
                    **make_job("KFP Webhooks - K8s v1.33.12", "cancelled"),
                    "id": 2,
                    "check_run_url": "checks/2",
                },
            ]
        }
        results = [{
            "schema_version": 1,
            "generated_at": "2026-07-13T00:10:00Z",
            "workflow": "WF",
            "report_name": "KFP Webhooks - K8s v1.36.1",
            "run_id": 10,
            "run_attempt": 1,
            "result": "success",
        }]

        with mock.patch.object(
            chr_mod,
            "api_request",
            return_value=[{
                "message": "The job has exceeded the maximum execution time of 40m0s"
            }],
        ):
            fallbacks, missing, observations = chr_mod.missing_result_fallbacks(
                "t", "kfp-webhooks.yml", run, jobs, results
            )

        self.assertEqual(missing, 1)
        self.assertEqual([result["result"] for result in fallbacks], ["job_timeout"])
        self.assertEqual(len(observations), 1)
        self.assertTrue(observations[0]["failed"])

    def test_concurrency_cancelled_job_remains_a_non_result(self):
        run = make_run(10, conclusion="cancelled")
        jobs = {
            1: [
                {
                    **make_job("KFP Webhooks - K8s v1.36.1", "success"),
                    "id": 1,
                    "check_run_url": "checks/1",
                },
                {
                    **make_job("KFP Webhooks - K8s v1.33.12", "cancelled"),
                    "id": 2,
                    "check_run_url": "checks/2",
                },
            ]
        }
        results = [{
            "schema_version": 1,
            "generated_at": "2026-07-13T00:10:00Z",
            "workflow": "WF",
            "report_name": "KFP Webhooks - K8s v1.36.1",
            "run_id": 10,
            "run_attempt": 1,
            "result": "success",
        }]

        with mock.patch.object(
            chr_mod,
            "api_request",
            return_value=[{"message": "The operation was canceled."}],
        ):
            fallbacks, missing, observations = chr_mod.missing_result_fallbacks(
                "t", "kfp-webhooks.yml", run, jobs, results
            )

        self.assertEqual((fallbacks, missing, observations), ([], 0, []))

    def test_no_results_is_treated_as_pre_rollout_not_missing(self):
        run = make_run(11, conclusion="failure")
        jobs = {
            1: [{
                **make_job("KFP Webhooks - K8s v1.36.1", "failure"),
                "id": 1,
                "check_run_url": "checks/1",
            }]
        }
        fallbacks, missing, observations = chr_mod.missing_result_fallbacks(
            "t", "kfp-webhooks.yml", run, jobs, []
        )
        self.assertEqual((fallbacks, missing, observations), ([], 0, []))

    def test_window_totals_include_specific_infrastructure_classes(self):
        history = {
            "days": [{
                "date": "2026-07-13",
                "totals": {
                    "lane_runs": 4,
                    "failures": 4,
                    "github_correlated_failures": 2,
                    "reruns": 0,
                    "rerun_rescues": 0,
                },
                "failure_classes": {
                    "runner_lost": 1,
                    "job_timeout": 1,
                    "missing_result": 1,
                    "infrastructure_failure": 1,
                    "unknown_failure": 1,
                },
            }]
        }
        totals = chr_mod.window_totals(
            history, "2026-07-13", "2026-07-14"
        )
        self.assertEqual(totals["infrastructure_failures"], 5)
        self.assertEqual(totals["github_correlated_failures"], 2)

    def test_site_writes_lightweight_manifest_and_individual_daily_history(self):
        history = {
            "schema_version": 1,
            "generated_at": "2026-07-14T00:00:00Z",
            "days": [{"date": "2026-07-13", "totals": {"lane_runs": 3}}],
        }
        with tempfile.TemporaryDirectory() as directory:
            source = os.path.join(directory, "source.html")
            with open(source, "w", encoding="utf-8") as output:
                output.write("<html></html>")
            site = os.path.join(directory, "site")
            chr_mod.write_site(site, history, "report", source)
            with open(
                os.path.join(site, "data", "daily", "2026-07-13.json"),
                encoding="utf-8",
            ) as daily:
                snapshot = json.load(daily)
            with open(
                os.path.join(site, "data", "index.json"),
                encoding="utf-8",
            ) as index_file:
                manifest = json.load(index_file)
            combined_history_exists = os.path.exists(
                os.path.join(site, "data", "history.json")
            )

        self.assertEqual(snapshot["totals"]["lane_runs"], 3)
        self.assertEqual(manifest, {
            "schema_version": chr_mod.HISTORY_SCHEMA_VERSION,
            "generated_at": "2026-07-14T00:00:00Z",
            "days": ["2026-07-13"],
        })
        self.assertFalse(combined_history_exists)

    def test_dashboard_lazily_loads_presets_and_custom_date_ranges(self):
        dashboard = os.path.join(
            os.path.dirname(__file__), "..", "ci-health-dashboard", "index.html"
        )
        with open(dashboard, encoding="utf-8") as source:
            content = source.read()

        for value in ("7", "14", "28", "90", "all", "custom"):
            self.assertIn(f'<option value="{value}"', content)
        self.assertIn('<input id="startDate" type="date">', content)
        self.assertIn('<input id="endDate" type="date">', content)
        self.assertIn("vs prior equal period", content)
        self.assertIn("raw.githubusercontent.com/kubeflow/pipelines/ci-metrics/data", content)
        self.assertIn("`${DATA_ROOT}/index.json?v=${Date.now()}`", content)
        self.assertIn("`${DATA_ROOT}/daily/${date}.json?v=${version}`", content)
        self.assertIn("state.pendingDays", content)
        self.assertGreaterEqual(content.count('class="help-tip'), 23)
        self.assertIn(".help-tip:focus-visible::after", content)
        self.assertIn("grid-template-columns: repeat(3, 1fr)", content)
        self.assertIn("button.setAttribute('aria-label', button.dataset.help)", content)
        self.assertIn("ordinary cancellations are excluded", content)
        self.assertIn("Correlation is not causation", content)
        self.assertIn("The median of the stored daily p95 elapsed durations", content)
        self.assertIn(".eyebrow { margin-bottom: 16px;", content)
        self.assertIn("<title>KFP CI Signal</title>", content)
        self.assertIn("<h1>CI signal</h1>", content)
        self.assertNotIn("CI Signal Room", content)
        self.assertNotIn("—", content)
        self.assertNotIn("fetch('./data/history.json')", content)

    def test_health_workflow_restores_and_persists_daily_snapshots(self):
        workflow = os.path.join(
            os.path.dirname(__file__), "..", "..", "workflows",
            "ci-health-report.yml",
        )
        with open(workflow, encoding="utf-8") as source:
            content = source.read()

        self.assertIn("git archive refs/remotes/origin/ci-metrics data/daily", content)
        self.assertIn(
            "HISTORY_INPUT: ${{ runner.temp }}/ci-health-history",
            content,
        )
        self.assertIn('cp "$SITE_DIR/data/index.json"', content)
        self.assertIn("git add -A data", content)
        self.assertIn(
            'rm -rf "$RUNNER_TEMP/ci-health-site/data"',
            content,
        )

    def test_disabled_pages_summary_does_not_publish_a_dead_link(self):
        history = {"schema_version": 1, "days": []}
        report = chr_mod.render_trend_summary(
            history, [], "https://example.invalid", pages_enabled=False
        )
        self.assertIn("enable GitHub Pages", report)
        self.assertNotIn("https://example.invalid", report)

    def test_summary_links_correlated_github_incident_without_hiding_failures(self):
        today = chr_mod.datetime.now(chr_mod.timezone.utc).date().isoformat()
        history = {
            "schema_version": 1,
            "days": [{
                "date": today,
                "totals": {
                    "lane_runs": 2,
                    "failures": 2,
                    "github_correlated_failures": 1,
                    "reruns": 0,
                    "rerun_rescues": 0,
                },
                "failure_classes": {"unclassified_failure": 2},
                "github_incidents": [{
                    "id": "incident-1",
                    "name": "Actions disruption",
                    "url": "https://stspg.io/example",
                    "impact": "major",
                    "started_at": f"{today}T00:00:00Z",
                    "resolved_at": f"{today}T01:00:00Z",
                    "components": ["Actions"],
                    "strict_overlaps": 1,
                    "nearby_matches": 0,
                }],
                "lanes": [],
                "tests": [],
            }],
        }

        report = chr_mod.render_trend_summary(
            history, [], "https://example.invalid"
        )

        self.assertIn("| Latest 7 days | 2 | 2 |", report)
        self.assertIn("[Actions disruption](https://stspg.io/example)", report)
        self.assertIn("not proof of causation", report)


class UpsertIssueTest(unittest.TestCase):
    def test_updates_existing_issue(self):
        calls = []

        def fake_api(token, url, method="GET", body=None, raw=False):
            calls.append((method, url))
            if method == "POST" and url.endswith("/labels"):
                return {}
            if "issues?labels=" in url:
                return [{"number": 42, "title": chr_mod.ISSUE_TITLE, "html_url": "issue-url"}]
            if method == "PATCH":
                return {}
            raise AssertionError(f"unexpected call {method} {url}")

        with mock.patch.object(chr_mod, "api_request", side_effect=fake_api):
            url = chr_mod.upsert_issue("t", "o/r", "report")
        self.assertEqual(url, "issue-url")
        self.assertIn(("PATCH", f"{chr_mod.API_ROOT}/repos/o/r/issues/42"), calls)

    def test_skips_pull_requests_and_wrong_titles(self):
        patched = []

        def fake_api(token, url, method="GET", body=None, raw=False):
            if method == "POST" and url.endswith("/labels"):
                return {}
            if "issues?labels=" in url:
                return [
                    {"number": 1, "title": chr_mod.ISSUE_TITLE,
                     "pull_request": {}, "html_url": "pr-url"},
                    {"number": 2, "title": "Something else entirely",
                     "html_url": "other-url"},
                ]
            if method == "PATCH":
                patched.append(url)
                return {}
            if method == "POST" and url.endswith("/issues"):
                return {"html_url": "new-issue-url"}
            raise AssertionError(f"unexpected call {method} {url}")

        with mock.patch.object(chr_mod, "api_request", side_effect=fake_api):
            url = chr_mod.upsert_issue("t", "o/r", "report")
        self.assertEqual(patched, [])  # neither the PR nor the unrelated issue
        self.assertEqual(url, "new-issue-url")

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
