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
"""Builds the compact, normalized result consumed by CI health reporting."""

import argparse
import collections
import glob
import json
import os
import xml.etree.ElementTree as ET
from datetime import datetime, timezone


def testcase_id(case):
    name = case.get("name") or "<unnamed>"
    classname = case.get("classname") or ""
    return f"{classname} :: {name}" if classname else name


def read_test_results(reports_dir):
    """Returns per-test execution counts, preferring the aggregate junit.xml."""
    aggregate = os.path.join(reports_dir, "junit.xml")
    paths = [aggregate] if os.path.isfile(aggregate) else sorted(
        glob.glob(os.path.join(reports_dir, "*.xml"))
    )
    tests = collections.defaultdict(lambda: {
        "executions": 0,
        "failures": 0,
        "skipped": 0,
    })
    parse_errors = 0
    for path in paths:
        try:
            root = ET.parse(path).getroot()
        except (ET.ParseError, OSError):
            parse_errors += 1
            continue
        for case in root.iter("testcase"):
            stats = tests[testcase_id(case)]
            was_skipped = case.find("skipped") is not None
            stats["executions"] += int(not was_skipped)
            if case.find("failure") is not None or case.find("error") is not None:
                stats["failures"] += 1
            stats["skipped"] += int(was_skipped)
    return [
        {"id": name, **stats}
        for name, stats in sorted(tests.items())
    ], parse_errors


def read_go_test_results(pattern):
    """Parses terminal per-test events from one or more `go test -json` logs."""
    tests = collections.defaultdict(lambda: {
        "executions": 0,
        "failures": 0,
        "skipped": 0,
    })
    parse_errors = 0
    for path in sorted(glob.glob(pattern)) if pattern else []:
        try:
            source = open(path, encoding="utf-8")
        except OSError:
            parse_errors += 1
            continue
        with source:
            for line in source:
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    parse_errors += 1
                    continue
                action, test = event.get("Action"), event.get("Test")
                if not test or action not in ("pass", "fail", "skip"):
                    continue
                package = event.get("Package") or "go"
                stats = tests[f"{package} :: {test}"]
                stats["executions"] += int(action != "skip")
                stats["failures"] += int(action == "fail")
                stats["skipped"] += int(action == "skip")
    return [
        {"id": name, **stats}
        for name, stats in sorted(tests.items())
    ], parse_errors


def merge_test_results(*collections_to_merge):
    merged = collections.defaultdict(lambda: {
        "executions": 0,
        "failures": 0,
        "skipped": 0,
    })
    for test_collection in collections_to_merge:
        for test in test_collection:
            stats = merged[test["id"]]
            for field in ("executions", "failures", "skipped"):
                stats[field] += test[field]
    return [{"id": name, **stats} for name, stats in sorted(merged.items())]


def read_json(path, default):
    if not path or not os.path.isfile(path):
        return default
    try:
        with open(path, encoding="utf-8") as source:
            return json.load(source)
    except (json.JSONDecodeError, OSError):
        return default


def read_epoch(path):
    try:
        with open(path, encoding="utf-8") as source:
            return float(source.read().strip())
    except (OSError, ValueError):
        return None


def elapsed_seconds(start_path, end_path):
    start, end = read_epoch(start_path), read_epoch(end_path)
    if start is None or end is None or end < start:
        return None
    return round(end - start, 3)


def classify(test_outcome, reporting_outcomes, tests, signatures):
    if test_outcome != "success":
        if any(signatures.values()):
            return "infrastructure_failure"
        if any(test["failures"] for test in tests):
            return "test_failure"
        return "unknown_failure"
    if any(outcome not in ("", "success", "skipped") for outcome in reporting_outcomes):
        return "reporting_failure"
    return "success"


def build_result(args):
    junit_tests, junit_errors = read_test_results(args.reports_dir)
    go_tests, go_errors = read_go_test_results(args.go_json)
    tests = merge_test_results(junit_tests, go_tests)
    parse_errors = junit_errors + go_errors
    signatures = read_json(args.signatures, {})
    reporting_outcomes = [
        args.install_outcome,
        args.html_outcome,
        args.upload_outcome,
        args.publish_outcome,
    ]
    result_class = classify(args.test_outcome, reporting_outcomes, tests, signatures)
    now = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
    dimensions = {
        "pipeline_store": args.pipeline_store,
        "proxy": args.proxy,
        "cache_enabled": args.cache_enabled,
        "multi_user": args.multi_user,
        "deployment_mode": args.deployment_mode,
        "mlflow_enabled": args.mlflow_enabled,
        "test_label": args.test_label,
        "parallel_nodes": args.parallel_nodes,
    }
    return {
        "schema_version": 1,
        "generated_at": now,
        "repository": args.repository,
        "workflow": args.workflow,
        "job": args.job,
        "report_name": args.report_name,
        "run_id": int(args.run_id),
        "run_attempt": int(args.run_attempt),
        "sha": args.sha,
        "result": result_class,
        "test_outcome": args.test_outcome,
        "reporting_outcomes": reporting_outcomes,
        "dimensions": dimensions,
        "durations_seconds": {
            "setup": elapsed_seconds(args.action_start, args.test_start),
            "test": elapsed_seconds(args.test_start, args.test_end),
            "report": elapsed_seconds(args.test_end, args.report_end),
        },
        "signatures": signatures,
        "tests": tests,
        "test_parse_errors": parse_errors,
    }


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--reports-dir", required=True)
    parser.add_argument("--go-json", default="")
    parser.add_argument("--signatures", default="")
    parser.add_argument("--action-start", required=True)
    parser.add_argument("--test-start", required=True)
    parser.add_argument("--test-end", required=True)
    parser.add_argument("--report-end", required=True)
    for name in (
        "repository", "workflow", "job", "report-name", "run-id", "run-attempt",
        "sha", "test-outcome", "install-outcome", "html-outcome",
        "upload-outcome", "publish-outcome", "pipeline-store", "proxy",
        "cache-enabled", "multi-user", "deployment-mode", "mlflow-enabled",
        "test-label", "parallel-nodes",
    ):
        parser.add_argument(f"--{name}", default="")
    return parser.parse_args()


def main():
    args = parse_args()
    result = build_result(args)
    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    with open(args.output, "w", encoding="utf-8") as destination:
        json.dump(result, destination, indent=2, sort_keys=True)
        destination.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
