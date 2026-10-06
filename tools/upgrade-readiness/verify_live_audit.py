# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify audit emission in the exclusive disposable schedule fixture.

The producer has no run or schedule ID. Evidence is limited to the
isolated namespace, denied account and activation window; it cannot
correlate each run. Raw API logs stay bounded in memory and are never
printed or saved.
"""

import argparse
from datetime import datetime
from datetime import timezone
import json
import re
import selectors
import subprocess
import time

from kfp_http import CollectionError
from kubectl_inventory import kill_process_group
from live_schedule_check import load
from live_schedule_check import timestamp

CONTEXT = 'kind-kfp-readiness'
NAMESPACE = 'kfp-readiness-test'
ACCOUNT = 'readiness-denied'
# Ten-minute fixture observations include normal API request logs as well as audit records.
MAX_BYTES = 16 * 1024 * 1024
TIMEOUT = 30
SCOPE = 'isolated_namespace_account_activation_window'
DIAGNOSTIC_REASONS = frozenset({
    'recent_isolated_audit_activation_required',
    'isolated_test_context_required',
    'successful_audit_fixture_required',
    'complete_audit_fixture_required',
    'audit_fixture_identity_mismatch',
    'audit_collection_timed_out',
    'audit_collection_exceeded_limit',
    'audit_collection_failed',
    'audit_collection_truncated',
    'audit_collection_invalid_encoding',
    'audit_record_invalid_timestamp',
    'invalid_timestamp',
})
SCENARIOS = {
    'default': 'pipeline-runner',
    'scoped': 'readiness-granted',
    'denied': ACCOUNT
}
MESSAGE = ('security_audit control=service_account mode=audit '
           'operation=authorize_service_account reason=account_denied '
           'namespace="' + NAMESPACE + '" service_account="' + ACCOUNT + '" '
           'disposition=allow_policy_violation')
LINE = re.compile(r'^(\S+)\s+W\d{4}\s+\d{2}:\d{2}:\d{2}\.\d+\s+\d+\s+'
                  r'resource_manager\.go:\d+\]\s+' + re.escape(MESSAGE) +
                  r'\s*$')


def validate_completion(report, start):
    if (report.get('scope') != 'fixture_run_completion' or
            report.get('mode') != 'audit' or
            report.get('outcome') != 'passed' or
            report.get('all_expected_runs_succeeded') is not True or
            report.get('namespace') != NAMESPACE or
            timestamp(report.get('observation_start')) != start):
        raise ValueError('successful_audit_fixture_required')
    cases = report.get('cases')
    if not isinstance(cases, list) or len(cases) != len(SCENARIOS):
        raise ValueError('complete_audit_fixture_required')
    seen = set()
    for case in cases:
        scenario = case.get('scenario')
        if scenario not in SCENARIOS or scenario in seen or case.get(
                'service_account') != SCENARIOS[scenario]:
            raise ValueError('audit_fixture_identity_mismatch')
        seen.add(scenario)
        records = case.get('runs')
        if not isinstance(records, list) or not records or len(
                records) > 10000 or not all(
                    isinstance(run, dict) and run.get('state') == 'SUCCEEDED'
                    for run in records):
            raise ValueError('successful_audit_fixture_required')


def collect_logs(context, start, progress=None):
    if context != CONTEXT:
        raise ValueError('isolated_test_context_required')
    command = [
        'kubectl', '--context', CONTEXT, '--request-timeout=20s', '--namespace',
        'kubeflow', 'logs', '--selector=app=ml-pipeline',
        '--container=ml-pipeline-api-server', '--timestamps=true', '--tail=-1',
        '--ignore-errors=false', '--since-time=' + start.isoformat(),
        '--limit-bytes=' + str(MAX_BYTES + 1)
    ]
    chunks = bytearray()
    if progress is not None:
        progress['collected_bytes'] = 0
    try:
        with subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                start_new_session=True) as process:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                deadline = time.monotonic() + TIMEOUT
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        kill_process_group(process)
                        raise CollectionError('audit_collection_timed_out')
                    chunk = process.stdout.read1(
                        min(65536, MAX_BYTES + 1 - len(chunks)))
                    if not chunk:
                        break
                    chunks.extend(chunk)
                    if progress is not None:
                        progress['collected_bytes'] = len(chunks)
                    if len(chunks) > MAX_BYTES:
                        kill_process_group(process)
                        raise CollectionError('audit_collection_exceeded_limit')
                try:
                    code = process.wait(
                        timeout=max(0.01, deadline - time.monotonic()))
                except subprocess.TimeoutExpired:
                    kill_process_group(process)
                    raise CollectionError(
                        'audit_collection_timed_out') from None
                if progress is not None:
                    progress['collector_exit_code'] = code
                if code:
                    raise CollectionError('audit_collection_failed')
    except OSError:
        raise CollectionError('audit_collection_failed') from None
    if chunks and not chunks.endswith(b'\n'):
        raise CollectionError('audit_collection_truncated')
    try:
        return chunks.decode('utf-8')
    except UnicodeError:
        raise CollectionError('audit_collection_invalid_encoding') from None


def count_records(logs, start):
    count = 0
    end = datetime.now(timezone.utc)
    for line in logs.splitlines():
        match = LINE.fullmatch(line)
        if match:
            try:
                if start <= timestamp(match.group(1)) <= end:
                    count += 1
            except ValueError:
                raise CollectionError(
                    'audit_record_invalid_timestamp') from None
    return count


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--context', required=True)
    parser.add_argument('--not-before', required=True)
    parser.add_argument('--completion-report', required=True)
    args = parser.parse_args(argv)
    report = dict(
        outcome='inconclusive',
        scope=SCOPE,
        namespace=NAMESPACE,
        service_account=ACCOUNT)
    began = time.monotonic()
    progress = dict(stage='activation_validation')
    try:
        start = timestamp(args.not_before)
        age = (datetime.now(timezone.utc) - start).total_seconds()
        if args.context != CONTEXT or not 0 <= age <= 1800:
            raise ValueError('recent_isolated_audit_activation_required')
        progress['stage'] = 'completion_validation'
        validate_completion(load(args.completion_report), start)
        progress['stage'] = 'log_collection'
        logs = collect_logs(args.context, start, progress=progress)
        progress['stage'] = 'audit_record_validation'
        count = count_records(logs, start)
        report.update(
            outcome='passed' if count else 'inconclusive',
            matching_records=count,
            observation_start=start.isoformat())
        if not count:
            report['reason'] = 'matching_audit_record_not_observed'
    except (OSError, ValueError, TypeError, KeyError, AttributeError) as error:
        report['reason'] = (
            str(error) if str(error) in DIAGNOSTIC_REASONS else
            'invalid_or_incomplete_audit_evidence')
        report['diagnostics'] = dict(
            progress, elapsed_seconds=round(time.monotonic() - began, 3))
    print(json.dumps(report, sort_keys=True))
    return 0 if report['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
