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

import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from unittest import mock

SCRIPT = Path(__file__).resolve(
).parents[2] / '.github/resources/scripts/readiness-schedules.sh'


class LiveCITests(unittest.TestCase):

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
