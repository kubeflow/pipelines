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
"""Validate and summarize bounded source observation without disclosing
callers."""

from collections import Counter
from datetime import datetime
from datetime import timezone
import json

from workload_assessment import finding

SOURCE_REVISION = '2511cdbd74cd531c6633f2e094d235082a32917d'
OPERATIONS = ('upload_pipeline', 'upload_pipeline_version', 'create_run',
              'create_recurring_run', 'retry_run', 'read_run_log',
              'read_artifact', 'authorization')
HEALTH = ('write_failures', 'dropped_queue', 'dropped_limit', 'invalid_fields')
UNSUPPORTED = ('frontend_operations', 'artifact_destinations', 'caller_groups',
               'authenticated_identity_on_bypasses', 'target_policy_evaluation',
               'request_authorization_correlation', 'workload_execution',
               'operations_before_or_after_interval')


def _time(value):
    if not isinstance(value, str):
        raise ValueError('Observation timestamps must include a timezone.')
    timestamp = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if timestamp.tzinfo is None:
        raise ValueError('Observation timestamps must include a timezone.')
    return timestamp


def _counter(value):
    return type(value) is int and 0 <= value < 2**64


def assess(data, source_version, now=None):
    if (not isinstance(data, dict) or
            data.get('schema_version') != 'kfp-source-observation/v1' or
            data.get('source_revision') != SOURCE_REVISION or
            data.get('source_version') != '2.17.2' or
            source_version != '2.17.2' or
            len(json.dumps(data).encode()) > 1048576):
        raise ValueError(
            'Unsupported or oversized source observation evidence.')
    limits = data.get('limits', {})
    duration = limits.get('duration_seconds')
    if (type(duration) is not int or not 1 <= duration <= 604800 or
            limits.get('max_records') != 512 or
            limits.get('max_bytes') != 1048576):
        raise ValueError('Invalid observation limits.')
    started, planned, checkpoint = (
        _time(data.get(key))
        for key in ('started_at', 'planned_end_at', 'checkpoint_at'))
    current = now or datetime.now(timezone.utc)
    if (planned <= started or
            abs((planned - started).total_seconds() - duration) > 1 or
            checkpoint < started or checkpoint > current):
        raise ValueError('Invalid observation interval.')
    status = data.get('status')
    if status not in ('active', 'completed', 'interrupted'):
        raise ValueError('Invalid observation state.')
    if status != 'active':
        ended = _time(data.get('ended_at'))
        if ended != checkpoint or (status == 'completed' and ended < planned):
            raise ValueError('Invalid observation completion evidence.')
    elif data.get('ended_at'):
        raise ValueError('Active observation cannot have an end time.')
    counts, health = data.get('operations'), data.get('health')
    if (not isinstance(counts, dict) or set(counts) != set(OPERATIONS) or
            not all(_counter(v) for v in counts.values()) or
            not isinstance(health, dict) or set(health) != set(HEALTH) or
            not all(_counter(v) for v in health.values())):
        raise ValueError('Invalid observation counters.')
    for key, expected in (('supported_operations', OPERATIONS),
                          ('unsupported_checks', UNSUPPORTED)):
        values = data.get(key)
        if not isinstance(values, list) or not all(
                isinstance(v, str) for v in values) or set(values) != set(
                    expected) or len(values) != len(expected):
            raise ValueError('Unsupported observation coverage claim.')
    unobserved = [
        operation for operation in OPERATIONS if counts[operation] == 0
    ]
    if sorted(data.get('unobserved_operations', [])) != sorted(unobserved):
        raise ValueError('Inconsistent unobserved operations.')
    records = data.get('records')
    if not isinstance(records, list) or len(records) > 512:
        raise ValueError('Invalid observation records.')
    retained, callers, omitted = Counter(), set(), 0
    for row in records:
        if (not isinstance(row, dict) or
                row.get('operation') not in OPERATIONS or set(row) - {
                    'observed_at', 'operation', 'api_version', 'namespace',
                    'namespace_omitted', 'caller', 'resource', 'verb', 'result'
                }):
            raise ValueError('Invalid observation record.')
        if not started <= _time(row.get('observed_at')) <= min(
                planned, checkpoint):
            raise ValueError('Observation record outside interval.')
        for key in ('api_version', 'namespace', 'caller', 'resource', 'verb',
                    'result'):
            value = row.get(key, '')
            if not isinstance(value, str) or len(value) > 256 or any(
                    ord(c) < 32 for c in value):
                raise ValueError('Invalid observation field.')
        if 'namespace_omitted' in row and type(
                row['namespace_omitted']) is not bool:
            raise ValueError('Invalid namespace observation.')
        retained[row['operation']] += 1
        if row['operation'] == 'authorization' and row.get('caller'):
            callers.add(row['caller'])
        if row['operation'] == 'upload_pipeline' and row.get(
                'namespace_omitted'):
            omitted += 1
    if any(retained[key] > counts[key] for key in retained):
        raise ValueError('Observation records exceed counters.')
    summary = dict(
        source_revision=SOURCE_REVISION,
        source_version='2.17.2',
        started_at=started.isoformat(),
        planned_end_at=planned.isoformat(),
        checkpoint_at=checkpoint.isoformat(),
        status=status,
        checkpoint_age_seconds=int((current - checkpoint).total_seconds()),
        health=dict(health),
        operations=dict(counts),
        retained_records=len(records),
        authenticated_callers_observed=len(callers),
        omitted_namespace_uploads_retained=omitted,
        unobserved_operations=unobserved,
        unsupported_checks=list(UNSUPPORTED),
        evidence='operator_supplied_not_verified')
    evidence = (
        'Observation status: ' + status + '; retained records: ' +
        str(len(records)) + '; health counters: ' + str(sum(health.values())) +
        '. Interval and observer health do not prove target policy or request success; caller values are withheld.'
    )
    findings = [
        finding(
            'observation.coverage', 'unknown', 'installation', evidence,
            'Collect a representative healthy interval and retain unsupported checks; investigate dropped or stale evidence.',
            'Compare source operation coverage with intended clients and infrequent schedules; quiet periods are not a pass.'
        )
    ]
    if omitted:
        findings.append(
            finding(
                'observation.uploadNamespace', 'operational_impact',
                'installation',
                str(omitted) +
                ' retained new-pipeline upload requests omitted namespace. Request outcome and authenticated caller cannot be correlated from these events.',
                'Set the upload method namespace explicitly for private uploads; verify shared writers in the installation namespace.',
                'Check target pipelines/create with the intended caller and upload namespace, then exercise the same SDK call.'
            ))
    for operation in unobserved:
        findings.append(
            finding(
                'observation.unobserved', 'unknown', operation,
                'No retained operation count during the reported source observation interval.',
                'Choose a representative interval or document why this operation is not used.',
                'Exercise the intended client separately in an isolated candidate test.'
            ))
    return findings, summary
