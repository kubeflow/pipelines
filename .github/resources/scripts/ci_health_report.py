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
"""Generates a CI health report for master-branch workflow runs.

Aggregates job-level failure rates and durations across the heavy test
workflows, and per-test failure counts from `junit-xml - *` artifacts uploaded
by failed runs (see test-and-report). Writes markdown to stdout and, when
available, to the GitHub job summary; optionally upserts a single tracking
issue labeled `ci-health` so flake trends live at a stable URL.

GitHub's built-in Actions performance metrics stop at job granularity and have
no API; this report adds the per-test layer and a public, automatable record.

Accuracy contract:
  - Every attempt of re-run workflows is counted, so a flake that passed on
    re-run still shows its original failure.
  - Cancelled/skipped jobs are excluded from rates; every other non-success
    conclusion (failure, timed_out, stale, ...) counts as a failure.
  - Any truncation (run caps, rate limits, artifact ingestion errors) is
    surfaced in the report instead of silently publishing partial data.

Environment:
  GITHUB_TOKEN             required, API token
  GITHUB_REPOSITORY        owner/repo (default kubeflow/pipelines)
  DAYS                     lookback window in days (default 14)
  MAX_RUNS_PER_WORKFLOW    cap per workflow (default 40)
  MAX_JUNIT_RUNS           failed runs to pull junit artifacts from (default 15)
  UPDATE_ISSUE             "true" to upsert the ci-health issue (default false)

Uses only the Python standard library.
"""

import collections
import io
import json
import os
import sys
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET
import zipfile
from datetime import datetime, timedelta, timezone

API_ROOT = "https://api.github.com"

# Heavy master-branch test workflows worth tracking for flake health.
# Missing entries are surfaced in the report notes rather than silently
# skipped, so workflow renames/deletions get noticed instead of permanently
# dropping a suite from the health data.
TARGET_WORKFLOWS = [
    "e2e-test.yml",
    "api-server-tests.yml",
    "integration-tests-v1.yml",
    "legacy-v2-api-integration-tests.yml",
    "kfp-sdk-client-tests.yml",
    "upgrade-test.yml",
    "kfp-webhooks.yml",
]

ISSUE_TITLE = "CI Health Report (automated)"
ISSUE_LABEL = "ci-health"

# Job conclusions that are not results at all: master-push workflows cancel
# in-progress runs, so counting cancellations as either success or failure
# biases rates. Everything not listed here and not "success" is a failure
# (failure, timed_out, stale, ...).
NON_RESULT_CONCLUSIONS = {None, "", "skipped", "cancelled", "neutral", "action_required"}


class RateLimited(Exception):
    pass


def build_opener_without_redirects():
    """Opener that surfaces redirects instead of following them.

    Artifact downloads redirect to blob storage; forwarding the GitHub
    Authorization header there makes the storage backend reject the request,
    so the redirect target must be fetched without it.
    """

    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            return None

    return urllib.request.build_opener(NoRedirect)


def api_request(token, url, method="GET", body=None, raw=False):
    headers = {
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {token}",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    data = json.dumps(body).encode() if body is not None else None
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    opener = build_opener_without_redirects()
    try:
        with opener.open(request, timeout=60) as response:
            content = response.read()
            return content if raw else json.loads(content or b"{}")
    except urllib.error.HTTPError as error:
        if error.code in (301, 302, 303, 307, 308):
            location = error.headers.get("Location")
            if location:
                # Redirect target (blob storage) must not receive the token.
                with urllib.request.urlopen(
                    urllib.request.Request(location), timeout=120
                ) as redirected:
                    return redirected.read()
        if error.code == 403 and error.headers.get("X-RateLimit-Remaining") == "0":
            raise RateLimited(url) from error
        raise


def paginate(token, url, key, max_pages=3):
    """Returns (items, may_have_more).

    may_have_more is True when the page budget ran out while pages were still
    full and the API's total_count confirms that more results exist. The caller
    must surface that as a completeness gap instead of silently publishing
    partial results.
    """
    items = []
    page_url = url
    for _ in range(max_pages):
        payload = api_request(token, page_url)
        chunk = payload.get(key, []) if isinstance(payload, dict) else payload
        items.extend(chunk)
        total_count = payload.get("total_count") if isinstance(payload, dict) else None
        if isinstance(total_count, int) and len(items) >= total_count:
            return items, False
        if len(chunk) < 100:
            return items, False
        separator = "&" if "?" in url else "?"
        page_number = len(items) // 100 + 1
        page_url = f"{url}{separator}page={page_number}"
    return items, True


def duration_minutes(job):
    started, completed = job.get("started_at"), job.get("completed_at")
    if not started or not completed:
        return None
    fmt = "%Y-%m-%dT%H:%M:%SZ"
    delta = datetime.strptime(completed, fmt) - datetime.strptime(started, fmt)
    minutes = delta.total_seconds() / 60
    return minutes if minutes >= 0 else None


def is_failure_conclusion(conclusion):
    return conclusion not in NON_RESULT_CONCLUSIONS and conclusion != "success"


def run_attempt_job_urls(repo, run):
    """Job listing URLs covering every attempt of a run.

    `filter=latest` alone would hide the original failed jobs of re-run
    workflows — exactly the flakes this report exists to measure.
    """
    attempts = run.get("run_attempt", 1) or 1
    if attempts <= 1:
        return [
            f"{API_ROOT}/repos/{repo}/actions/runs/{run['id']}/jobs"
            "?filter=latest&per_page=100"
        ]
    return [
        f"{API_ROOT}/repos/{repo}/actions/runs/{run['id']}/attempts/{attempt}/jobs"
        "?per_page=100"
        for attempt in range(1, attempts + 1)
    ]


def collect_lane_stats(token, repo, since, max_runs):
    """Aggregates per-lane job outcomes across every attempt of each run.

    Returns (lanes, failed_runs, reruns, notes) where failed_runs is a list of
    (created_at, run_id) for runs with a failed job in any attempt, and notes
    is a list of data-completeness warnings for the report header.
    """
    lanes = collections.defaultdict(lambda: {"total": 0, "failed": 0, "durations": []})
    failed_runs = []
    reruns = 0
    notes = []
    capped_workflows = []
    missing_workflows = []
    job_fetch_errors = 0
    job_page_truncations = 0

    for workflow in TARGET_WORKFLOWS:
        url = (
            f"{API_ROOT}/repos/{repo}/actions/workflows/{workflow}/runs"
            f"?branch=master&created=>={since}&per_page=100"
        )
        try:
            runs, more_runs = paginate(token, url, "workflow_runs", max_pages=1)
        except urllib.error.HTTPError as error:
            if error.code == 404:
                # A renamed/deleted workflow silently dropping out would
                # permanently omit that suite's health; make it visible.
                missing_workflows.append(f"`{workflow}`")
                continue
            raise
        except RateLimited:
            notes.append(
                f"rate-limited while listing `{workflow}` runs; later workflows omitted"
            )
            break

        if len(runs) > max_runs:
            capped_workflows.append(f"`{workflow}` ({len(runs)}→{max_runs})")
            runs = runs[:max_runs]
        elif more_runs:
            capped_workflows.append(f"`{workflow}` (page budget)")

        rate_limited = False
        for run in runs:
            if run.get("status") != "completed":
                continue
            if (run.get("run_attempt", 1) or 1) > 1:
                reruns += 1
            run_had_failure = False
            for jobs_url in run_attempt_job_urls(repo, run):
                try:
                    jobs, more_jobs = paginate(token, jobs_url, "jobs")
                    if more_jobs:
                        job_page_truncations += 1
                except RateLimited:
                    notes.append(
                        f"rate-limited while reading jobs for `{workflow}`; "
                        "results truncated"
                    )
                    rate_limited = True
                    break
                except urllib.error.HTTPError:
                    # Attempt data can expire; skip just this attempt, but a
                    # silently dropped job listing must not read as a healthy
                    # lane — surface the gap in the completeness notes.
                    job_fetch_errors += 1
                    continue
                for job in jobs:
                    conclusion = job.get("conclusion")
                    if conclusion in NON_RESULT_CONCLUSIONS:
                        continue
                    lane = lanes[(run.get("name") or workflow, job["name"])]
                    lane["total"] += 1
                    if is_failure_conclusion(conclusion):
                        lane["failed"] += 1
                        run_had_failure = True
                    minutes = duration_minutes(job)
                    if minutes is not None:
                        lane["durations"].append(minutes)
            if run_had_failure:
                failed_runs.append((run.get("created_at") or "", run["id"]))
            if rate_limited:
                break
        if rate_limited:
            break

    if missing_workflows:
        notes.append(
            "tracked workflow(s) not found (renamed/removed — update "
            "TARGET_WORKFLOWS): " + ", ".join(missing_workflows)
        )
    if capped_workflows:
        notes.append("run cap applied: " + ", ".join(capped_workflows))
    if job_fetch_errors:
        notes.append(
            f"{job_fetch_errors} job listing(s) failed with HTTP errors; "
            "lane rates may undercount failures"
        )
    if job_page_truncations:
        notes.append(
            f"{job_page_truncations} job listing(s) exceeded the page budget; "
            "lane rates may undercount"
        )
    return lanes, failed_runs, reruns, notes


def collect_failed_tests(token, repo, failed_runs, max_junit_runs):
    """Tallies failed testcases from junit-xml artifacts of failed runs.

    Newest failed runs first, across all workflows, so one busy workflow
    cannot monopolize the artifact budget. Returns
    (failed_tests, artifacts_parsed, ingestion_errors, scanned_run_count).
    """
    failed_tests = collections.Counter()
    artifacts_parsed = 0
    ingestion_errors = 0

    newest_first = [run_id for _, run_id in sorted(failed_runs, reverse=True)]
    scanned_runs = newest_first[:max_junit_runs]
    for run_id in scanned_runs:
        try:
            artifacts, more_artifacts = paginate(
                token,
                f"{API_ROOT}/repos/{repo}/actions/runs/{run_id}/artifacts?per_page=100",
                "artifacts",
                max_pages=1,
            )
            if more_artifacts:
                # Artifacts beyond the first page are invisible; that is an
                # ingestion gap, not a clean zero.
                ingestion_errors += 1
        except (urllib.error.HTTPError, RateLimited, OSError):
            ingestion_errors += 1
            continue
        for artifact in artifacts:
            if not artifact.get("name", "").startswith("junit-xml - "):
                continue
            if artifact.get("expired"):
                continue
            try:
                content = api_request(token, artifact["archive_download_url"], raw=True)
                archive = zipfile.ZipFile(io.BytesIO(content))
            except Exception:  # noqa: BLE001 - best-effort artifact ingestion
                ingestion_errors += 1
                continue
            artifacts_parsed += 1
            for member in archive.namelist():
                if not member.endswith(".xml"):
                    continue
                try:
                    root = ET.fromstring(archive.read(member))
                except ET.ParseError:
                    ingestion_errors += 1
                    continue
                for case in root.iter("testcase"):
                    if case.find("failure") is not None or case.find("error") is not None:
                        # name alone is only unique within a suite/class;
                        # include classname so unrelated tests do not merge.
                        name = case.get("name") or "<unnamed>"
                        classname = case.get("classname") or ""
                        key = f"{classname} :: {name}" if classname else name
                        failed_tests[key] += 1
    return failed_tests, artifacts_parsed, ingestion_errors, len(scanned_runs)


def render_report(
    lanes,
    failed_tests,
    artifacts_parsed,
    ingestion_errors,
    reruns,
    days,
    notes,
    junit_scanned=0,
    junit_total=0,
):
    lines = []
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    lines.append("# CI Health Report\n")
    lines.append(f"_Master-branch runs, last {days} days. Generated {now}._\n")
    if notes:
        lines.append("> **Data completeness:** " + "; ".join(notes) + "\n")

    ranked = sorted(
        lanes.items(), key=lambda item: (item[1]["failed"], item[1]["total"]), reverse=True
    )
    failing = [(key, stats) for key, stats in ranked if stats["failed"] > 0][:25]

    lines.append("## Failing lanes (top 25 by failure count)\n")
    if not failing:
        lines.append("No failing lanes in the window. 🎉\n")
    else:
        lines.append("| Workflow | Lane | Runs | Failures | Fail % | Avg min |")
        lines.append("|---|---|---:|---:|---:|---:|")
        for (workflow, lane), stats in failing:
            rate = 100 * stats["failed"] / stats["total"]
            avg = (
                sum(stats["durations"]) / len(stats["durations"])
                if stats["durations"]
                else 0
            )
            lines.append(
                f"| {workflow} | {lane} | {stats['total']} | {stats['failed']}"
                f" | {rate:.0f}% | {avg:.0f} |"
            )
        lines.append("")

    total_lanes = len(lanes)
    total_jobs = sum(stats["total"] for stats in lanes.values())
    total_failures = sum(stats["failed"] for stats in lanes.values())
    lines.append(
        f"**Window totals:** {total_jobs} lane runs across {total_lanes} lanes, "
        f"{total_failures} failures, {reruns} workflow re-runs "
        "(all attempts counted; cancelled/skipped excluded).\n"
    )

    lines.append("## Flakiest tests (from junit-xml artifacts of failed runs)\n")
    # Name the actual scan scope: the artifact budget covers only the newest
    # failed runs, and describing a capped scan as the whole window would let
    # artifact-less infrastructure failures mask real junit data on older runs.
    if junit_total > junit_scanned:
        lines.append(
            f"_Scanned the newest {junit_scanned} of {junit_total} failed runs "
            "for junit artifacts._\n"
        )
    else:
        lines.append(f"_Scanned {junit_scanned} failed run(s) for junit artifacts._\n")
    if ingestion_errors:
        lines.append(
            f"> **Note:** {ingestion_errors} artifact ingestion gap(s) "
            "(errors or page-budget truncation); per-test counts are a lower "
            "bound.\n"
        )
    if failed_tests:
        lines.append(f"_Parsed {artifacts_parsed} junit artifact(s)._\n")
        lines.append("| Test | Failures |")
        lines.append("|---|---:|")
        for name, count in failed_tests.most_common(15):
            lines.append(f"| {name} | {count} |")
        lines.append("")
    elif not ingestion_errors:
        lines.append(
            "_No `junit-xml - *` artifacts found on the scanned failed runs; "
            "per-test data populates as runs upload them (see test-and-report)._\n"
        )
    else:
        lines.append("_No junit artifacts could be ingested from the scanned runs._\n")
    return "\n".join(lines) + "\n"


def upsert_issue(token, repo, report):
    try:
        api_request(
            token,
            f"{API_ROOT}/repos/{repo}/labels",
            method="POST",
            body={
                "name": ISSUE_LABEL,
                "color": "1d76db",
                "description": "Automated CI health reports",
            },
        )
    except urllib.error.HTTPError as error:
        if error.code != 422:  # 422 = label already exists
            raise

    listed = api_request(
        token,
        f"{API_ROOT}/repos/{repo}/issues?labels={ISSUE_LABEL}&state=open&per_page=10",
    )
    # The issues endpoint also returns pull requests, and a label alone could
    # match unrelated content; only ever overwrite the bot's own report issue.
    issues = [
        item
        for item in listed
        if "pull_request" not in item and item.get("title") == ISSUE_TITLE
    ]
    body = {"title": ISSUE_TITLE, "body": report[:65000], "labels": [ISSUE_LABEL]}
    if issues:
        number = issues[0]["number"]
        api_request(
            token, f"{API_ROOT}/repos/{repo}/issues/{number}", method="PATCH", body=body
        )
        return issues[0]["html_url"]
    created = api_request(
        token, f"{API_ROOT}/repos/{repo}/issues", method="POST", body=body
    )
    return created["html_url"]


def main():
    token = os.environ.get("GITHUB_TOKEN")
    if not token:
        print("GITHUB_TOKEN is required", file=sys.stderr)
        return 1
    repo = os.environ.get("GITHUB_REPOSITORY", "kubeflow/pipelines")
    days = int(os.environ.get("DAYS", "14"))
    max_runs = int(os.environ.get("MAX_RUNS_PER_WORKFLOW", "40"))
    max_junit_runs = int(os.environ.get("MAX_JUNIT_RUNS", "15"))
    since = (datetime.now(timezone.utc) - timedelta(days=days)).strftime("%Y-%m-%d")

    lanes, failed_runs, reruns, notes = collect_lane_stats(token, repo, since, max_runs)
    failed_tests, artifacts_parsed, ingestion_errors, junit_scanned = (
        collect_failed_tests(token, repo, failed_runs, max_junit_runs)
    )
    report = render_report(
        lanes,
        failed_tests,
        artifacts_parsed,
        ingestion_errors,
        reruns,
        days,
        notes,
        junit_scanned=junit_scanned,
        junit_total=len(failed_runs),
    )

    print(report)
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_path:
        with open(summary_path, "a", encoding="utf-8") as summary:
            summary.write(report)

    if os.environ.get("UPDATE_ISSUE", "").lower() == "true":
        url = upsert_issue(token, repo, report)
        print(f"Updated tracking issue: {url}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
