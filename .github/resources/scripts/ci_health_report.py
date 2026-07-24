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
"""Generates durable CI health trends for master-branch workflow runs.

Combines GitHub job/step timing with compact `ci-result - *` artifacts from
successful and failed test lanes. It refreshes recent daily snapshots, merges
them into persisted history, renders a static dashboard dataset, writes the
current markdown summary, and optionally updates the stable `ci-health` issue.

Accuracy contract:
  - Every attempt of re-run workflows is counted, so a flake that passed on
    re-run still shows its original failure.
  - Cancelled/skipped jobs are excluded from rates; every other non-success
    conclusion (failure, timed_out, stale, ...) counts as a failure.
  - Any truncation (API limits, rate limits, artifact ingestion errors) is
    surfaced in the report instead of silently publishing partial data.

Environment:
  GITHUB_TOKEN             required, API token
  GITHUB_REPOSITORY        owner/repo (default kubeflow/pipelines)
  HISTORY_INPUT            existing durable history JSON (optional)
  BOOTSTRAP_DAYS           first-run lookback (default 90)
  REFRESH_DAYS             overlap refreshed for late reruns (default 14)
  OUTPUT_DIR               generated dashboard directory (default ci-health-site)
  DASHBOARD_URL            public dashboard URL
  DASHBOARD_SOURCE         static dashboard index.html source
  PAGES_ENABLED            whether the public dashboard link is live
  UPDATE_ISSUE             "true" to upsert the ci-health issue (default false)

Uses only the Python standard library.
"""

import collections
import io
import json
import math
import os
import re
import shutil
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
JUNIT_ARTIFACT_PREFIX = "junit-xml - "
CI_RESULT_ARTIFACT_PREFIX = "ci-result - "
RETRY_ARTIFACT_SUFFIX = re.compile(r"^(?P<base>.+) - retry-(?P<attempt>\d+)$")
HISTORY_SCHEMA_VERSION = 1

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


def parse_timestamp(value):
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def elapsed_minutes(start, end):
    started, completed = parse_timestamp(start), parse_timestamp(end)
    if not started or not completed or completed < started:
        return None
    return (completed - started).total_seconds() / 60


def percentile(values, percentage):
    """Linear-interpolated percentile with sensible zero/one-value behavior."""
    if not values:
        return None
    ordered = sorted(values)
    if len(ordered) == 1:
        return ordered[0]
    position = (len(ordered) - 1) * percentage / 100
    lower, upper = math.floor(position), math.ceil(position)
    if lower == upper:
        return ordered[lower]
    return ordered[lower] + (ordered[upper] - ordered[lower]) * (position - lower)


def job_phase_minutes(run, job):
    """Derives queue/setup/test/report phases from Actions job step timestamps."""
    phases = {
        "queue": elapsed_minutes(run.get("created_at"), job.get("started_at")),
        "setup": None,
        "bootstrap": None,
        "build": None,
        "deploy": None,
        "test": None,
        "report": None,
    }
    test_step = next(
        (
            step for step in job.get("steps", [])
            if (step.get("name") or "").split(" / ")[-1] == "Run Tests"
        ),
        None,
    )
    if test_step:
        phases["setup"] = elapsed_minutes(
            job.get("started_at"), test_step.get("started_at")
        )
        phases["test"] = elapsed_minutes(
            test_step.get("started_at"), test_step.get("completed_at")
        )
        phases["report"] = elapsed_minutes(
            test_step.get("completed_at"), job.get("completed_at")
        )
    for step in job.get("steps", []):
        name = (step.get("name") or "").lower()
        duration = elapsed_minutes(step.get("started_at"), step.get("completed_at"))
        if duration is None:
            continue
        if "deploy" in name:
            phases["deploy"] = (phases["deploy"] or 0) + duration
        elif "build" in name:
            phases["build"] = (phases["build"] or 0) + duration
        elif any(
            marker in name
            for marker in (
                "create cluster",
                "create kfp cluster",
                "set up",
                "setup",
                "restore",
                "download",
                "load image",
            )
        ):
            phases["bootstrap"] = (phases["bootstrap"] or 0) + duration
    return phases


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


def select_junit_artifacts(artifacts):
    """Selects the latest artifact for each possibly retried JUnit report."""
    selected_by_base_name = {}
    for artifact in artifacts:
        name = artifact.get("name", "")
        if not name.startswith(JUNIT_ARTIFACT_PREFIX) or artifact.get("expired"):
            continue

        retry_match = RETRY_ARTIFACT_SUFFIX.match(name)
        base_name = retry_match.group("base") if retry_match else name
        retry_attempt = int(retry_match.group("attempt")) if retry_match else -1
        selected = selected_by_base_name.get(base_name)
        if selected is None or retry_attempt > selected[0]:
            selected_by_base_name[base_name] = (retry_attempt, artifact)

    return [artifact for _, artifact in selected_by_base_name.values()]


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
        for artifact in select_junit_artifacts(artifacts):
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


def select_ci_result_artifacts(artifacts):
    """Selects the highest upload retry for each normalized-result artifact."""
    selected = {}
    for artifact in artifacts:
        name = artifact.get("name", "")
        if not name.startswith(CI_RESULT_ARTIFACT_PREFIX) or artifact.get("expired"):
            continue
        retry_match = RETRY_ARTIFACT_SUFFIX.match(name)
        base_name = retry_match.group("base") if retry_match else name
        attempt = int(retry_match.group("attempt")) if retry_match else -1
        current = selected.get(base_name)
        if current is None or attempt > current[0]:
            selected[base_name] = (attempt, artifact)
    return [artifact for _, artifact in selected.values()]


def read_ci_result_artifacts(token, repo, run_id):
    """Returns normalized results and an ingestion-gap count for one run."""
    try:
        artifacts, truncated = paginate(
            token,
            f"{API_ROOT}/repos/{repo}/actions/runs/{run_id}/artifacts?per_page=100",
            "artifacts",
            max_pages=2,
        )
    except (urllib.error.HTTPError, RateLimited, OSError):
        return [], 1
    errors = int(truncated)
    results = []
    for artifact in select_ci_result_artifacts(artifacts):
        try:
            content = api_request(token, artifact["archive_download_url"], raw=True)
            with zipfile.ZipFile(io.BytesIO(content)) as archive:
                members = [
                    member for member in archive.namelist()
                    if member.endswith(".json") and not member.endswith("/")
                ]
                if not members:
                    errors += 1
                    continue
                result = json.loads(archive.read(members[0]))
                if result.get("schema_version") != 1:
                    errors += 1
                    continue
                results.append(result)
        except (OSError, ValueError, KeyError, zipfile.BadZipFile, json.JSONDecodeError):
            errors += 1
    return results, errors


def collect_trend_data(token, repo, since):
    """Collects dated lane observations and normalized test results.

    Unlike the legacy rolling summary, this has no 40-run cap. Pagination is
    bounded only by GitHub's documented 1,000-result filtered-search limit,
    which is surfaced if reached. Daily runs normally scan only a two-day
    overlap; the first run bootstraps the requested history window.
    """
    observations = []
    normalized_results = []
    failed_runs = []
    notes = []
    rerun_runs = {}
    artifact_errors = 0

    for workflow in TARGET_WORKFLOWS:
        url = (
            f"{API_ROOT}/repos/{repo}/actions/workflows/{workflow}/runs"
            f"?branch=master&created=>={since}&per_page=100"
        )
        try:
            runs, truncated = paginate(token, url, "workflow_runs", max_pages=10)
        except urllib.error.HTTPError as error:
            if error.code == 404:
                notes.append(f"tracked workflow `{workflow}` was not found")
                continue
            raise
        except RateLimited:
            notes.append(f"rate-limited while listing `{workflow}` runs")
            break
        if truncated:
            notes.append(
                f"`{workflow}` reached GitHub's 1,000-run filtered-search limit"
            )

        for run in runs:
            if run.get("status") != "completed":
                continue
            attempts = run.get("run_attempt", 1) or 1
            rerun_runs[run["id"]] = attempts
            run_had_failure = False
            for attempt, jobs_url in enumerate(run_attempt_job_urls(repo, run), start=1):
                try:
                    jobs, truncated_jobs = paginate(
                        token, jobs_url, "jobs", max_pages=3
                    )
                except (urllib.error.HTTPError, RateLimited):
                    notes.append(f"job listing unavailable for run {run['id']}")
                    continue
                if truncated_jobs:
                    notes.append(f"job listing truncated for run {run['id']}")
                for job in jobs:
                    conclusion = job.get("conclusion")
                    if conclusion in NON_RESULT_CONCLUSIONS:
                        continue
                    created = (
                        job.get("started_at")
                        or run.get("created_at")
                        or datetime.now(timezone.utc).isoformat()
                    )
                    failed = is_failure_conclusion(conclusion)
                    run_had_failure = run_had_failure or failed
                    observations.append(
                        {
                            "id": f"{run['id']}:{attempt}:{job.get('id', job.get('name'))}",
                            "date": created[:10],
                            "workflow": run.get("name") or workflow,
                            "lane": job.get("name") or "<unnamed>",
                            "run_id": run["id"],
                            "attempt": attempt,
                            "sha": run.get("head_sha") or "",
                            "commit_message": (
                                (run.get("head_commit") or {}).get("message") or ""
                            ).splitlines()[0],
                            "run_created": run.get("created_at") or "",
                            "completed": job.get("completed_at") or "",
                            "conclusion": conclusion,
                            "failed": failed,
                            "duration": duration_minutes(job),
                            "phases": job_phase_minutes(run, job),
                        }
                    )
            if run_had_failure:
                failed_runs.append((run.get("created_at") or "", run["id"]))

            results, errors = read_ci_result_artifacts(token, repo, run["id"])
            normalized_results.extend(results)
            artifact_errors += errors

    if artifact_errors:
        notes.append(
            f"{artifact_errors} normalized-result artifact ingestion gap(s)"
        )
    return observations, normalized_results, failed_runs, rerun_runs, notes


def summarized_distribution(values):
    values = [value for value in values if value is not None]
    return {
        "p50": round(percentile(values, 50), 2) if values else None,
        "p95": round(percentile(values, 95), 2) if values else None,
    }


def aggregate_daily(observations, normalized_results, rerun_runs):
    """Aggregates replaceable daily snapshots from raw API/artifact records."""
    observations_by_day = collections.defaultdict(list)
    for observation in observations:
        observations_by_day[observation["date"]].append(observation)

    results_by_day = collections.defaultdict(list)
    for result in normalized_results:
        generated = result.get("generated_at") or ""
        if len(generated) >= 10:
            results_by_day[generated[:10]].append(result)

    rerun_events_by_day = collections.defaultdict(list)
    observations_by_run = collections.defaultdict(list)
    for observation in observations:
        observations_by_run[observation["run_id"]].append(observation)
    for run_id, run_observations in observations_by_run.items():
        attempts = rerun_runs.get(run_id, 1)
        if attempts <= 1:
            continue
        attempt_failures = collections.defaultdict(bool)
        for observation in run_observations:
            attempt_failures[observation["attempt"]] |= observation["failed"]
        latest_observations = [
            observation for observation in run_observations
            if observation["attempt"] == attempts
        ]
        if not latest_observations:
            continue
        earlier_failed = any(
            failed for attempt, failed in attempt_failures.items() if attempt < attempts
        )
        latest_failed = attempt_failures.get(attempts, False)
        latest_day = max(observation["date"] for observation in latest_observations)
        completed = max(
            (
                observation.get("completed", "")
                for observation in latest_observations
            ),
            default="",
        )
        created = run_observations[0].get("run_created")
        rerun_events_by_day[latest_day].append(
            {
                "rescued": earlier_failed and not latest_failed,
                "time_to_green": elapsed_minutes(created, completed),
            }
        )

    snapshots = []
    for day in sorted(set(observations_by_day) | set(results_by_day)):
        day_observations = observations_by_day[day]
        day_results = results_by_day[day]
        lanes = collections.defaultdict(
            lambda: {
                "runs": 0,
                "failures": 0,
                "durations": [],
                "phases": collections.defaultdict(list),
            }
        )
        all_durations = []
        all_phases = collections.defaultdict(list)
        failed_jobs = 0
        for observation in day_observations:
            lane = lanes[(observation["workflow"], observation["lane"])]
            lane["runs"] += 1
            lane["failures"] += int(observation["failed"])
            failed_jobs += int(observation["failed"])
            if observation["duration"] is not None:
                lane["durations"].append(observation["duration"])
                all_durations.append(observation["duration"])
            for phase, value in observation["phases"].items():
                if value is not None:
                    lane["phases"][phase].append(value)
                    all_phases[phase].append(value)

        failure_classes = collections.Counter()
        signatures = collections.Counter()
        tests = collections.defaultdict(lambda: {"executions": 0, "failures": 0, "skipped": 0})
        result_lanes = collections.defaultdict(
            lambda: {"runs": 0, "classes": collections.Counter(), "dimensions": {}}
        )
        for result in day_results:
            result_class = result.get("result") or "unknown"
            failure_classes[result_class] += 1
            signatures.update(result.get("signatures") or {})
            lane_name = result.get("report_name") or result.get("job") or "<unnamed>"
            result_lane = result_lanes[(result.get("workflow") or "<unknown>", lane_name)]
            result_lane["runs"] += 1
            result_lane["classes"][result_class] += 1
            result_lane["dimensions"] = result.get("dimensions") or {}
            for test in result.get("tests") or []:
                stats = tests[test.get("id") or "<unnamed>"]
                for field in ("executions", "failures", "skipped"):
                    stats[field] += int(test.get(field) or 0)

        classified_failures = sum(
            count for name, count in failure_classes.items() if name != "success"
        )
        if failed_jobs > classified_failures:
            failure_classes["unclassified_failure"] += failed_jobs - classified_failures

        rerun_events = rerun_events_by_day.get(day, [])
        reruns = len(rerun_events)
        rescued = sum(int(event["rescued"]) for event in rerun_events)
        time_to_green = [
            event["time_to_green"] for event in rerun_events
            if event["rescued"] and event["time_to_green"] is not None
        ]

        lane_rows = []
        for (workflow, lane_name), stats in sorted(lanes.items()):
            lane_rows.append(
                {
                    "workflow": workflow,
                    "lane": lane_name,
                    "runs": stats["runs"],
                    "failures": stats["failures"],
                    "duration": summarized_distribution(stats["durations"]),
                    "phases": {
                        phase: summarized_distribution(values)
                        for phase, values in stats["phases"].items()
                    },
                }
            )
        result_lane_rows = [
            {
                "workflow": workflow,
                "lane": lane_name,
                "runs": stats["runs"],
                "classes": dict(stats["classes"]),
                "dimensions": stats["dimensions"],
            }
            for (workflow, lane_name), stats in sorted(result_lanes.items())
        ]
        snapshots.append(
            {
                "date": day,
                "commits": [
                    {"sha": sha, "message": message}
                    for sha, message in sorted(
                        {
                            observation["sha"]: observation.get("commit_message", "")
                            for observation in day_observations
                            if observation.get("sha")
                        }.items()
                    )
                ],
                "totals": {
                    "lane_runs": len(day_observations),
                    "failures": failed_jobs,
                    "reruns": reruns,
                    "rerun_rescues": rescued,
                    "time_to_green": summarized_distribution(time_to_green),
                    "duration": summarized_distribution(all_durations),
                    "phases": {
                        phase: summarized_distribution(values)
                        for phase, values in all_phases.items()
                    },
                },
                "failure_classes": dict(failure_classes),
                "signatures": dict(signatures),
                "lanes": lane_rows,
                "result_lanes": result_lane_rows,
                "tests": [
                    {"id": name, **stats}
                    for name, stats in sorted(tests.items())
                ],
            }
        )
    return snapshots


def load_history(path):
    if not path or not os.path.isfile(path):
        return {"schema_version": HISTORY_SCHEMA_VERSION, "days": []}
    try:
        with open(path, encoding="utf-8") as source:
            history = json.load(source)
    except (OSError, json.JSONDecodeError):
        return {"schema_version": HISTORY_SCHEMA_VERSION, "days": []}
    if history.get("schema_version") != HISTORY_SCHEMA_VERSION:
        return {"schema_version": HISTORY_SCHEMA_VERSION, "days": []}
    return history


def merge_history(history, snapshots, retention_days=400):
    by_day = {
        snapshot["date"]: snapshot
        for snapshot in history.get("days", [])
        if snapshot.get("date")
    }
    by_day.update({snapshot["date"]: snapshot for snapshot in snapshots})
    cutoff = (datetime.now(timezone.utc) - timedelta(days=retention_days)).date().isoformat()
    return {
        "schema_version": HISTORY_SCHEMA_VERSION,
        "generated_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "days": [by_day[day] for day in sorted(by_day) if day >= cutoff],
    }


def window_totals(history, start, end):
    days = [
        day for day in history.get("days", [])
        if start <= day.get("date", "") < end
    ]
    return {
        "days": len(days),
        "lane_runs": sum(day["totals"]["lane_runs"] for day in days),
        "failures": sum(day["totals"]["failures"] for day in days),
        "test_failures": sum(
            day.get("failure_classes", {}).get("test_failure", 0) for day in days
        ),
        "infrastructure_failures": sum(
            day.get("failure_classes", {}).get("infrastructure_failure", 0)
            + day.get("failure_classes", {}).get("unclassified_failure", 0)
            for day in days
        ),
        "reruns": sum(day["totals"].get("reruns", 0) for day in days),
        "rerun_rescues": sum(day["totals"].get("rerun_rescues", 0) for day in days),
    }


def rate(numerator, denominator):
    return 100 * numerator / denominator if denominator else 0


def wilson_interval(successes, total, z=1.96):
    """95% Wilson score interval, returned as percentages."""
    if not total:
        return 0.0, 0.0
    proportion = successes / total
    denominator = 1 + z * z / total
    center = (proportion + z * z / (2 * total)) / denominator
    margin = (
        z
        * math.sqrt(
            proportion * (1 - proportion) / total + z * z / (4 * total * total)
        )
        / denominator
    )
    return 100 * (center - margin), 100 * (center + margin)


def render_trend_summary(history, notes, dashboard_url, pages_enabled=True):
    today = datetime.now(timezone.utc).date()
    current_start = (today - timedelta(days=7)).isoformat()
    previous_start = (today - timedelta(days=14)).isoformat()
    end = (today + timedelta(days=1)).isoformat()
    current = window_totals(history, current_start, end)
    previous = window_totals(history, previous_start, current_start)
    current_rate = rate(current["failures"], current["lane_runs"])
    previous_rate = rate(previous["failures"], previous["lane_runs"])
    confidence_low, confidence_high = wilson_interval(
        current["failures"], current["lane_runs"]
    )
    delta = current_rate - previous_rate
    lines = [
        "# CI Health Report",
        "",
        (
            "_Master-branch trends. Generated "
            f"{datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}._"
        ),
        "",
        (
            f"[Open the interactive 7/14/28/90-day dashboard]({dashboard_url})"
            if pages_enabled
            else (
                "_Dashboard publishing is ready; enable GitHub Pages with "
                "GitHub Actions as its source to make it public._"
            )
        ),
        "",
    ]
    if notes:
        lines.extend(["> **Data completeness:** " + "; ".join(notes), ""])
    lines.extend(
        [
            "## Seven-day comparison",
            "",
            (
                "| Window | Lane runs | Failures | Failure rate | Test | "
                "Infrastructure/unknown | Rerun rescues |"
            ),
            "|---|---:|---:|---:|---:|---:|---:|",
            (
                f"| Latest 7 days | {current['lane_runs']} | {current['failures']} | "
                f"{current_rate:.1f}% | {current['test_failures']} | "
                f"{current['infrastructure_failures']} | "
                f"{current['rerun_rescues']}/{current['reruns']} |"
            ),
            (
                f"| Previous 7 days | {previous['lane_runs']} | {previous['failures']} | "
                f"{previous_rate:.1f}% | {previous['test_failures']} | "
                f"{previous['infrastructure_failures']} | "
                f"{previous['rerun_rescues']}/{previous['reruns']} |"
            ),
            "",
            f"**Failure-rate change:** {delta:+.1f} percentage points.",
            (
                f" Latest-window 95% Wilson interval: "
                f"{confidence_low:.1f}–{confidence_high:.1f}%."
            ),
            "",
        ]
    )

    cutoff = (today - timedelta(days=14)).isoformat()
    lane_totals = collections.defaultdict(lambda: {"runs": 0, "failures": 0, "durations": []})
    test_totals = collections.defaultdict(lambda: {"executions": 0, "failures": 0})
    for day in history.get("days", []):
        if day.get("date", "") < cutoff:
            continue
        for lane in day.get("lanes", []):
            stats = lane_totals[(lane["workflow"], lane["lane"])]
            stats["runs"] += lane["runs"]
            stats["failures"] += lane["failures"]
            if lane.get("duration", {}).get("p95") is not None:
                stats["durations"].append(lane["duration"]["p95"])
        for test in day.get("tests", []):
            stats = test_totals[test["id"]]
            stats["executions"] += test["executions"]
            stats["failures"] += test["failures"]

    failing_lanes = sorted(
        lane_totals.items(),
        key=lambda item: (item[1]["failures"], item[1]["runs"]),
        reverse=True,
    )[:25]
    lines.extend(
        [
            "## Failing lanes (14 days)",
            "",
            "| Workflow | Lane | Runs | Failures | Fail % | Daily p95 min (median) |",
            "|---|---|---:|---:|---:|---:|",
        ]
    )
    for (workflow, lane_name), stats in failing_lanes:
        if not stats["failures"]:
            continue
        daily_p95 = percentile(stats["durations"], 50) or 0
        lines.append(
            f"| {workflow} | {lane_name} | {stats['runs']} | {stats['failures']} | "
            f"{rate(stats['failures'], stats['runs']):.1f}% | {daily_p95:.1f} |"
        )
    lines.extend(["", "## Flakiest tests (true execution denominator)", ""])
    flaky_tests = sorted(
        (
            (name, stats) for name, stats in test_totals.items()
            if stats["failures"]
        ),
        key=lambda item: (
            rate(item[1]["failures"], item[1]["executions"]),
            item[1]["failures"],
        ),
        reverse=True,
    )[:15]
    if flaky_tests:
        lines.extend(["| Test | Executions | Failures | Fail % |", "|---|---:|---:|---:|"])
        for name, stats in flaky_tests:
            lines.append(
                f"| {name} | {stats['executions']} | {stats['failures']} | "
                f"{rate(stats['failures'], stats['executions']):.1f}% |"
            )
    else:
        lines.append(
            "_True per-test rates will populate as successful and failed lanes upload "
            "`ci-result` artifacts._"
        )
    lines.append("")
    return "\n".join(lines)


def write_site(output_dir, history, report, dashboard_source):
    data_dir = os.path.join(output_dir, "data")
    daily_dir = os.path.join(data_dir, "daily")
    os.makedirs(daily_dir, exist_ok=True)
    with open(os.path.join(output_dir, "data", "history.json"), "w", encoding="utf-8") as output:
        json.dump(history, output, separators=(",", ":"), sort_keys=True)
        output.write("\n")
    latest = history["days"][-1] if history.get("days") else {}
    with open(os.path.join(output_dir, "data", "latest.json"), "w", encoding="utf-8") as output:
        json.dump(latest, output, indent=2, sort_keys=True)
        output.write("\n")
    for snapshot in history.get("days", []):
        with open(
            os.path.join(daily_dir, f"{snapshot['date']}.json"),
            "w",
            encoding="utf-8",
        ) as output:
            json.dump(snapshot, output, separators=(",", ":"), sort_keys=True)
            output.write("\n")
    shutil.copyfile(dashboard_source, os.path.join(output_dir, "index.html"))
    with open(os.path.join(output_dir, "report.md"), "w", encoding="utf-8") as output:
        output.write(report)


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
    history_input = os.environ.get("HISTORY_INPUT", "")
    history = load_history(history_input)
    if history.get("days"):
        newest = datetime.strptime(history["days"][-1]["date"], "%Y-%m-%d")
        refresh_days = int(os.environ.get("REFRESH_DAYS", "14"))
        since = (newest - timedelta(days=refresh_days)).strftime("%Y-%m-%d")
    else:
        bootstrap_days = int(os.environ.get("BOOTSTRAP_DAYS", "90"))
        since = (
            datetime.now(timezone.utc) - timedelta(days=bootstrap_days)
        ).strftime("%Y-%m-%d")

    observations, normalized_results, _, rerun_runs, notes = collect_trend_data(
        token, repo, since
    )
    snapshots = aggregate_daily(observations, normalized_results, rerun_runs)
    history = merge_history(history, snapshots)
    dashboard_url = os.environ.get(
        "DASHBOARD_URL", "https://kubeflow.github.io/pipelines/"
    )
    pages_enabled = os.environ.get("PAGES_ENABLED", "true").lower() == "true"
    report = render_trend_summary(
        history, notes, dashboard_url, pages_enabled=pages_enabled
    )
    output_dir = os.environ.get("OUTPUT_DIR", "ci-health-site")
    dashboard_source = os.environ.get(
        "DASHBOARD_SOURCE",
        os.path.join(
            os.path.dirname(__file__), "..", "ci-health-dashboard", "index.html"
        ),
    )
    write_site(output_dir, history, report, dashboard_source)

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
