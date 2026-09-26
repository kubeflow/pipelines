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
"""Compare predictions with pinned backend policy code; not a live upgrade
test."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

import schedule_policy


def cases():
    fixtures = [
        ('default-exemption', 'pipeline-runner', '', 'enforce', False, None),
        ('custom-scoped-grant', 'custom', 'custom', 'enforce', True, 'custom'),
        ('custom-denied', 'custom', 'custom', 'enforce', False, None),
        ('audit-denied', 'custom', 'custom', 'audit', False, None),
        ('allowlist-denied', 'custom', '', 'enforce', True, 'custom'),
        ('audit-allowlist-denied', 'custom', '', 'audit', True, 'custom'),
        ('literal-allowlist-star', 'custom', '*', 'enforce', True, 'custom'),
        ('wrong-named-grant', 'custom', 'custom', 'enforce', False, 'other'),
        ('trimmed-allowlist', 'custom', ' custom ', 'enforce', True, 'custom'),
        ('group-only-grant', 'custom', 'custom', 'enforce', False, 'group'),
    ]
    results = []
    for name, account, allowlist, mode, sar_allowed, granted_name in fixtures:
        rbac = []
        if granted_name:
            rbac = [
                dict(
                    kind='Role',
                    metadata=dict(name='use', namespace='ns1'),
                    rules=[
                        dict(
                            apiGroups=[''],
                            resources=['serviceaccounts'],
                            verbs=['use'],
                            resourceNames=[
                                account
                                if granted_name == 'group' else granted_name
                            ])
                    ]),
                dict(
                    kind='RoleBinding',
                    metadata=dict(name='use', namespace='ns1'),
                    roleRef=dict(
                        apiGroup='rbac.authorization.k8s.io',
                        kind='Role',
                        name='use'),
                    subjects=[
                        dict(
                            apiGroup='rbac.authorization.k8s.io',
                            kind='Group' if granted_name == 'group' else 'User',
                            name='system:authenticated'
                            if granted_name == 'group' else 'user@google.com')
                    ])
            ]
        bundle = dict(
            policy_contract=schedule_policy.CONTRACT,
            target_revision=schedule_policy.POLICY_SOURCE,
            multi_user=True,
            mode=mode,
            default_service_account='pipeline-runner',
            controller_user='user@google.com',
            allowed_service_accounts=allowlist.split(',') if allowlist else [],
            rbac_complete=True,
            rbac_only=True,
            rbac=rbac,
            recurring_runs=[
                dict(
                    recurring_run_id='job',
                    experiment_id='exp',
                    service_account=account)
            ],
            experiments=[dict(experiment_id='exp', namespace='ns1')])
        schedule = dict(
            kind='ScheduledWorkflow',
            metadata=dict(name='fixture', namespace='ns1', uid='job'),
            spec={})
        prediction = schedule_policy.assess(schedule,
                                            schedule_policy.validate(bundle))[0]
        results.append(
            dict(
                name=name,
                account=account,
                allowlist=allowlist,
                mode=mode,
                default_account='pipeline-runner',
                sar_allowed=sar_allowed,
                prediction=prediction))
    return results


def run(source):
    source = source.resolve()
    revision = subprocess.check_output(
        ['git', '-c', 'core.fsmonitor=false', 'rev-parse', 'HEAD'],
        cwd=source,
        text=True).strip()
    if revision != schedule_policy.POLICY_SOURCE:
        raise ValueError(
            'Backend checkout must match the pinned policy source.')
    dirty = subprocess.check_output([
        'git', '-c', 'core.fsmonitor=false', 'status', '--porcelain',
        '--untracked-files=all'
    ],
                                    cwd=source,
                                    text=True)
    if dirty:
        raise ValueError(
            'Use a clean backend checkout for reproducible conformance.')
    with tempfile.TemporaryDirectory(
            prefix='kfp-readiness-conformance-') as directory:
        directory = Path(directory)
        case_path = directory / 'cases.json'
        case_path.write_text(json.dumps(cases()))
        template = Path(
            __file__).parent / 'conformance' / 'backend_policy_test.go.tmpl'
        overlay_path = directory / 'overlay.json'
        overlay_path.write_text(
            json.dumps({
                'Replace': {
                    str(source /
                        'backend/src/apiserver/resource/readiness_conformance_test.go'
                       ):
                        str(template.resolve())
                }
            }))
        environment = dict(os.environ, KFP_READINESS_CASES=str(case_path))
        subprocess.run([
            'go', 'test', '-overlay',
            str(overlay_path), './backend/src/apiserver/resource', '-run',
            '^TestReadinessPolicyConformance$', '-count=1', '-v'
        ],
                       cwd=source,
                       env=environment,
                       check=True)
    print('Policy-code conformance passed for ' + revision +
          '; live upgrade validation remains unassessed.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--backend-source', type=Path, required=True)
    args = parser.parse_args()
    try:
        run(args.backend_source)
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        parser.exit(1, 'Policy conformance failed: ' + str(error) + '\n')


if __name__ == '__main__':
    main()
