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
"""Check CI prerequisites fail before cluster or credential operations."""

import copy
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).resolve(
).parents[2] / '.github/resources/scripts/readiness-schedules.sh'


class LiveCITests(unittest.TestCase):

    def test_api_policy_cutover_waits_for_actual_pods_and_fails_closed(self):
        script = SCRIPT.read_text()
        function = script[script.index('configure_api() {'):script
                          .index('configure_controllers() {')]
        ready = dict(
            kind='Pod',
            metadata=dict(namespace='kubeflow', uid='new-enforce'),
            status=dict(
                phase='Running', conditions=[dict(type='Ready',
                                                  status='True')]),
            spec=dict(containers=[
                dict(
                    name='ml-pipeline-api-server',
                    env=[
                        dict(name='MULTIUSER', value='true'),
                        dict(
                            name='KFP_SECURITY_SERVICE_ACCOUNT_MODE',
                            value='enforce'),
                        dict(
                            name='KFP_SECURITY_WORKFLOW_IDENTITY_MODE',
                            value='enforce')
                    ])
            ]))
        stale = copy.deepcopy(ready)
        stale['metadata']['uid'] = 'old-audit'
        stale['spec']['containers'][0]['env'][1]['value'] = 'audit'
        terminating = copy.deepcopy(ready)
        terminating['metadata'].update(
            uid='terminating', deletionTimestamp='2026-01-01T00:00:00Z')
        unready = copy.deepcopy(ready)
        unready['status']['conditions'][0]['status'] = 'False'
        wrong_container = copy.deepcopy(ready)
        wrong_container['spec']['containers'][0]['name'] = 'other'
        malformed_env = copy.deepcopy(ready)
        malformed_env['spec']['containers'][0]['env'].append(
            dict(name='KFP_SECURITY_SERVICE_ACCOUNT_MODE', value='enforce'))
        unauthenticated = copy.deepcopy(ready)
        unauthenticated['spec']['containers'][0]['env'][0]['value'] = 'false'
        fixtures = [
            dict(items=[ready, stale]),
            dict(items=[ready, terminating]),
            dict(items=[]),
            dict(items=[unready]),
            dict(items=[wrong_container]),
            dict(items=[malformed_env]),
            dict(items=[unauthenticated]),
            dict(items='malformed')
        ]
        for snapshot in fixtures:
            for times_out in (False, True):
                with self.subTest(
                        snapshot=snapshot, times_out=times_out
                ), tempfile.TemporaryDirectory() as tmp:
                    root = Path(tmp)
                    (root / 'first.json').write_text(json.dumps(snapshot))
                    (root / 'ready.json').write_text(
                        json.dumps(dict(items=[ready])))
                    setup = """set -euo pipefail
phase=target
reports=$TEST_DIR
kube() {
  case "$*" in
    '-n kubeflow set env deployment/ml-pipeline '*) ;;
    '-n kubeflow rollout status deployment/ml-pipeline --timeout=300s') ;;
    '-n kubeflow get pods -l app=ml-pipeline -o json')
      local count=0
      [[ ! -e "$TEST_DIR/queries" ]] || count=$(cat "$TEST_DIR/queries")
      count=$((count + 1))
      printf '%s' "$count" > "$TEST_DIR/queries"
      if [[ "$count" == 1 || "$TIMES_OUT" == 1 ]]; then
        cat "$TEST_DIR/first.json"
      else
        cat "$TEST_DIR/ready.json"
      fi ;;
    *) return 8 ;;
  esac
}
sleep() { if [[ "$TIMES_OUT" == 1 ]]; then SECONDS=$((SECONDS + 121)); fi; }
"""
                    result = subprocess.run(
                        [
                            'bash', '-c',
                            setup + function + '\nconfigure_api enforce\n'
                        ],
                        env=dict(
                            os.environ,
                            TEST_DIR=tmp,
                            TIMES_OUT='1' if times_out else '0'),
                        capture_output=True,
                        text=True,
                        timeout=5)
                    reports = list(root.glob('api-policy-cutover-*.json'))
                    self.assertTrue(
                        (root / 'queries').exists(),
                        'rollout status alone cannot prove policy cutover')
                    if times_out:
                        self.assertNotEqual(result.returncode, 0)
                        self.assertEqual(reports, [])
                    else:
                        self.assertEqual(result.returncode, 0, result.stderr)
                        self.assertEqual((root / 'queries').read_text(), '2')
                        self.assertEqual(len(reports), 1)
                        evidence = json.loads(reports[0].read_text())
                        self.assertEqual(evidence['pod_uids'], ['new-enforce'])
                        self.assertEqual(evidence['service_account_mode'],
                                         'enforce')
                        self.assertEqual(evidence['workflow_identity_mode'],
                                         'enforce')
                        self.assertEqual(evidence['outcome'], 'passed')
                        self.assertNotIn('env', evidence)

    def test_drain_covers_persistence_backoff_and_retains_failure_evidence(
            self):
        import time

        import kfp_http
        import live_schedule_check
        import source_schedule_check

        program = SCRIPT.read_text().split("<<'PYDRAIN'\n",
                                           1)[1].split('\nPYDRAIN', 1)[0]
        scenarios = [
            ('late_success', 360, None),
            ('timeout', 900, 'fixture_runs_not_drained'),
            ('failed', 0, 'fixture_run_did_not_succeed'),
            ('blocked', 0, 'blocked_schedule_created_run'),
            ('collection', 0, 'collection_failed'),
        ]
        for scenario, ready_at, reason in scenarios:
            with self.subTest(
                    scenario=scenario), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                (root / 'fixture').mkdir()
                (root / 'reports').mkdir()
                (root / 'fixture/state.json').write_text(
                    json.dumps(dict(namespace='test')))
                (root / 'fixture/activation-start.txt'
                ).write_text('2026-01-01T00:00:00Z')
                cases = [
                    dict(
                        scenario=name,
                        schedule_uid=name,
                        service_account=name,
                        expected_outcome='blocked'
                        if name == 'denied' else 'run_succeeded')
                    for name in ('default', 'scoped', 'denied')
                ]
                (root / 'reports/restored-baseline.json').write_text(
                    json.dumps(dict(cases=cases)))
                clock = [0]

                def collect(client, namespace, case, start):
                    if scenario == 'collection':
                        raise ValueError(
                            'secret/raw response must not reach the report')
                    if case['scenario'] == 'denied' and scenario != 'blocked':
                        return []
                    state = 'FAILED' if scenario == 'failed' else (
                        'SUCCEEDED' if clock[0] >= ready_at else 'RUNNING')
                    return [dict(run_id='run-' + case['scenario'], state=state)]

                def sleep(seconds):
                    clock[0] += seconds

                with mock.patch.object(sys, 'argv', ['drain', tmp, 'restored', str(root / 'fixture')]), \
                     mock.patch.object(time, 'monotonic', side_effect=lambda: clock[0]), \
                     mock.patch.object(time, 'sleep', side_effect=sleep), \
                     mock.patch.object(kfp_http, 'Client'), \
                     mock.patch.object(live_schedule_check, 'run_evidence', side_effect=collect), \
                     mock.patch.object(source_schedule_check, 'diagnostics', return_value={'safe_counts': True}):
                    if reason is None:
                        exec(compile(program, str(SCRIPT), 'exec'), {})
                    else:
                        with self.assertRaises(SystemExit) as failure:
                            exec(compile(program, str(SCRIPT), 'exec'), {})
                        self.assertIn('Fixture completion failed',
                                      str(failure.exception))
                report = json.loads(
                    (root / 'reports/restored-completion.json').read_text())
                self.assertNotIn('secret', json.dumps(report))
                if reason is None:
                    self.assertEqual(report['outcome'], 'passed')
                    self.assertEqual(clock[0], 360)
                    self.assertEqual(len(report['cases']), 3)
                else:
                    self.assertEqual(report['outcome'], 'inconclusive')
                    self.assertEqual(report['reason'], reason)
                    self.assertEqual(report['evidence_scope'],
                                     'last_successful_collection_per_case')
                    if scenario == 'timeout':
                        self.assertEqual(clock[0], 600)
                        self.assertEqual(report['cases'][0]['runs'][0]['state'],
                                         'RUNNING')
                    elif scenario == 'failed':
                        self.assertEqual(report['cases'][0]['runs'][0]['state'],
                                         'FAILED')
                    elif scenario == 'blocked':
                        self.assertEqual(report['cases'][-1]['scenario'],
                                         'denied')
                        self.assertEqual(
                            report['cases'][-1]['runs'][0]['run_id'],
                            'run-denied')
                    else:
                        self.assertEqual(report['cases'], [])

    def test_controller_namespace_replaces_existing_flags(self):
        script = SCRIPT.read_text()
        function = script[script.index('configure_controllers() {'):script
                          .index('restore_controller_namespaces() {')]
        target = '--managed-namespace=kfp-readiness-test'
        cases = [
            (None, [target]),
            ([], [target]),
            (['--loglevel=info'], ['--loglevel=info', target]),
            (['--managed-namespace=old',
              '--loglevel=info'], ['--loglevel=info', target]),
            (['--managed-namespace', 'old',
              '--loglevel=info'], ['--loglevel=info', target]),
            ([target, '--managed-namespace=old'], [target]),
            ([target], None),
        ]
        for args, expected in cases:
            with self.subTest(args=args), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                container = {} if args is None else {'args': args}
                deployment = {
                    'spec': {
                        'template': {
                            'spec': {
                                'containers': [container]
                            }
                        }
                    }
                }
                (root / 'deployment.json').write_text(json.dumps(deployment))
                setup = 'set -euo pipefail\nnamespace=kfp-readiness-test\nkube() {\n  case "$*" in\n    *" get "*) cat "$TEST_DIR/deployment.json" ;;\n    *" patch "*) printf \'%s\' "${@: -1}" > "$TEST_DIR/patch.json" ;;\n  esac\n}\n'
                result = subprocess.run([
                    'bash', '-c', setup + function + '\nconfigure_controllers'
                ],
                                        env=dict(os.environ, TEST_DIR=tmp),
                                        capture_output=True,
                                        text=True,
                                        timeout=5)
                self.assertEqual(result.returncode, 0, result.stderr)
                patch_file = root / 'patch.json'
                if expected is None:
                    self.assertFalse(patch_file.exists())
                else:
                    patch = json.loads(patch_file.read_text())
                    self.assertEqual(patch, [{
                        'op': 'add',
                        'path': '/spec/template/spec/containers/0/args',
                        'value': expected
                    }])

    def test_forward_owns_and_reaps_listener_and_rejects_unrelated_health(self):
        script = SCRIPT.read_text()
        functions = script[script.index('start_forward() {'):script
                           .index('mint_token() {')]
        for binds in (True, False):
            with self.subTest(
                    binds=binds), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                kubectl = root / 'kubectl'
                kubectl.write_text(
                    '#!' + sys.executable + '\n' + 'import os, time\n' +
                    'from pathlib import Path\n' +
                    'Path(os.environ["LISTENER_PID"]).write_text(str(os.getpid()))\n'
                    +
                    ('print("Forwarding from 127.0.0.1:8888 -> 8888", flush=True)\ntime.sleep(30)\n'
                     if binds else 'raise SystemExit(1)\n'))
                kubectl.chmod(0o755)
                curl = root / 'curl'
                curl.write_text('#!/bin/sh\ntouch "$CURL_CALLED"\nexit 0\n')
                curl.chmod(0o755)
                env = dict(
                    os.environ,
                    PATH=tmp + os.pathsep + os.environ['PATH'],
                    LISTENER_PID=str(root / 'listener-pid'),
                    CURL_CALLED=str(root / 'curl-called'))
                # Keep the old wrapper definition so this regression also detects
                # accidentally reverting to backgrounding kube() instead of exec.
                setup = 'set -euo pipefail\ncontext=kind-kfp-readiness\nstate=' + tmp + '\nendpoint=http://127.0.0.1:8888\nkube() { kubectl "$@"; }\n'
                check = (
                    'start_forward\n[[ "$forward_pid" == "$(cat "$LISTENER_PID")" ]]\n'
                    'listener=$forward_pid\nstop_forward\n! kill -0 "$listener" 2>/dev/null\n'
                    if binds else
                    'if start_forward; then exit 9; fi\nstop_forward\n[[ ! -e "$CURL_CALLED" ]]\n'
                )
                try:
                    result = subprocess.run(
                        ['bash', '-c', setup + functions + check],
                        env=env,
                        capture_output=True,
                        text=True,
                        timeout=8)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual((root / 'curl-called').exists(), binds)
                finally:
                    # Also clean a leaked child if this test catches a regression.
                    if (root / 'listener-pid').exists():
                        try:
                            os.kill(
                                int((root / 'listener-pid').read_text()), 15)
                        except ProcessLookupError:
                            pass

    def test_audit_window_covers_existing_controller_retry_backoff(self):
        observe = SCRIPT.read_text().split('observe() {', 1)[1].split('\n}',
                                                                      1)[0]
        self.assertIn('local mode=$1 timeout=180', observe)
        self.assertIn(
            '[[ "$mode" == enforce || "$mode" == v1 ]] || timeout=600', observe)
        self.assertIn('--timeout-seconds "$timeout"', observe)
        self.assertNotIn('rollout restart', observe)

    def test_release_upgrade_is_pinned_and_schedule_lane_requires_opt_in(self):
        workflow = SCRIPT.parents[2] / 'workflows/upgrade-test.yml'
        text = workflow.read_text()
        self.assertIn('lastRelease=2.17.2', text)
        self.assertNotIn('releases/latest', text)
        self.assertIn('branches: [release-2.18]', text)
        lane = text.split('  readiness-schedules:', 1)[1]
        self.assertIn("vars.KFP_218_READINESS_SCHEDULES == 'enabled'", lane)
        self.assertIn("github.event_name == 'workflow_dispatch'", lane)
        self.assertIn('inputs.run_readiness_schedules', lane)
        dispatch = text.split('  workflow_dispatch:',
                              1)[1].split('  pull_request:', 1)[0]
        self.assertRegex(
            dispatch,
            r'run_readiness_schedules:[\s\S]*type: boolean[\s\S]*default: false'
        )
        self.assertIn("steps.prepare-upgrade.outcome == 'success'", text)
        self.assertNotIn('KFP_ENABLE_MLMD_UPGRADE_TESTS', text)

    def test_legacy_rejection_precedes_recreated_functional_observation(self):
        script = SCRIPT.read_text()
        target = script.split('else\n  configure_api enforce', 1)[1]
        self.assertLess(
            target.index('verify_legacy_schedules.py'),
            target.index('fixture --phase recreate'))
        self.assertLess(
            target.index('fixture --phase recreate'),
            target.index('capture enforce'))
        self.assertLess(
            target.index('capture enforce'), target.index('observe enforce'))
        self.assertIn('source-$mode-prediction.json', script)
        self.assertIn('check_fixture_policy.py', target)
        self.assertIn('--legacy-migration', script)
        self.assertNotIn('remap_predictions', script)

    def test_source_namespace_restore_occurs_after_drain_and_baseline(self):
        text = SCRIPT.read_text()
        restore = text.index('\n  restore_controller_namespaces\n')
        self.assertLess(text.index('\n  drain source\n'), restore)
        self.assertLess(text.index('--legacy-migration'), restore)
        self.assertLess(restore, text.index('else\n  configure_api enforce'))

    def test_namespace_restore_removes_literal_and_checks_live_shape(self):
        function = re.search(
            r'(restore_controller_namespaces\(\) \{[\s\S]*?^\})',
            SCRIPT.read_text(), re.MULTILINE).group(1)
        for invalid in (False, True):
            # Only kubectl is replaced; execute the real jq patch construction
            # and live-shape assertion to cover this shell transition.
            environment = dict(
                name='NAMESPACE',
                valueFrom=dict(fieldRef=dict(fieldPath='metadata.namespace')))
            if invalid:
                environment['value'] = 'fixture'
            deployments = {
                name:
                    dict(
                        spec=dict(
                            template=dict(
                                spec=dict(containers=[
                                    dict(name=name, env=[environment])
                                ]))))
                for name in ('ml-pipeline-scheduledworkflow',
                             'ml-pipeline-persistenceagent')
            }
            with tempfile.TemporaryDirectory() as directory:
                for name, deployment in deployments.items():
                    (Path(directory) / name).write_text(json.dumps(deployment))
                kube = (
                    'kube() {\n'
                    '  case "$3" in\n'
                    '    patch) printf "%s\\n" "$7" ;;\n'
                    '    rollout) ;;\n'
                    '    get) cat "$FIXTURE_DEPLOYMENTS/${4#deployment/}" ;;\n'
                    '    *) return 1 ;;\n'
                    '  esac\n'
                    '}\n')
                result = subprocess.run([
                    'bash', '-c', 'set -euo pipefail\n' + kube + function +
                    '\nrestore_controller_namespaces\n'
                ],
                                        text=True,
                                        capture_output=True,
                                        env=dict(
                                            os.environ,
                                            FIXTURE_DEPLOYMENTS=directory))
            if invalid:
                self.assertNotEqual(result.returncode, 0)
            else:
                self.assertEqual(result.returncode, 0, result.stderr)
                patches = [
                    json.loads(line) for line in result.stdout.splitlines()
                ]
                self.assertEqual(len(patches), 2)
                for patch, name in zip(patches, deployments):
                    container = patch['spec']['template']['spec']['containers'][
                        0]
                    self.assertEqual(
                        container,
                        dict(
                            name=name,
                            env=[
                                dict(
                                    name='NAMESPACE',
                                    value=None,
                                    valueFrom=dict(
                                        fieldRef=dict(
                                            fieldPath='metadata.namespace')))
                            ]))

    def test_embedded_python_compiles(self):
        blocks = [
            body for _, body in re.findall(r"<<'(PY[A-Z]*)'\n(.*?)\n\1\n",
                                           SCRIPT.read_text(), re.DOTALL)
        ]
        self.assertTrue(blocks)
        for block in blocks:
            compile(block, str(SCRIPT), 'exec')

    def test_phase_drain_requires_successful_runs_after_schedules_stop(self):
        body = re.search(r"<<'PYDRAIN'\n(.*?)\nPYDRAIN\n", SCRIPT.read_text(),
                         re.DOTALL).group(1)
        for phase in ('source', 'enforce', 'audit'):
            for run_state in ('RUNNING', 'FAILED', 'CANCELED', 'SKIPPED',
                              'UNKNOWN', 'SUCCEEDED'):
                with self.subTest(phase=phase, state=run_state):
                    self.run_drain(
                        body,
                        phase, [{
                            'run_id': 'new',
                            'state': run_state
                        }],
                        passes=run_state == 'SUCCEEDED')

    def run_drain(self, body, phase, records, *, passes, blocked=False):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'fixture').mkdir()
            (root / 'reports').mkdir()
            case = {
                'schedule_uid': 'uid',
                'scenario': 'test',
                'service_account': 'runner',
                'baseline_run_ids': ['source-run'],
                'expected_outcome': 'blocked' if blocked else 'run_created'
            }
            (root / 'fixture/state.json').write_text(
                json.dumps({
                    'namespace': 'fixture',
                    'schedules': [case]
                }))
            (root /
             'fixture/activation-start.txt').write_text('2026-01-01T00:00:00Z')
            (root / f'reports/{phase}-baseline.json').write_text(
                json.dumps({'cases': [case]}))
            with mock.patch(
                    'sys.argv',
                ['drain', directory, phase
                ]), mock.patch('kfp_http.Client'), mock.patch(
                    'source_schedule_check.source_run_evidence' if phase
                    == 'source' else 'live_schedule_check.run_evidence',
                    return_value=records) as evidence, mock.patch(
                        'time.monotonic',
                        side_effect=[0, 0, 301
                                    ]), mock.patch('time.sleep'), mock.patch(
                                        'source_schedule_check.diagnostics',
                                        return_value={}):
                if passes:
                    exec(compile(body, str(SCRIPT), 'exec'), {})
                else:
                    with self.assertRaises(SystemExit):
                        exec(compile(body, str(SCRIPT), 'exec'), {})
            self.assertEqual(evidence.call_args.args[2]['baseline_run_ids'],
                             [] if phase == 'source' else ['source-run'])
            report = root / f'reports/{phase}-completion.json'
            self.assertTrue(report.exists())
            value = json.loads(report.read_text())
            self.assertEqual(value['outcome'],
                             'passed' if passes else 'inconclusive')
            if passes:
                value = json.loads(report.read_text())
                self.assertEqual(value['cases'][0]['runs'], records)
                self.assertTrue(value['all_expected_runs_succeeded'])

    def test_target_completion_rejects_late_blocked_or_failed_runs(self):
        body = re.search(r"<<'PYDRAIN'\n(.*?)\nPYDRAIN\n", SCRIPT.read_text(),
                         re.DOTALL).group(1)
        self.run_drain(body, 'enforce', [], passes=True, blocked=True)
        self.run_drain(
            body,
            'enforce', [{
                'run_id': 'late',
                'state': 'RUNNING'
            }],
            passes=False,
            blocked=True)
        self.run_drain(
            body,
            'audit', [{
                'run_id': 'first',
                'state': 'SUCCEEDED'
            }, {
                'run_id': 'late',
                'state': 'FAILED'
            }],
            passes=False)
        self.run_drain(body, 'audit', [], passes=False)

    def test_absent_first_policy_marker_fails_even_with_later_markers(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            files = {
                'backend/src/apiserver/common/config.go':
                    'KFP_SECURITY_WORKFLOW_IDENTITY_MODE',
                'backend/src/apiserver/resource/resource_manager.go':
                    'authorizeServiceAccountWithPolicy',
                'backend/src/apiserver/resource/recurring_run.go':
                    'run.ServiceAccount = job.ServiceAccount\nauthorizeStoredRunServiceAccount',
            }
            for name, content in files.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
            binary = root / 'bin'
            binary.mkdir()
            git = binary / 'git'
            git.write_text(
                '#!/bin/sh\nprintf "%s\\n" aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n'
            )
            git.chmod(0o755)
            env = dict(
                os.environ,
                RUNNER_TEMP=directory,
                PATH=str(binary) + os.pathsep + os.environ['PATH'])
            result = subprocess.run(['bash', str(SCRIPT), 'preflight'],
                                    cwd=directory,
                                    env=env,
                                    text=True,
                                    capture_output=True,
                                    check=False)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('prerequisites', result.stdout)


if __name__ == '__main__':
    unittest.main()
