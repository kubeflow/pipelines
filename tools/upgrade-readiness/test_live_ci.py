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

    def test_embedded_python_compiles(self):
        blocks = [
            body for _, body in re.findall(r"<<'(PY[A-Z]*)'\n(.*?)\n\1\n",
                                           SCRIPT.read_text(), re.DOTALL)
        ]
        self.assertTrue(blocks)
        for block in blocks:
            compile(block, str(SCRIPT), 'exec')

    def test_phase_drain_requires_terminal_runs_and_source_success(self):
        body = re.search(r"<<'PYDRAIN'\n(.*?)\nPYDRAIN\n", SCRIPT.read_text(),
                         re.DOTALL).group(1)
        for phase, run_state, passes in [('target', 'RUNNING', False),
                                         ('source', 'FAILED', False),
                                         ('source', 'SUCCEEDED', True),
                                         ('target', 'FAILED', True)]:
            with self.subTest(
                    phase=phase, state=run_state), tempfile.TemporaryDirectory(
                    ) as directory:
                root = Path(directory)
                (root / 'fixture').mkdir()
                (root / 'reports').mkdir()
                (root / 'fixture/state.json').write_text(
                    json.dumps({
                        'namespace': 'fixture',
                        'schedules': [{
                            'schedule_uid': 'uid'
                        }]
                    }))
                with mock.patch(
                        'sys.argv',
                    ['drain', directory, phase
                    ]), mock.patch('kfp_http.Client'), mock.patch(
                        'live_schedule_check.list_runs',
                        return_value=[{
                            'state': run_state
                        }]), mock.patch(
                            'time.monotonic',
                            side_effect=[0, 0, 301]), mock.patch('time.sleep'):
                    if passes:
                        exec(compile(body, str(SCRIPT), 'exec'), {})
                    else:
                        with self.assertRaises(SystemExit):
                            exec(compile(body, str(SCRIPT), 'exec'), {})
                self.assertEqual(
                    (root / 'reports/source-completion.json').exists(),
                    phase == 'source' and passes)

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
