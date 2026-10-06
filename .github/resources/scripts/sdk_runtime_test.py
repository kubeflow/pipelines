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
"""Regression coverage for the SDK presubmit's runtime artifact selection."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

SCRIPT = Path(__file__).resolve().parents[3] / 'test/presubmit-tests-sdk.sh'


class SdkRuntimeTest(unittest.TestCase):

    def setUp(self) -> None:
        """Isolate the presubmit from actual builds and Docker-backed tests."""
        directory = tempfile.TemporaryDirectory(prefix="SDK wheel's ")
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.fake_bin = self.root / 'bin'
        self.fake_bin.mkdir()
        self.log = self.root / 'commands.jsonl'
        uv = self.fake_bin / 'uv'
        uv.write_text(f'#!{sys.executable}\n' + textwrap.dedent('''\
            import json
            import os
            from pathlib import Path
            import sys

            args = sys.argv[1:]
            phase = ('build' if args[0] == 'build' else
                     'runtime' if 'sdk/python/test/runtime' in args else 'suite')
            with open(os.environ['COMMAND_LOG'], 'a') as log:
                log.write(json.dumps({
                    'phase': phase,
                    'args': args,
                    'package': os.environ.get('KFP_PACKAGE_PATH'),
                }) + '\\n')
            if phase == os.environ['FAIL_PHASE']:
                sys.exit(23)
            if phase == 'build':
                output = Path(args[args.index('--out-dir') + 1])
                for index in range(int(os.environ['WHEEL_COUNT'])):
                    (output / f'kfp-{index}-py3-none-any.whl').touch()
        '''))
        uv.chmod(0o755)

    def run_presubmit(
        self,
        failure: str = '',
        wheel_count: int = 1,
        pull_number: str = '',
    ) -> subprocess.CompletedProcess[str]:
        """Run the entrypoint and verify its temporary build is removed."""
        self.log.write_text('')
        result = subprocess.run(
            ['bash', str(SCRIPT)],
            cwd=self.root,
            env={
                **os.environ,
                'PATH':
                    f'{self.fake_bin}{os.pathsep}{os.environ["PATH"]}',
                'COMMAND_LOG':
                    str(self.log),
                'FAIL_PHASE':
                    failure,
                'WHEEL_COUNT':
                    str(wheel_count),
                'TMPDIR':
                    str(self.root),
                'SETUP_ENV':
                    'false',
                'REPO_NAME':
                    'kubeflow/pipelines',
                'PULL_NUMBER':
                    pull_number,
                'PYTEST_PARALLEL_WORKERS':
                    '2',
            },
            capture_output=True,
            text=True,
        )
        self.commands = [
            json.loads(line) for line in self.log.read_text().splitlines()
        ]
        build = self.commands[0]['args']
        self.assertEqual(build[:4], ['build', '--package', 'kfp', '--wheel'])
        self.output = Path(build[build.index('--out-dir') + 1])
        self.assertFalse(self.output.exists())
        return result

    def test_runtime_wheel_preserves_docker_source_and_coverage(self) -> None:
        """Keep both test phases on PRs and pushes to master."""
        for pull_number in ('', '42'):
            with self.subTest(pull_number=pull_number):
                result = self.run_presubmit(pull_number=pull_number)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(
                    [command['phase'] for command in self.commands],
                    ['build', 'suite', 'runtime'])
                source = 'git+https://github.com/kubeflow/pipelines'
                if pull_number:
                    source += f'@refs/pull/{pull_number}/merge'
                source += '#egg=kfp&subdirectory=sdk/python'
                self.assertEqual(self.commands[1]['package'], source)
                self.assertIn('--ignore=sdk/python/test/runtime',
                              self.commands[1]['args'])
                self.assertNotIn('--cov-append', self.commands[1]['args'])
                self.assertEqual(self.commands[2]['package'],
                                 str(self.output / 'kfp-0-py3-none-any.whl'))
                self.assertIn('--cov-append', self.commands[2]['args'])
                for command in self.commands[1:]:
                    self.assertEqual(command['args'][:3],
                                     ['run', 'python', '-m'])
                    self.assertIn('--cov=kfp', command['args'])
                    self.assertIn('regression', command['args'])
                    self.assertNotIn('-s', command['args'])
                    self.assertEqual(command['args'][-2:], ['-n', '2'])

    def test_build_and_test_failures_stop_the_presubmit(self) -> None:
        """Propagate each failed phase and remove its build directory."""
        for failure, expected_phases in (
            ('build', ['build']),
            ('suite', ['build', 'suite']),
            ('runtime', ['build', 'suite', 'runtime']),
        ):
            with self.subTest(failure=failure):
                result = self.run_presubmit(failure=failure)
                self.assertEqual(result.returncode, 23, result.stderr)
                self.assertEqual(
                    [command['phase'] for command in self.commands],
                    expected_phases)

    def test_missing_or_ambiguous_wheels_fail_before_testing(self) -> None:
        """Do not fall back to a published package after a bad build."""
        for wheel_count in (0, 2):
            with self.subTest(wheel_count=wheel_count):
                result = self.run_presubmit(wheel_count=wheel_count)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertEqual(
                    [command['phase'] for command in self.commands], ['build'])
                self.assertIn('exactly one freshly built kfp wheel',
                              result.stderr)


if __name__ == '__main__':
    unittest.main()
