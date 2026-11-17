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
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]


class SdkTimingReportsTest(unittest.TestCase):

    def test_reports_survive_test_failure(self):
        for suffix, label in (('', 'regression'), ('-unit', 'unit')):
            for exit_code in (0, 42):
                with self.subTest(suite=label, exit_code=exit_code):
                    with tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        executable = root / 'python'
                        executable.write_text(
                            '#!/usr/bin/env python3\n'
                            'import json, os, pathlib, sys\n'
                            'if sys.argv[1:4] == ["-m", "pip", "wheel"]:\n'
                            '    output = pathlib.Path(sys.argv[sys.argv.index("--wheel-dir") + 1])\n'
                            '    (output / "kfp-test.whl").touch()\n'
                            '    sys.exit(0)\n'
                            'if sys.argv[1:3] != ["-m", "pytest"]: sys.exit(0)\n'
                            'pathlib.Path("args.json").write_text(json.dumps(sys.argv[1:]))\n'
                            'print("slowest durations: timing sentinel", flush=True)\n'
                            'print("stderr sentinel", file=sys.stderr, flush=True)\n'
                            'for arg in sys.argv:\n'
                            '    if arg.startswith("--junitxml="):\n'
                            '        pathlib.Path(arg.split("=", 1)[1]).write_text("<testsuites/>")\n'
                            'sys.exit(int(os.environ["TEST_EXIT_CODE"]))\n')
                        executable.chmod(0o755)
                        env = dict(os.environ)
                        env.update({
                            'PATH': f'{root}{os.pathsep}{env["PATH"]}',
                            'SETUP_ENV': 'false',
                            'REPO_NAME': 'kubeflow/pipelines',
                            'PULL_NUMBER': '123',
                            'JUNIT_XML': f'sdk-{label}.xml',
                            'PYTEST_LOG': f'sdk-{label}.log',
                            'TEST_EXIT_CODE': str(exit_code),
                        })
                        result = subprocess.run([
                            'bash', '-e',
                            str(ROOT / f'test/presubmit-tests-sdk{suffix}.sh')
                        ],
                                                cwd=root,
                                                env=env,
                                                text=True,
                                                capture_output=True)
                        self.assertEqual(result.returncode, exit_code,
                                         result.stderr)
                        args = json.loads((root / 'args.json').read_text())
                        self.assertIn('--durations=30', args)
                        self.assertIn('--durations-min=0.1', args)
                        self.assertTrue((root / f'sdk-{label}.xml').is_file())
                        report = (root / f'sdk-{label}.log').read_text()
                        self.assertIn('timing sentinel', report)
                        self.assertIn('stderr sentinel', report)
                        self.assertIn('timing sentinel', result.stdout)


if __name__ == '__main__':
    unittest.main()
