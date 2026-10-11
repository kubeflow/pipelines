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
"""Compare modeled policy with a pinned backend checkout; not a live upgrade
test."""

import argparse
import json
import math
import os
from pathlib import Path
import re
import signal
import subprocess
import tempfile

import schedule_policy

CONFORMANCE_TIMEOUT_SECONDS = 900
GIT_TIMEOUT_SECONDS = 30


def cases(target_revision=schedule_policy.POLICY_SOURCE):
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
            target_revision=target_revision,
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
                expected_sar_requests=int(
                    account != 'pipeline-runner' and
                    (mode == 'audit' or account
                     in [name.strip() for name in allowlist.split(',')])),
                prediction=prediction))
    return results


def _run_backend(command, source, environment, timeout_seconds):
    # Go launches compiler and test subprocesses; kill the entire session on a
    # timeout or interruption so they cannot outlive the temporary overlay.
    with subprocess.Popen(
            command, cwd=source, env=environment,
            start_new_session=True) as process:
        try:
            returncode = process.wait(timeout=timeout_seconds)
        except BaseException:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            raise
        if returncode:
            raise subprocess.CalledProcessError(returncode, command)


def run(source,
        target_revision=schedule_policy.POLICY_SOURCE,
        timeout_seconds=CONFORMANCE_TIMEOUT_SECONDS):
    if not isinstance(target_revision, str) or not re.fullmatch(
            r'[0-9a-fA-F]{40}', target_revision):
        raise ValueError('Supply a full 40-character target revision.')
    target_revision = target_revision.lower()
    if (not math.isfinite(timeout_seconds) or not 0 < timeout_seconds <= 3600):
        raise ValueError(
            'Timeout must be greater than zero and at most 3600 seconds.')
    source = source.resolve()
    revision = subprocess.check_output(
        ['git', '-c', 'core.fsmonitor=false', 'rev-parse', 'HEAD'],
        cwd=source,
        text=True,
        timeout=GIT_TIMEOUT_SECONDS).strip()
    if revision != target_revision:
        raise ValueError(
            'Backend checkout must match the requested target revision.')
    dirty = subprocess.check_output([
        'git', '-c', 'core.fsmonitor=false', 'status', '--porcelain',
        '--untracked-files=all'
    ],
                                    cwd=source,
                                    text=True,
                                    timeout=GIT_TIMEOUT_SECONDS)
    if dirty:
        raise ValueError(
            'Use a clean backend checkout for reproducible conformance.')
    with tempfile.TemporaryDirectory(
            prefix='kfp-readiness-conformance-') as directory:
        directory = Path(directory)
        case_path = directory / 'cases.json'
        case_path.write_text(json.dumps(cases(target_revision)))
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
        _run_backend([
            'go', 'test', '-overlay',
            str(overlay_path), './backend/src/apiserver/resource', '-run',
            '^TestReadinessPolicyConformance$', '-count=1', '-v'
        ], source, environment, timeout_seconds)
    print('Policy-code conformance passed.\n'
          'Modeled policy contract: ' + schedule_policy.CONTRACT + '\n'
          'Modeled policy source: ' + schedule_policy.POLICY_SOURCE + '\n'
          'Tested backend revision: ' + revision + '\n'
          'Live upgrade validation remains unassessed.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--backend-source', type=Path, required=True)
    parser.add_argument(
        '--target-revision',
        default=schedule_policy.POLICY_SOURCE,
        help='Full candidate commit SHA; defaults to the modeled policy source.'
    )
    parser.add_argument(
        '--timeout-seconds',
        type=float,
        default=CONFORMANCE_TIMEOUT_SECONDS,
        help='Backend test deadline in seconds (default: 900, maximum: 3600).')
    args = parser.parse_args()
    try:
        run(args.backend_source, args.target_revision, args.timeout_seconds)
    except subprocess.TimeoutExpired:
        parser.exit(
            1, 'Policy conformance failed: command exceeded its deadline.\n')
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        parser.exit(1, 'Policy conformance failed: ' + str(error) + '\n')


if __name__ == '__main__':
    main()
