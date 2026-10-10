# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
"""Execute reporting cleanup and verify the opt-in evidence boundary."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / '.github/resources/scripts/readiness-reporting.sh'
WORKFLOW = ROOT / '.github/workflows/upgrade-test.yml'


class ReportingCITest(unittest.TestCase):

    def test_script_defines_fixture_path_and_cleans_up_after_phase_errors(self):
        # The release library provides state, but does not define fixture_dir.
        for phase, fail_phase in (('source', ''), ('source', 'prepare'),
                                  ('target', 'recover'), ('cleanup', '')):
            with self.subTest(phase=phase, failure=fail_phase), \
                 tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                scripts = root / '.github/resources/scripts'
                scripts.mkdir(parents=True)
                library = r"""state=$TEST_STATE
helpers=helpers
endpoint=http://127.0.0.1:8888
context=kind-kfp-readiness
configure_controllers() { :; }
configure_api() { :; }
start_forward() { :; }
mint_token() { touch "$state/token"; }
restore_controller_namespaces() { :; }
cleanup() { echo cleanup >>"$state/calls"; rm -f "$state/token"; }
"""
                (scripts / 'readiness-schedules.sh').write_text(library)
                state = root / 'state'
                (state / 'fixture').mkdir(parents=True)
                (state / 'fixture/state.json').write_text('{}')
                (state / 'token').write_text('synthetic')
                commands = root / 'bin'
                commands.mkdir()
                python = commands / 'python3'
                python.write_text(r"""#!/usr/bin/env bash
set -euo pipefail
[[ "$3" == --fixture-state && "$4" == "$TEST_STATE/fixture/state.json" ]]
echo "$2" >>"$TEST_STATE/calls"
[[ "$2" != "$FAIL_PHASE" ]]
""")
                python.chmod(0o755)
                env = dict(
                    os.environ,
                    TEST_STATE=str(state),
                    FAIL_PHASE=fail_phase,
                    PATH=str(commands) + os.pathsep + os.environ['PATH'])
                env.pop('fixture_dir', None)
                result = subprocess.run(['bash', str(SCRIPT), phase],
                                        cwd=root,
                                        env=env,
                                        capture_output=True,
                                        text=True)
                self.assertEqual(result.returncode, 1 if fail_phase else 0,
                                 result.stderr)
                calls = (state / 'calls').read_text().splitlines()
                expected = [] if phase == 'cleanup' else [
                    'prepare' if phase == 'source' else 'recover'
                ]
                self.assertEqual(calls, expected + ['restore', 'cleanup'])
                self.assertFalse((state / 'token').exists())

    def test_source_restores_watchers_then_prepares_and_resets_for_apply(self):
        source = SCRIPT.read_text().split(
            'if [[ "$reporting_phase" == source ]]; then',
            1)[1].split('\nfi', 1)[0]
        self.assertLess(
            source.index('configure_controllers'),
            source.index('check prepare'))
        self.assertLess(
            source.index('check prepare'),
            source.index('restore_controller_namespaces'))

    def test_restore_attempts_both_faults_and_propagates_each_failure(self):
        source = SCRIPT.read_text()
        function = re.search(r'(restore_faults\(\) \{[\s\S]*?^\})', source,
                             re.MULTILINE).group(1)
        for failure in ('none', 'rbac', 'patch', 'rollout'):
            with self.subTest(
                    failure=failure), tempfile.TemporaryDirectory() as tmp:
                directory = Path(tmp)
                (directory / 'state.json').write_text('{}')
                (directory / 'agent-restore.json').write_text('[]')
                program = r"""set -euo pipefail
fixture_dir=$1; reporting_state=$1; failure=$2
check() { echo rbac >>"$fixture_dir/calls"; [[ "$failure" != rbac ]]; }
kube() {
  if [[ "$*" == *' patch '* ]]; then action=patch; else action=rollout; fi
  echo "$action" >>"$fixture_dir/calls"
  [[ "$failure" != "$action" ]]
}
""" + function + '\nresult=0; restore_faults || result=1; exit "$result"\n'
                result = subprocess.run(
                    ['bash', '-c', program, 'test', tmp, failure],
                    capture_output=True,
                    text=True)
                self.assertEqual(result.returncode,
                                 0 if failure == 'none' else 1)
                self.assertEqual((directory / 'calls').read_text().splitlines(),
                                 ['rbac', 'patch', 'rollout'])

    def test_agent_restore_patch_can_be_applied_twice_with_absent_original_args(
            self):
        source = SCRIPT.read_text()
        blocks = re.findall(r"<<'PY'\n([\s\S]*?)\nPY", source)
        program = next(
            block for block in blocks if 'agent-restore.json' in block)
        for original in (None, [], ['-logtostderr=true']):
            with self.subTest(
                    original=original), tempfile.TemporaryDirectory() as tmp:
                directory = Path(tmp)
                container = dict(name='ml-pipeline-persistenceagent')
                if original is not None:
                    container['args'] = original
                deployment = dict(
                    spec=dict(template=dict(spec=dict(containers=[container]))))
                (directory / 'agent-before.json').write_text(
                    json.dumps(deployment))
                result = subprocess.run(['python3', '-c', program, tmp],
                                        capture_output=True,
                                        text=True)
                self.assertEqual(result.returncode, 0, result.stderr)
                patch = json.loads(
                    (directory / 'agent-restore.json').read_text())
                current = dict(
                    args=['-mlPipelineAPIServerName=kfp-reporting-proxy'])
                for _ in range(2):
                    for operation in patch:
                        self.assertEqual(
                            operation['path'],
                            '/spec/template/spec/containers/0/args')
                        if operation['op'] == 'add':
                            current['args'] = operation['value']
                        elif operation['op'] == 'remove':
                            self.assertIn('args', current,
                                          'second cleanup must remain safe')
                            del current['args']
                        else:
                            self.fail('unexpected restoration operation')
                self.assertEqual(current.get('args', []), original or [])

    def test_reporting_lane_uploads_only_sanitized_allowlist(self):
        text = WORKFLOW.read_text().split('  readiness-reporting:', 1)[1]
        self.assertIn(
            "github.event_name == 'workflow_dispatch' && inputs.run_readiness_reporting",
            text)
        upload = text.split('      - name: Upload sanitized reporting evidence',
                            1)[1]
        upload = upload.split('      - name:', 1)[0]
        self.assertIn('if: ${{ always() }}', upload)
        paths = re.findall(r'^            (.+)$', upload, re.MULTILINE)
        expected = {
            'reporting-source.json', 'reporting-blocked.json',
            'reporting-recovered.json', 'reporting-deleted.json',
            'reporting-ownership.json'
        }
        self.assertEqual({path.rsplit('/', 1)[-1] for path in paths}, expected)
        self.assertTrue(
            all(
                path.startswith(
                    '${{ runner.temp }}/readiness-schedules/reporting/') and
                '*' not in path for path in paths))
        cleanup = text.split('      - name: Restore fault injection settings',
                             1)[1]
        cleanup = cleanup.split('      - name:', 1)[0]
        self.assertIn('if: ${{ always() }}', cleanup)
        self.assertIn('readiness-reporting.sh cleanup', cleanup)

    def test_proxy_grants_exact_delete_targets_and_only_agent_network_access(
            self):
        source = SCRIPT.read_text()
        program = next(
            block for block in re.findall(r"<<'PY'\n([\s\S]*?)\nPY", source)
            if 'proxy-resources.json' in block)
        records = [
            dict(run_id='a', workflow_name='wa', workflow_uid='ua'),
            dict(run_id='b', workflow_name='wb', workflow_uid='ub')
        ]
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            (directory / 'reporting-source.json').write_text(
                json.dumps(dict(deletion_runs=records)))
            result = subprocess.run(['python3', '-c', program, tmp],
                                    capture_output=True,
                                    text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            objects = json.loads(
                (directory / 'proxy-resources.json').read_text())['items']
        role = next(obj for obj in objects if obj['kind'] == 'Role')
        self.assertEqual(role['rules'], [
            dict(
                apiGroups=['argoproj.io'],
                resources=['workflows'],
                resourceNames=['wa', 'wb'],
                verbs=['get', 'delete'])
        ])
        policy = next(obj for obj in objects if obj['kind'] == 'NetworkPolicy')
        rule = policy['spec']['ingress'][0]
        self.assertEqual(rule['from'], [
            dict(
                podSelector=dict(
                    matchLabels=dict(app='ml-pipeline-persistenceagent')))
        ])
        self.assertEqual({port['port'] for port in rule['ports']}, {8887, 8888})


if __name__ == '__main__':
    unittest.main()
