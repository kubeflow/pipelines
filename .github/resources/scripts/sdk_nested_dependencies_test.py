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
"""Keep unpublished companion wheels available to SDK child environments."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]


class SdkNestedDependenciesTest(unittest.TestCase):

    def run_script(self,
                   setup: bool = False,
                   wheel_exit: int = 0,
                   pytest_exit: int = 0) -> tuple:
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            capture = directory / 'calls'
            (directory / 'api').mkdir()
            script = (
                f'#!{sys.executable}\n'
                'import json, os, pathlib, sys\n'
                'args = sys.argv[1:]\n'
                'with open(os.environ["CAPTURE"], "a") as f:\n'
                '    f.write(json.dumps({"args": args, "links": os.getenv("PIP_FIND_LINKS", "")}) + "\\n")\n'
                'if args[:2] == ["-m", "venv"]:\n'
                '    pathlib.Path("venv/bin").mkdir(parents=True)\n'
                '    pathlib.Path("venv/bin/activate").write_text("deactivate() { :; }\\n")\n'
                'if args[:3] == ["-m", "pip", "wheel"]: sys.exit(int(os.environ["WHEEL_EXIT"]))\n'
                'if args[:2] == ["-m", "pytest"]: sys.exit(int(os.environ["PYTEST_EXIT"]))\n'
            )
            for name in ('python', 'python3', 'make'):
                executable = directory / name
                executable.write_text(script)
                executable.chmod(0o755)
            result = subprocess.run(
                ['bash', '-e',
                 str(ROOT / 'test/presubmit-tests-sdk-unit.sh')],
                cwd=directory,
                env={
                    **os.environ,
                    'PATH':
                        f'{directory}{os.pathsep}{os.environ["PATH"]}',
                    'SETUP_ENV':
                        str(setup).lower(),
                    'REPO_NAME':
                        'kubeflow/pipelines',
                    'PULL_NUMBER':
                        '15618',
                    'PYTEST_LOG':
                        str(directory / 'pytest.log'),
                    'CAPTURE':
                        str(capture),
                    'WHEEL_EXIT':
                        str(wheel_exit),
                    'PYTEST_EXIT':
                        str(pytest_exit),
                    'PIP_FIND_LINKS':
                        'https://example.invalid/existing',
                },
                capture_output=True,
                text=True)
            calls = [
                json.loads(line) for line in capture.read_text().splitlines()
            ]
            return result, calls, directory.resolve()

    def test_companion_wheels_are_built_before_tests_and_inherited(
            self) -> None:
        result, calls, root = self.run_script()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls[0]['args'][:4],
                         ['-m', 'pip', 'wheel', '--no-deps'])
        self.assertIn(str(root / 'api/v2alpha1/python'), calls[0]['args'])
        self.assertIn(
            str(root / 'backend/api/v2beta1/python_http_client'),
            calls[0]['args'])
        wheels = calls[0]['args'][calls[0]['args'].index('--wheel-dir') + 1]
        self.assertEqual(calls[-1]['args'][:2], ['-m', 'pytest'])
        self.assertEqual(calls[-1]['links'],
                         wheels + ' https://example.invalid/existing')
        self.assertFalse(
            Path(wheels).exists(), 'Temporary wheels survive suite completion')

    def test_setup_generates_and_builds_companions_before_sdk_dependencies(
            self) -> None:
        result, calls, _ = self.run_script(setup=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        args = [call['args'] for call in calls]
        generation = args.index(['clean', 'python'])
        wheel = next(i for i, values in enumerate(args)
                     if values[:3] == ['-m', 'pip', 'wheel'])
        requirements = args.index(
            ['-m', 'pip', 'install', '-r', 'sdk/python/requirements.txt'])
        sdk = args.index(['-m', 'pip', 'install', 'sdk/python'])
        self.assertLess(generation, wheel)
        self.assertLess(wheel, requirements)
        self.assertLess(wheel, sdk)
        self.assertIn('kfp-sdk-wheels.', calls[requirements]['links'])
        self.assertIn('kfp-sdk-wheels.', calls[sdk]['links'])

    def test_failed_pytest_preserves_exit_and_cleans_up(self) -> None:
        result, calls, _ = self.run_script(pytest_exit=23)
        self.assertEqual(result.returncode, 23, result.stderr)
        wheels = calls[0]['args'][calls[0]['args'].index('--wheel-dir') + 1]
        self.assertFalse(Path(wheels).exists())

    def test_failed_wheel_build_stops_tests_and_cleans_up(self) -> None:
        result, calls, _ = self.run_script(wheel_exit=17)
        self.assertEqual(result.returncode, 17, result.stderr)
        self.assertEqual(len(calls), 1)
        wheels = calls[0]['args'][calls[0]['args'].index('--wheel-dir') + 1]
        self.assertFalse(Path(wheels).exists())


if __name__ == '__main__':
    unittest.main()
