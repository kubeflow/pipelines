#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify SDK report capture preserves pytest failures."""

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class SdkTimingReportsTest(unittest.TestCase):
    def test_reports_survive_test_failure(self):
        for suffix, label in (('', 'regression'), ('-unit', 'unit')):
            for fail_call in (0, 1, 2) if not suffix else (0, 1):
                exit_code = 42 if fail_call else 0
                with self.subTest(suite=label, exit_code=exit_code):
                    with tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        executable = root / 'uv'
                        executable.write_text(
                            '#!/usr/bin/env python3\n'
                            'import json, os, pathlib, sys\n'
                            'args = sys.argv[1:]\n'
                            'if args[0] == "build":\n'
                            '    dist = pathlib.Path(args[args.index("--out-dir") + 1])\n'
                            '    (dist / "kfp-test.whl").touch()\n'
                            '    sys.exit(0)\n'
                            'capture = pathlib.Path("args.json")\n'
                            'calls = json.loads(capture.read_text()) if capture.exists() else []\n'
                            'calls.append(args)\n'
                            'capture.write_text(json.dumps(calls))\n'
                            'print("slowest durations: timing sentinel", flush=True)\n'
                            'print("stderr sentinel", file=sys.stderr, flush=True)\n'
                            'for arg in args:\n'
                            '    if arg.startswith("--junitxml="):\n'
                            '        pathlib.Path(arg.split("=", 1)[1]).write_text("<testsuites/>")\n'
                            'sys.exit(42 if len(calls) == int(os.environ["TEST_FAIL_CALL"]) else 0)\n'
                        )
                        executable.chmod(0o755)
                        env = dict(os.environ)
                        env.update(
                            {
                                'PATH': f'{root}{os.pathsep}{env["PATH"]}',
                                'SETUP_ENV': 'false',
                                'REPO_NAME': 'kubeflow/pipelines',
                                'PULL_NUMBER': '123',
                                'JUNIT_XML': f'sdk-{label}.xml',
                                'PYTEST_LOG': f'sdk-{label}.log',
                                'TEST_FAIL_CALL': str(fail_call),
                            }
                        )
                        result = subprocess.run(
                            [
                                'bash',
                                '-e',
                                str(
                                    ROOT
                                    / f'test/presubmit-tests-sdk{suffix}.sh'
                                ),
                            ],
                            cwd=root,
                            env=env,
                            text=True,
                            capture_output=True,
                        )
                        self.assertEqual(
                            result.returncode, exit_code, result.stderr
                        )
                        calls = json.loads((root / 'args.json').read_text())
                        count = fail_call or (1 if suffix else 2)
                        self.assertEqual(len(calls), count)
                        for index, args in enumerate(calls):
                            self.assertIn('--durations=30', args)
                            self.assertIn('--durations-min=0.1', args)
                            report_label = label + ('-runtime' if index else '')
                            self.assertIn(
                                f'--junitxml=sdk-{report_label}.xml', args
                            )
                            self.assertTrue(
                                (root / f'sdk-{report_label}.xml').is_file()
                            )
                            report = (
                                root / f'sdk-{report_label}.log'
                            ).read_text()
                            self.assertIn('timing sentinel', report)
                            self.assertIn('stderr sentinel', report)
                        self.assertIn('timing sentinel', result.stdout)


if __name__ == '__main__':
    unittest.main()
