# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise API generator acquisition without Docker or network access."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

REPO_ROOT = Path(__file__).resolve().parents[3]
REMOTE_IMAGE = 'ghcr.io/kubeflow/kfp-api-generator:master'


class ApiGeneratorPullRetryTest(unittest.TestCase):

    def run_make(self,
                 *targets,
                 cached=False,
                 pull_failures=0,
                 run_exit=0,
                 prebuilt=True,
                 image=REMOTE_IMAGE):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            docker = root / 'docker'
            docker.write_text('''#!/usr/bin/env bash
set -eu
printf '%s\\n' "$*" >> "$TEST_DOCKER_LOG"
case "$1" in
  image)
    test "$2" = inspect
    test "$TEST_IMAGE_CACHED" = true
    ;;
  pull)
    count=0
    if test -f "$TEST_PULL_COUNT"; then
      read -r count < "$TEST_PULL_COUNT"
    fi
    count=$((count + 1))
    printf '%s\\n' "$count" > "$TEST_PULL_COUNT"
    if test "$count" -le "$TEST_PULL_FAILURES"; then
      echo '503 Service Unavailable' >&2
      exit 1
    fi
    ;;
  run) exit "$TEST_RUN_EXIT" ;;
  *) echo "Unexpected Docker command: $*" >&2; exit 99 ;;
esac
''')
            docker.chmod(0o755)
            sleep = root / 'sleep'
            sleep.write_text('''#!/usr/bin/env bash
printf '%s\\n' "$*" >> "$TEST_SLEEP_LOG"
''')
            sleep.chmod(0o755)
            docker_log = root / 'docker.log'
            sleep_log = root / 'sleep.log'
            environment = dict(os.environ)
            # Keep a caller's make flags from changing this isolated invocation.
            for variable in ('MAKEFLAGS', 'MFLAGS', 'MAKEOVERRIDES'):
                environment.pop(variable, None)
            environment.update(
                PATH=f'{root}{os.pathsep}{environment["PATH"]}',
                TEST_DOCKER_LOG=str(docker_log),
                TEST_SLEEP_LOG=str(sleep_log),
                TEST_PULL_COUNT=str(root / 'pull-count'),
                TEST_IMAGE_CACHED=str(cached).lower(),
                TEST_PULL_FAILURES=str(pull_failures),
                TEST_RUN_EXIT=str(run_exit),
            )
            result = subprocess.run([
                'make', '--no-print-directory', '-o', 'fetch-protos', '-o',
                'ensure-api-generator-image', *targets,
                'MAKE=make --no-print-directory -o fetch-protos -o ensure-api-generator-image',
                f'USE_PREBUILT_IMAGE={str(prebuilt).lower()}',
                f'PREBUILT_REMOTE_IMAGE={image}'
            ],
                                    cwd=REPO_ROOT / 'api',
                                    env=environment,
                                    text=True,
                                    capture_output=True,
                                    timeout=15)
            commands = docker_log.read_text().splitlines() if docker_log.exists(
            ) else []
            sleeps = sleep_log.read_text().splitlines() if sleep_log.exists(
            ) else []
            return result, commands, sleeps

    def test_transient_pull_failure_recovers_for_both_generators(self):
        for target in ('golang', 'python'):
            with self.subTest(target=target):
                result, commands, sleeps = self.run_make(
                    target, pull_failures=2)
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)
                self.assertEqual(commands[:4],
                                 [f'image inspect {REMOTE_IMAGE}'] +
                                 [f'pull {REMOTE_IMAGE}'] * 3)
                self.assertEqual(len(commands), 5)
                self.assertTrue(commands[-1].startswith('run '))
                self.assertIn('--pull=never', commands[-1])
                self.assertIn(REMOTE_IMAGE, commands[-1])
                self.assertEqual(sleeps, ['20', '20'])

    def test_exhausted_pull_never_runs_generator(self):
        for target in ('golang', 'python'):
            with self.subTest(target=target):
                result, commands, sleeps = self.run_make(
                    target, pull_failures=3)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(commands, [f'image inspect {REMOTE_IMAGE}'] +
                                 [f'pull {REMOTE_IMAGE}'] * 3)
                self.assertEqual(sleeps, ['20', '20'])
                self.assertIn('503 Service Unavailable', result.stderr)

    def test_cached_image_does_not_contact_registry(self):
        for target in ('golang', 'python'):
            with self.subTest(target=target):
                result, commands, sleeps = self.run_make(target, cached=True)
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)
                self.assertEqual(commands[0], f'image inspect {REMOTE_IMAGE}')
                self.assertEqual(len(commands), 2)
                self.assertIn('--pull=never', commands[1])
                self.assertEqual(sleeps, [])

    def test_generator_failure_is_not_retried(self):
        for target in ('golang', 'python'):
            with self.subTest(target=target):
                result, commands, sleeps = self.run_make(target, run_exit=42)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(
                    len([c for c in commands if c.startswith('run ')]), 1)
                self.assertEqual(commands.count(f'pull {REMOTE_IMAGE}'), 1)
                self.assertEqual(sleeps, [])

    def test_shared_prerequisite_pulls_once_for_all_target(self):
        result, commands, sleeps = self.run_make('all')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(commands.count(f'image inspect {REMOTE_IMAGE}'), 1)
        self.assertEqual(commands.count(f'pull {REMOTE_IMAGE}'), 1)
        self.assertEqual(len([c for c in commands if c.startswith('run ')]), 2)
        self.assertEqual(sleeps, [])

    def test_explicit_image_is_used_for_inspect_pull_and_run(self):
        image = 'example.com/generator@sha256:' + 'a' * 64
        result, commands, _ = self.run_make('golang', image=image)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertEqual(commands[:2],
                         [f'image inspect {image}', f'pull {image}'])
        self.assertIn(image, commands[-1])
        self.assertNotIn(REMOTE_IMAGE, '\n'.join(commands))

    def test_source_build_path_does_not_acquire_remote_image(self):
        for target in ('golang', 'python'):
            with self.subTest(target=target):
                result, commands, sleeps = self.run_make(target, prebuilt=False)
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)
                self.assertEqual(len(commands), 1)
                self.assertTrue(commands[0].startswith('run '))
                self.assertIn(' kfp-api-generator ', commands[0])
                self.assertNotIn(REMOTE_IMAGE, commands[0])
                self.assertEqual(sleeps, [])


if __name__ == '__main__':
    unittest.main()
