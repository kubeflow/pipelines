# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Ensure nested SDK installs test queue candidates rather than master."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class QueueTestSourcesTest(unittest.TestCase):
    def test_sdk_install_sources(self):
        sha = 'a' * 40
        for script in ['presubmit-tests-sdk.sh', 'presubmit-tests-sdk-unit.sh']:
            for source, pull, expected in [
                (sha, '', '@' + sha),
                (sha, '7', '@' + sha),
                ('', '7', '@refs/pull/7/merge'),
                ('', '', ''),
            ]:
                with self.subTest(script=script, source=source, pull=pull):
                    with tempfile.TemporaryDirectory() as directory:
                        uv = Path(directory) / 'uv'
                        uv.write_text(
                            '#!/bin/sh\n'
                            'if [ "$1" = build ]; then\n'
                            '  while [ "$1" != --out-dir ]; do shift; done\n'
                            '  touch "$2/kfp-test.whl"\n'
                            'else\n'
                            '  printf "%s\\n" "$KFP_PACKAGE_PATH"\n'
                            'fi\n'
                        )
                        uv.chmod(0o755)
                        result = subprocess.run(
                            ['bash', str(ROOT / 'test' / script)],
                            env={
                                **os.environ,
                                'PATH': directory + ':' + os.environ['PATH'],
                                'SETUP_ENV': 'false',
                                'TEST_SOURCE_REF': source,
                                'PULL_NUMBER': pull,
                                'REPO_NAME': 'kubeflow/pipelines',
                            },
                            cwd=ROOT,
                            capture_output=True,
                            text=True,
                            check=True,
                        )
                        self.assertEqual(
                            result.stdout.splitlines()[0],
                            'git+https://github.com/kubeflow/pipelines'
                            + expected
                            + '#egg=kfp&subdirectory=sdk/python',
                        )
                        if script == 'presubmit-tests-sdk.sh':
                            self.assertTrue(
                                result.stdout.splitlines()[1].endswith(
                                    '/kfp-test.whl'
                                )
                            )

    def test_go_suite_receives_queue_source(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text()
        )
        step = next(
            step
            for step in action['runs']['steps']
            if step.get('id') == 'run-tests'
        )
        self.assertEqual(
            step['env']['TEST_SOURCE_REF'],
            "${{ github.event.merge_group.head_sha || 'master' }}",
        )
        self.assertIn('-branchName="$TEST_SOURCE_REF"', step['run'])


if __name__ == '__main__':
    unittest.main()
