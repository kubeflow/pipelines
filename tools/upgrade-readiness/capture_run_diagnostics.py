# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Bounded, read-only diagnostics for disappearing live-fixture run evidence.

This report cannot establish acceptance and never substitutes for the
drain. Only allowlisted metadata is retained, never raw API bodies or
exception text.
"""
import argparse
from collections import Counter
from datetime import datetime
from datetime import timezone
import json
from pathlib import Path
import re
import time

from kfp_http import Client
from kfp_inventory import field
from kfp_inventory import identifier
from live_schedule_check import DIAGNOSTIC_REASONS
from live_schedule_check import list_runs
from live_schedule_check import RUN_STATES
from live_schedule_check import timestamp
from live_schedule_check import validate_baseline
from verify_live_audit import collect_logs

MAX_INPUT = 4 * 1024 * 1024
MAX_CASES = 3
MAX_LIST_RECORDS = 100
MAX_DIRECT_RECORDS = 20
DIAGNOSTIC_SECONDS = 90


def reason(error):
    value = str(error)
    return value if value in DIAGNOSTIC_REASONS | {
        'diagnostic_deadline_exceeded'
    } else 'invalid_or_incomplete_evidence'


def load(path):
    with Path(path).open('rb') as stream:
        raw = stream.read(MAX_INPUT + 1)
    if len(raw) > MAX_INPUT:
        raise ValueError('input_limit_exceeded')
    return json.loads(raw)


class RecordingClient:

    def __init__(self, client, deadline):
        self.client = client
        self.deadline = deadline
        self.pages = []

    def get(self, path, params=None):
        if time.monotonic() >= self.deadline:
            raise ValueError('diagnostic_deadline_exceeded')
        response = self.client.get(path, params)
        if path == '/apis/v2beta1/runs':
            records = response.get('runs', [])
            total = field(response, 'total_size', 'totalSize')
            self.pages.append(
                dict(
                    returned_count=len(records)
                    if isinstance(records, list) else None,
                    total_size=total if type(total) is int and
                    -1 <= total <= 1000000000 else None,
                    has_next_page=bool(
                        field(response, 'next_page_token', 'nextPageToken'))))
        return response


def project(run, case, start):
    uid = identifier(field(run, 'run_id', 'runId'))
    recurring = identifier(field(run, 'recurring_run_id', 'recurringRunId'))
    experiment = identifier(field(run, 'experiment_id', 'experimentId'))
    created = timestamp(field(run, 'created_at', 'createdAt'))
    account = field(run, 'service_account', 'serviceAccount')
    if account not in ('', None):
        account = identifier(account)
    state = run.get('state')
    excluded = []
    if uid in case['baseline_run_ids']:
        excluded.append('baseline_run')
    if created < start:
        excluded.append('before_activation')
    return dict(
        run_id=uid,
        recurring_run_id=recurring,
        experiment_id=experiment,
        service_account=account,
        created_at=created.isoformat(),
        state=state
        if isinstance(state, str) and state in RUN_STATES else 'UNKNOWN',
        schedule_matches=recurring == case['schedule_uid'],
        account_matches=account == case['service_account'],
        excluded_reasons=excluded)


def collect(client_factory, namespace, cases, observed, start, completion=None):
    if len(cases) > MAX_CASES:
        raise ValueError('case_limit')
    if not isinstance(observed, dict) or not isinstance(
            observed.get('cases'), list):
        raise ValueError('invalid_observation')
    result = dict(
        scope='run_evidence_diagnostics_only',
        outcome='collected',
        collected_at=datetime.now(timezone.utc).isoformat(),
        observation_start=start.isoformat(),
        namespace=namespace,
        cases=[])
    deadline = time.monotonic() + DIAGNOSTIC_SECONDS
    for case in cases:
        output = dict(
            scenario=case['scenario'], schedule_uid=case['schedule_uid'])
        result['cases'].append(output)
        client = RecordingClient(client_factory(), deadline)
        try:
            records = list_runs(client, namespace, case['schedule_uid'])
            projected = [project(run, case, start) for run in records]
            output['list'] = dict(
                collection='complete',
                pages=client.pages,
                returned_count=len(records),
                fresh_count=sum(not r['excluded_reasons'] for r in projected),
                excluded_baseline_count=sum(
                    'baseline_run' in r['excluded_reasons'] for r in projected),
                excluded_before_activation_count=sum(
                    'before_activation' in r['excluded_reasons']
                    for r in projected),
                records=projected[:MAX_LIST_RECORDS],
                truncated=len(projected) > MAX_LIST_RECORDS,
                namespace_verified=True)
        except (ValueError, TypeError, KeyError, AttributeError,
                OSError) as error:
            output['list'] = dict(
                collection='inconclusive',
                reason=reason(error),
                pages=client.pages)
        matches = [
            c for c in observed['cases']
            if c.get('schedule_uid') == case['schedule_uid']
        ]
        if len(matches) != 1 or not isinstance(matches[0].get('runs'), list):
            output['direct'] = dict(
                collection='inconclusive',
                reason='invalid_observation_identity')
            continue
        ids = {identifier(r['run_id']) for r in matches[0]['runs']}
        if completion is not None:
            known = completion.get('known_run_ids',
                                   {}).get(case['schedule_uid'], [])
            if not isinstance(known, list) or len(known) > 10000:
                raise ValueError('invalid_completion_ids')
            ids.update(identifier(uid) for uid in known)
        ids = sorted(ids)
        direct = dict(
            records=[],
            truncated=len(ids) > MAX_DIRECT_RECORDS,
            known_id_count=len(ids))
        output['direct'] = direct
        for uid in ids[:MAX_DIRECT_RECORDS]:
            item = dict(requested_run_id=uid)
            direct['records'].append(item)
            try:
                run = client.get('/apis/v2beta1/runs/' + uid)
                metadata = project(run, case, start)
                if metadata['run_id'] != uid:
                    raise ValueError('direct_run_identity_mismatch')
                experiment_id = metadata['experiment_id']
                experiment = client.get('/apis/v2beta1/experiments/' +
                                        experiment_id)
                proof = (
                    field(experiment, 'experiment_id',
                          'experimentId') == experiment_id and
                    experiment.get('namespace') == namespace)
                item.update(
                    collection='complete',
                    metadata=metadata,
                    namespace_verified=proof)
            except (ValueError, TypeError, KeyError, AttributeError,
                    OSError) as error:
                item.update(collection='inconclusive', reason=reason(error))
    return result


def database_diagnostics(context, start):
    """Keep only numeric MySQL codes from failed ListRuns log records."""
    try:
        logs = collect_logs(context, start)
        codes = Counter(
            code for line in logs.splitlines()
            if '/ListRuns call failed' in line
            for code in re.findall(r'Error ([0-9]{4}) \([0-9A-Z]{5}\)', line))
        return dict(collection='complete', mysql_error_code_counts=dict(codes))
    except (OSError, ValueError):
        return dict(
            collection='inconclusive', reason='api_log_collection_failed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', required=True)
    parser.add_argument('--context')
    parser.add_argument('--baseline', required=True)
    parser.add_argument('--observed', required=True)
    parser.add_argument('--completion')
    parser.add_argument('--activation-start-file', required=True)
    parser.add_argument('--endpoint', required=True)
    parser.add_argument('--token-file', required=True)
    args = parser.parse_args()
    start = None
    try:
        cases, baseline_start = validate_baseline(
            load(args.baseline), args.namespace)
        with open(args.activation_start_file, encoding='utf-8') as stream:
            start = timestamp(stream.read(128).strip())
        if start < baseline_start:
            raise ValueError('invalid_activation_start')
        report = collect(lambda: Client(args.endpoint, args.token_file),
                         args.namespace, cases, load(args.observed), start,
                         load(args.completion) if args.completion else None)
    except (ValueError, TypeError, KeyError, AttributeError, OSError) as error:
        report = dict(
            scope='run_evidence_diagnostics_only',
            outcome='inconclusive',
            reason=reason(error))
    if args.context and start is not None:
        report['database_diagnostics'] = database_diagnostics(
            args.context, start)
    print(json.dumps(report, sort_keys=True))


if __name__ == '__main__':
    main()
