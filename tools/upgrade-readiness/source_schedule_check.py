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
"""Verify source 2.17.2 run identity using correlated live Workflows."""

import argparse
from pathlib import Path
import time

from fixture_diagnostics import container_diagnostics
from fixture_diagnostics import node_diagnostics
from kfp_http import Client
from kfp_http import CollectionError
from live_schedule_check import field
from live_schedule_check import list_runs
from live_schedule_check import RUN_STATES
from live_schedule_check import timestamp
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
from provision_live_schedules import write_object
from readiness import kubectl_get


def source_run_evidence(client, namespace, case, start, get=kubectl_get):
    """2.17.2 ReportWorkflow omits API service_account; verify live identity."""
    # Workflow creation precedes API persistence; read runs before its snapshot.
    records = list_runs(client, namespace, case['schedule_uid'])
    data, error = get(CONTEXT, namespace, 'workflows.argoproj.io')
    if error or not isinstance(data, dict) or not isinstance(
            data.get('items'), list):
        raise CollectionError('source_workflow_collection_failed')
    workflows = data['items']
    if len(workflows) > 1000:
        raise CollectionError('source_workflow_limit')
    fresh = []
    for run in records:
        uid = field(run, 'run_id', 'runId')
        if uid in case['baseline_run_ids'] or timestamp(
                field(run, 'created_at', 'createdAt')) < start:
            continue
        matches = [
            w for w in workflows
            if w.get('metadata', {}).get('namespace') == namespace and
            w.get('metadata', {}).get('name') == field(run, 'display_name',
                                                       'displayName') and
            w.get('metadata', {}).get('labels', {}).get('pipeline/runid') == uid
        ]
        if len(matches) != 1:
            raise CollectionError('source_workflow_identity_unavailable')
        workflow = matches[0]
        metadata = workflow['metadata']
        owners = metadata.get('ownerReferences', [])
        if (not metadata.get('uid') or
                timestamp(metadata.get('creationTimestamp')) < start or not any(
                    o.get('uid') == case['schedule_uid'] and
                    o.get('name') == case['schedule_name'] and o.get('kind') ==
                    'ScheduledWorkflow' and o.get('controller') is True
                    for o in owners)):
            raise CollectionError('source_workflow_owner_mismatch')
        account = field(run, 'service_account', 'serviceAccount')
        if (account not in (None, '', case['service_account']) or
                workflow.get('spec', {}).get('serviceAccountName')
                != case['service_account']):
            raise CollectionError('source_workflow_account_mismatch')
        state = run.get('state')
        fresh.append(
            dict(
                run_id=uid,
                workflow_uid=metadata['uid'],
                state=state if isinstance(state, str) and state in RUN_STATES
                else 'UNKNOWN'))
    return fresh


def diagnostics(get=kubectl_get):
    """Persist counts and enumerated states only; never specs, messages, or
    logs."""
    result = {}
    objects = {}
    for resource, phases in (('workflows.argoproj.io',
                              ('Pending', 'Running', 'Succeeded', 'Failed',
                               'Error')), ('pods',
                                           ('Pending', 'Running', 'Succeeded',
                                            'Failed', 'Unknown'))):
        data, error = get(CONTEXT, NAMESPACE, resource)
        if error or not isinstance(data, dict) or not isinstance(
                data.get('items'), list):
            result[resource] = dict(collection='unavailable')
            continue
        items = data['items']
        if len(items) > 1000:
            result[resource] = dict(collection='limit_exceeded')
            continue
        objects[resource] = items
        counts = {phase: 0 for phase in phases}
        counts['other'] = 0
        for item in items:
            phase = item.get('status', {}).get('phase')
            counts[phase if phase in phases else 'other'] += 1
        result[resource] = dict(
            collection='complete', count=len(items), phases=counts)
    if 'pods' in objects and 'workflows.argoproj.io' in objects:
        result['workflow_nodes'] = node_diagnostics(
            objects['workflows.argoproj.io'])
        result['container_failures'] = container_diagnostics(
            objects['pods'], objects['workflows.argoproj.io'])
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('state-dir', 'endpoint', 'token-file', 'output'):
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    state = Path(args.state_dir)
    report = dict(
        source_version='2.17.2',
        scope='source_run_creation_with_workflow_identity',
        outcome='inconclusive',
        observed_scenarios=[])
    try:
        fixture = read_object(state / 'state.json')
        if fixture.get('context') != CONTEXT or fixture.get(
                'namespace') != NAMESPACE:
            raise ValueError('invalid_fixture_scope')
        start = timestamp(fixture['activation_start'])
        cases = [
            dict(case, baseline_run_ids=[]) for case in fixture['schedules']
        ]
        deadline = time.monotonic() + 180
        while time.monotonic() < deadline:
            client = Client(args.endpoint, args.token_file)
            seen = []
            for case in cases:
                if source_run_evidence(client, NAMESPACE, case, start):
                    seen.append(case['scenario'])
            report['observed_scenarios'] = sorted(seen)
            if len(seen) == 3:
                report['outcome'] = 'passed'
                break
            time.sleep(5)
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        report['reason'] = 'source_evidence_collection_failed'
    if report['outcome'] != 'passed':
        report['diagnostics'] = diagnostics()
    write_object(Path(args.output), report)
    return 0 if report['outcome'] == 'passed' else 1


if __name__ == '__main__':
    raise SystemExit(main())
