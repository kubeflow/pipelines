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
TARGET_WORKFLOWS = [
    "e2e-test.yml",
    "api-server-tests.yml",
    "integration-tests-v1.yml",
    "legacy-v2-api-integration-tests.yml",
    "kfp-kubernetes-execution-tests.yml",
    "sdk-execution.yml",
    "upgrade-test.yml",
    "kfp-webhooks.yml",
]

ISSUE_TITLE = "CI Health Report (automated)"
ISSUE_LABEL = "ci-health"


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
    items = []
    page_url = url
    for _ in range(max_pages):
        payload = api_request(token, page_url)
        chunk = payload.get(key, []) if isinstance(payload, dict) else payload
        items.extend(chunk)
        if len(chunk) < 100:
            break
        separator = "&" if "?" in url else "?"
        page_number = len(items) // 100 + 1
        page_url = f"{url}{separator}page={page_number}"
    return items


def duration_minutes(job):
    started, completed = job.get("started_at"), job.get("completed_at")
    if not started or not completed:
        return None
    fmt = "%Y-%m-%dT%H:%M:%SZ"
    delta = datetime.strptime(completed, fmt) - datetime.strptime(started, fmt)
    minutes = delta.total_seconds() / 60
    return minutes if minutes >= 0 else None


def collect_lane_stats(token, repo, since, max_runs):
    """Returns (lane stats dict, failed run ids, rerun count, truncated flag)."""
    lanes = collections.defaultdict(
        lambda: {"total": 0, "failed": 0, "cancelled": 0, "durations": []}
    )
    failed_run_ids = []
    reruns = 0
    truncated = False

    for workflow in TARGET_WORKFLOWS:
        url = (
            f"{API_ROOT}/repos/{repo}/actions/workflows/{workflow}/runs"
            f"?branch=master&created=>={since}&per_page=100"
        )
        try:
            runs = paginate(token, url, "workflow_runs", max_pages=1)[:max_runs]
        except urllib.error.HTTPError as error:
            if error.code == 404:  # workflow renamed/removed
                continue
            raise
        except RateLimited:
            truncated = True
            break

        for run in runs:
            if run.get("status") != "completed":
                continue
            if run.get("run_attempt", 1) > 1:
                reruns += 1
            if run.get("conclusion") == "failure":
                failed_run_ids.append(run["id"])
            try:
                jobs = paginate(
                    token,
                    f"{API_ROOT}/repos/{repo}/actions/runs/{run['id']}/jobs"
                    "?filter=latest&per_page=100",
                    "jobs",
                )
            except RateLimited:
                truncated = True
                break
            for job in jobs:
                if job.get("conclusion") in (None, "skipped"):
                    continue
                lane = lanes[(run.get("name") or workflow, job["name"])]
                lane["total"] += 1
                if job["conclusion"] == "failure":
                    lane["failed"] += 1
                elif job["conclusion"] == "cancelled":
                    lane["cancelled"] += 1
                minutes = duration_minutes(job)
                if minutes is not None:
                    lane["durations"].append(minutes)
        if truncated:
            break
    return lanes, failed_run_ids, reruns, truncated


def collect_failed_tests(token, repo, failed_run_ids, max_junit_runs):
    """Tallies failed testcases from junit-xml artifacts of failed runs."""
    failed_tests = collections.Counter()
    artifact_runs_seen = 0
    for run_id in failed_run_ids[:max_junit_runs]:
        try:
            artifacts = paginate(
                token,
                f"{API_ROOT}/repos/{repo}/actions/runs/{run_id}/artifacts?per_page=100",
                "artifacts",
                max_pages=1,
            )
        except (urllib.error.HTTPError, RateLimited):
            continue
        for artifact in artifacts:
            if not artifact.get("name", "").startswith("junit-xml - "):
                continue
            if artifact.get("expired"):
                continue
            try:
                content = api_request(
                    token, artifact["archive_download_url"], raw=True
                )
                archive = zipfile.ZipFile(io.BytesIO(content))
            except Exception:  # noqa: BLE001 - best-effort artifact ingestion
                continue
            artifact_runs_seen += 1
            for member in archive.namelist():
                if not member.endswith(".xml"):
                    continue
                try:
                    root = ET.fromstring(archive.read(member))
                except ET.ParseError:
                    continue
                for case in root.iter("testcase"):
                    if case.find("failure") is not None or case.find("error") is not None:
                        failed_tests[case.get("name") or "<unnamed>"] += 1
    return failed_tests, artifact_runs_seen


def render_report(lanes, failed_tests, artifact_runs_seen, reruns, days, truncated):
    lines = []
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    lines.append(f"# CI Health Report\n")
    lines.append(
        f"_Master-branch runs, last {days} days. Generated {now}."
        + (" **Rate-limited: results truncated.**" if truncated else "")
        + "_\n"
    )

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
        f"{total_failures} failures, {reruns} workflow re-runs.\n"
    )

    lines.append("## Flakiest tests (from junit-xml artifacts of failed runs)\n")
    if failed_tests:
        lines.append(f"_Parsed junit artifacts from {artifact_runs_seen} artifact(s)._\n")
        lines.append("| Test | Failures |")
        lines.append("|---|---:|")
        for name, count in failed_tests.most_common(15):
            lines.append(f"| {name} | {count} |")
        lines.append("")
    else:
        lines.append(
            "_No `junit-xml - *` artifacts found on failed runs in the window; "
            "per-test data populates as runs upload them (see test-and-report)._\n"
        )
    return "\n".join(lines) + "\n"


def upsert_issue(token, repo, report):
    try:
        api_request(
            token,
            f"{API_ROOT}/repos/{repo}/labels",
            method="POST",
            body={"name": ISSUE_LABEL, "color": "1d76db", "description": "Automated CI health reports"},
        )
    except urllib.error.HTTPError as error:
        if error.code != 422:  # 422 = label already exists
            raise

    issues = api_request(
        token,
        f"{API_ROOT}/repos/{repo}/issues?labels={ISSUE_LABEL}&state=open&per_page=10",
    )
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

    lanes, failed_run_ids, reruns, truncated = collect_lane_stats(
        token, repo, since, max_runs
    )
    failed_tests, artifact_runs_seen = collect_failed_tests(
        token, repo, failed_run_ids, max_junit_runs
    )
    report = render_report(
        lanes, failed_tests, artifact_runs_seen, reruns, days, truncated
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
