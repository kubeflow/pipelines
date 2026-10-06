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
"""Create raw-Argo V1 jobs in an already owned disposable CI namespace.

Job creation and activation use the V1 API. Observation intentionally uses
its V2 run read view: toApiRun converts stored V1 states while preserving
run, recurring-run, experiment and service-account identity.
"""

import argparse
import json
from pathlib import Path
import re

from fixture_http import FixtureClient
from fixture_http import FixtureError
from kfp_http import Client
import live_schedule_check as live
import provision_live_schedules as fixture
from source_schedule_check import source_run_evidence


def reference(kind, uid):
    return dict(key=dict(type=kind, id=uid), relationship='OWNER')


def workflow():
    # A real V1 workflow, with no V2 IR, launcher or package installation.
    return dict(
        apiVersion='argoproj.io/v1alpha1',
        kind='Workflow',
        metadata=dict(generateName='readiness-v1-'),
        spec=dict(
            entrypoint='hello',
            templates=[
                dict(
                    name='hello',
                    container=dict(
                        image='busybox:1.36',
                        command=['sh', '-c'],
                        args=['echo readiness-v1']))
            ]))


def prepare(context, state_dir, parent, client):
    fixture.verify_state(context, parent)
    if not parent.get('rbac_ready') or parent.get('enabled') is not False:
        raise FixtureError('disabled_owned_fixture_required')
    path = state_dir / 'state.json'
    if path.exists():
        raise FixtureError('v1_fixture_already_started')
    state = dict(
        context=context,
        namespace=fixture.NAMESPACE,
        owner_marker=parent['owner_marker'],
        rbac_ready=True,
        api_version='v1beta1',
        workload_format='argo_workflow',
        enabled=False,
        schedules=[])
    fixture.write_object(path, state)
    experiment = client.post(
        '/apis/v1beta1/experiments',
        dict(
            name='readiness-v1-' + state['owner_marker'],
            resource_references=[reference('NAMESPACE', fixture.NAMESPACE)]))
    state['experiment_id'] = fixture.api_identifier(experiment.get('id'))
    fixture.write_object(path, state)
    for scenario, account in zip(('default', 'scoped', 'denied'),
                                 fixture.ACCOUNTS):
        payload = dict(
            name='readiness-v1-' + scenario,
            pipeline_spec=dict(workflow_manifest=json.dumps(workflow())),
            resource_references=[
                reference('EXPERIMENT', state['experiment_id']),
                reference('NAMESPACE', fixture.NAMESPACE)
            ],
            max_concurrency='1',
            enabled=False,
            no_catchup=True,
            trigger=dict(periodic_schedule=dict(interval_second='30')))
        if scenario != 'default':
            payload['service_account'] = account
        response = client.post('/apis/v1beta1/jobs', payload)
        record = dict(
            scenario=scenario,
            service_account=account,
            schedule_uid=fixture.api_identifier(response.get('id')))
        state['schedules'].append(record)
        # Persist disabled IDs before resolving Kubernetes identities so cleanup
        # still works when a later request or identity check fails.
        fixture.write_object(path, state)
        record['schedule_name'] = fixture.schedule_identity(
            context, record['schedule_uid'])
        fixture.write_object(path, state)
    state['prepared'] = True
    fixture.write_object(path, state)
    cases = [
        dict(
            record,
            expected_outcome='blocked'
            if record['scenario'] == 'denied' else 'run_created',
            expected_prediction='policy_rejection'
            if record['scenario'] == 'denied' else 'no_issue_detected')
        for record in state['schedules']
    ]
    fixture.write_object(
        state_dir / 'cases.json',
        dict(
            namespace=fixture.NAMESPACE,
            scope='v1_runtime_acceptance',
            cases=cases))
    # This is the acceptance contract, not output of the readiness scanner.
    fixture.write_object(
        state_dir / 'expectations.json',
        dict(
            scope='v1_runtime_expectations',
            pre_upgrade_prediction_validated=False,
            findings=[
                dict(
                    rule='schedule.targetMainAccount',
                    resource='ScheduledWorkflow/' + fixture.NAMESPACE + '/' +
                    case['schedule_name'],
                    status=case['expected_prediction']) for case in cases
            ]))


class ActivationClient:
    """Reuse the tested all-ID rollback behavior with V1 activation paths."""

    def __init__(self, client):
        self.client = client

    def post(self, path, body):
        match = re.fullmatch(
            r'/apis/v2beta1/recurringruns/([a-zA-Z0-9_.-]+):(enable|disable)',
            path)
        if not match:
            raise FixtureError('invalid_v1_activation_path')
        return self.client.post(
            '/apis/v1beta1/jobs/' + match[1] + '/' + match[2], body)


def verify_execution(state, client):
    """Require strict API identity plus owned live Workflows after drain."""
    if state.get('enabled') is not False or not state.get('prepared'):
        raise FixtureError('disabled_v1_fixture_required')
    start = live.timestamp(state['activation_start'])
    records = []
    for definition in state['schedules']:
        case = dict(definition, baseline_run_ids=[])
        strict = live.run_evidence(client, fixture.NAMESPACE, case, start)
        workflows = source_run_evidence(client, fixture.NAMESPACE, case, start)
        if {r['run_id'] for r in strict} != {r['run_id'] for r in workflows}:
            raise FixtureError('v1_workflow_run_set_mismatch')
        if case['scenario'] == 'denied':
            if strict:
                raise FixtureError('denied_v1_run_created')
        elif not strict or any(r['state'] != 'SUCCEEDED' for r in strict):
            raise FixtureError('v1_run_not_successful')
        records.append(dict(scenario=case['scenario'], runs=workflows))
    return dict(
        scope='v1_runtime_workflow_identity',
        api_version='v1beta1',
        pre_upgrade_prediction_validated=False,
        outcome='passed',
        cases=records)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('context', 'state-dir', 'endpoint', 'token-file'):
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--parent-state')
    parser.add_argument('--allow-test-cluster-mutations', action='store_true')
    parser.add_argument(
        '--phase',
        choices=('prepare', 'enable', 'disable', 'verify'),
        required=True)
    args = parser.parse_args()
    try:
        if args.context != fixture.CONTEXT or not args.allow_test_cluster_mutations:
            raise FixtureError('explicit_isolated_cluster_consent_required')
        state_dir = Path(args.state_dir)
        state_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
        client = FixtureClient(args.endpoint, args.token_file)
        if args.phase == 'prepare':
            prepare(args.context, state_dir,
                    fixture.read_object(args.parent_state), client)
        else:
            state = fixture.read_object(state_dir / 'state.json')
            fixture.verify_state(args.context, state)
            if state.get('api_version') != 'v1beta1':
                raise FixtureError('v1_fixture_required')
            if args.phase == 'verify':
                fixture.write_object(
                    state_dir / 'workflow-evidence.json',
                    verify_execution(state,
                                     Client(args.endpoint, args.token_file)))
            else:
                fixture.set_enabled(state_dir, state, ActivationClient(client),
                                    args.phase == 'enable')
    except (OSError, ValueError, TypeError, KeyError, AttributeError):
        parser.exit(
            1, 'V1 fixture operation failed; inspect isolated fixture state.\n')


if __name__ == '__main__':
    main()
