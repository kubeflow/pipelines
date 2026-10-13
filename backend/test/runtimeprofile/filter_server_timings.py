#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Persist correlated timing records without raw server logs."""

import json
import math
from pathlib import Path
import sys
import uuid

MAX_BYTES = 16 * 1024 * 1024
METHODS = {
    'GetRun', 'GetTask', 'CreateTask', 'UpdateTask', 'UpdateTasksBulk',
    'ListTasks'
}
PHASES = {
    'authorization',
    'auth_run_lookup',
    'token_review',
    'subject_access_review',
    'kubernetes_rate_limit_wait',
    'kubernetes_http_roundtrip',
    'task_read',
    'task_create_resource',
    'task_update_resource',
    'task_children',
    'parent_validation',
    'run_read',
}


def timing_record(line, marker):
    if marker not in line:
        return None
    try:
        record = json.loads(line.split(marker, 1)[1].strip())
        value = record['correlation_id']
        if not isinstance(value, str) or str(uuid.UUID(value)) != value:
            return None
        return record
    except (ValueError, TypeError, KeyError, AttributeError):
        return None


def milliseconds(value):
    return (isinstance(value, (int, float)) and not isinstance(value, bool) and
            math.isfinite(value) and value >= 0)


def collect(root, source):
    selected = {}
    truncated_clients = 0
    for path in sorted(root.glob('*/*.log')):
        with path.open('rb') as stream:
            data = stream.read(MAX_BYTES + 1)
        if len(data) >= MAX_BYTES:
            truncated_clients += 1
        for line in data[:MAX_BYTES].decode(
                'utf-8', errors='replace').splitlines():
            record = timing_record(line, 'KFP_RUNTIME_CLIENT_TIMING ')
            if record is None:
                continue
            method = record.get('method', '')
            if not isinstance(method, str):
                continue
            method = method.rsplit('/', 1)[-1]
            if method in METHODS:
                selected[record['correlation_id']] = method
    raw = source.read(MAX_BYTES + 1)
    records = []
    matched = set()
    rejected = 0
    for line in raw[:MAX_BYTES].decode('utf-8', errors='replace').splitlines():
        if 'KFP_RUNTIME_SERVER_TIMING ' not in line:
            continue
        record = timing_record(line, 'KFP_RUNTIME_SERVER_TIMING ')
        if record is None:
            rejected += 1
            continue
        correlation_id = record['correlation_id']
        if correlation_id not in selected:
            continue
        phases = record.get('phases_ms')
        if (record.get('method') != selected[correlation_id] or
                not milliseconds(record.get('total_ms')) or
                not isinstance(phases, dict) or not all(
                    key in PHASES and milliseconds(value)
                    for key, value in phases.items())):
            rejected += 1
            continue
        # Copy an explicit schema instead of serializing untrusted extra fields.
        records.append({
            'correlation_id': correlation_id,
            'method': record['method'],
            'total_ms': record['total_ms'],
            'phases_ms': phases,
        })
        matched.add(correlation_id)
    missing = sorted(set(selected) - matched)
    # kubectl can cap output at exactly the requested limit; treat that boundary
    # conservatively rather than calling an incomplete capture complete.
    server_truncated = len(raw) >= MAX_BYTES
    incomplete_reasons = []
    if not selected:
        incomplete_reasons.append('no_selected_client_records')
    if not matched:
        incomplete_reasons.append('no_matching_server_records')
    if missing:
        incomplete_reasons.append('missing_server_records')
    if server_truncated:
        incomplete_reasons.append('server_input_at_byte_limit')
    if truncated_clients:
        incomplete_reasons.append('client_logs_at_byte_limit')
    if rejected:
        incomplete_reasons.append('invalid_server_records')
    summary = {
        'complete': not incomplete_reasons,
        'incomplete_reasons': incomplete_reasons,
        'selected_client_ids': len(selected),
        'matched_client_ids': len(matched),
        'matched_server_records': len(records),
        'missing_server_ids': missing,
        'rejected_records': rejected,
        'server_input_truncated': server_truncated,
        'client_logs_truncated': truncated_clients,
        'phase_durations_are_inclusive': True,
    }
    (root / 'server-timings.jsonl').write_text(''.join(
        json.dumps(record, sort_keys=True) + '\n' for record in records))
    (root / 'server-timings-summary.json'
    ).write_text(json.dumps(summary, indent=2) + '\n')
    return summary


def main():
    if len(sys.argv) != 2 or not Path(sys.argv[1]).is_dir():
        raise SystemExit(
            'Usage: filter_server_timings.py EXISTING_PROFILE_DIRECTORY')
    collect(Path(sys.argv[1]), sys.stdin.buffer)


if __name__ == '__main__':
    main()
