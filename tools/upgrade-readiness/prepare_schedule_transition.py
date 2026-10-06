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
"""Prepare isolated, drained schedule transitions from verified live
evidence."""

import argparse
import json
from pathlib import Path
import subprocess
import time

from build_live_policy import build_policy
from kubectl_inventory import kubectl_get
from provision_live_schedules import CONTEXT
from provision_live_schedules import fixture_rbac
from provision_live_schedules import NAMESPACE
from provision_live_schedules import read_object
from provision_live_schedules import verify_state
from provision_live_schedules import write_object

PREVIOUS = {
    'audit-enforce': 'audit',
    'revoked': 'audit-enforce',
    'restored': 'revoked'
}


def cases_for_transition(fixture, previous, phase):
    if (phase not in PREVIOUS or fixture.get('enabled') is not False or
            not fixture.get('recreated') or not fixture.get('prepared') or
            previous.get('mode') != PREVIOUS[phase] or
            previous.get('scope') != 'fixture_run_completion' or
            previous.get('outcome') != 'passed' or
            previous.get('all_expected_runs_succeeded') is not True or
            previous.get('namespace') != NAMESPACE):
        raise ValueError('drained_previous_phase_required')
    cases = []
    if len(fixture['schedules']) != 3 or len(previous['cases']) != 3:
        raise ValueError('three_cases_required')
    for record in fixture['schedules']:
        matches = [
            c for c in previous['cases'] if all(
                c.get(k) == record[k]
                for k in ('scenario', 'schedule_uid', 'service_account'))
        ]
        if len(matches) != 1:
            raise ValueError('previous_identity_mismatch')
        prior_blocked = (record['scenario'] == 'denied' and
                         phase != 'audit-enforce') or (record['scenario']
                                                       == 'scoped' and
                                                       phase == 'restored')
        runs = matches[0]['runs']
        if (prior_blocked and runs or not prior_blocked and
            (not runs or any(r.get('state') != 'SUCCEEDED' for r in runs))):
            raise ValueError('previous_execution_not_established')
        blocked = record['scenario'] == 'denied' or (
            phase == 'revoked' and record['scenario'] == 'scoped')
        cases.append(
            dict(
                record,
                expected_outcome='blocked' if blocked else 'run_created',
                expected_prediction='policy_rejection'
                if blocked else 'no_issue_detected'))
    if {c['scenario'] for c in cases} != {'default', 'scoped', 'denied'}:
        raise ValueError('expected_cases_required')
    return dict(namespace=NAMESPACE, cases=cases)


def grant_patch(role, restore):
    expected = next(r['rules']
                    for r in fixture_rbac([])
                    if r['kind'] == 'Role' and
                    r['metadata']['name'] == 'fixture-controller')
    before, after = (expected[:1], expected) if restore else (expected,
                                                              expected[:1])
    metadata = role.get('metadata', {})
    if (role.get('kind') != 'Role' or
            metadata.get('name') != 'fixture-controller' or
            metadata.get('namespace') != NAMESPACE or
            role.get('rules') != before or not metadata.get('resourceVersion')):
        raise ValueError('unexpected_controller_role')
    return [
        dict(
            op='test',
            path='/metadata/resourceVersion',
            value=metadata['resourceVersion']),
        dict(op='test', path='/rules', value=before),
        dict(op='replace', path='/rules', value=after)
    ]


def review_use(context, user, expected):
    request = dict(
        apiVersion='authorization.k8s.io/v1',
        kind='SubjectAccessReview',
        spec=dict(
            user=user,
            resourceAttributes=dict(
                namespace=NAMESPACE,
                group='',
                resource='serviceaccounts',
                name='readiness-granted',
                verb='use')))
    result = subprocess.run([
        'kubectl', '--context', context, '--request-timeout=20s', 'create',
        '-f', '-', '-o', 'json'
    ],
                            input=json.dumps(request),
                            text=True,
                            capture_output=True,
                            timeout=30,
                            check=True)
    if len(result.stdout) > 65536:
        raise ValueError('authorization_response_limit')
    status = json.loads(result.stdout).get('status', {})
    if (status.get('allowed') is not expected or
            status.get('evaluationError') or
        (expected and status.get('denied'))):
        raise ValueError('unexpected_live_authorization')


def wait_for_use(context, user, expected):
    # RBAC authorization caches may lag the patch. Never start observations
    # until the live authorizer confirms the cutover.
    for attempt in range(30):
        try:
            review_use(context, user, expected)
            return
        except ValueError:
            if attempt == 29:
                raise
            time.sleep(2)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--context', required=True, choices=[CONTEXT])
    parser.add_argument('--state-dir', type=Path, required=True)
    parser.add_argument('--phase', required=True, choices=list(PREVIOUS))
    args = parser.parse_args()
    try:
        state = args.state_dir
        fixture = read_object(state / 'fixture/state.json')
        verify_state(args.context, fixture)
        cases = cases_for_transition(
            fixture,
            read_object(state /
                        f'reports/{PREVIOUS[args.phase]}-completion.json'),
            args.phase)
        if args.phase in ('revoked', 'restored'):
            role, error = kubectl_get(args.context, NAMESPACE,
                                      'role/fixture-controller')
            if error:
                raise ValueError('controller_role_unavailable')
            patch = grant_patch(role, args.phase == 'restored')
            subprocess.run([
                'kubectl', '--context', args.context, '--request-timeout=20s',
                '-n', NAMESPACE, 'patch', 'role/fixture-controller',
                '--type=json', '-p',
                json.dumps(patch)
            ],
                           stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL,
                           timeout=30,
                           check=True)
        wait_for_use(
            args.context,
            'system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow',
            args.phase != 'revoked')
        wait_for_use(args.context,
                     'system:serviceaccount:' + NAMESPACE + ':fixture-owner',
                     True)
        rbac, error = kubectl_get(
            args.context,
            None,
            'roles,rolebindings,clusterroles,clusterrolebindings',
            all_namespaces=True)
        if error:
            raise ValueError('live_rbac_unavailable')
        revision = read_object(state /
                               'reports/revisions.json')['target_revision']
        policy = build_policy(rbac, {'items': []}, revision, 'enforce')
        write_object(state / f'{args.phase}-policy.json', policy)
        write_object(state / f'{args.phase}-cases.json', cases)
        write_object(
            state / f'reports/{args.phase}-cutover.json',
            dict(
                scope='isolated_schedule_transition',
                phase=args.phase,
                previous_phase=PREVIOUS[args.phase],
                previous_runs_drained=True,
                controller_use_allowed=args.phase != 'revoked',
                owner_use_allowed=True,
                namespace=NAMESPACE,
                schedule_uids=[c['schedule_uid'] for c in cases['cases']]))
    except (OSError, ValueError, KeyError, TypeError,
            subprocess.SubprocessError):
        parser.exit(
            1,
            'Unable to establish isolated schedule transition prerequisites.\n')


if __name__ == '__main__':
    main()
