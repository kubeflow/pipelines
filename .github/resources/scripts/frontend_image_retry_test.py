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
"""Verify frontend image timeouts leave room for recovery and artifact
upload."""

import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/image-builds.yml'
BUILDER_OUTPUT = '${{ steps.buildx.outputs.builder_name }}'
FAILED_ATTEMPT = "${{ steps.save-image.outcome == 'failure' }}"


class FrontendImageRetryTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.job = yaml.safe_load(WORKFLOW.read_text())['jobs']['image-build']
        cls.steps = cls.job['steps']
        cls.first = next(s for s in cls.steps if s.get('id') == 'save-image')
        cls.retry = next(s for s in cls.steps if s.get('id') == 'rebuild')
        cls.reset = next(s for s in cls.steps
                         if s['name'] == 'Reset frontend builder before retry')
        cls.upload = next(
            s for s in cls.steps if s['name'] == 'Upload artifact')

    def test_both_attempts_leave_time_for_reset_setup_and_upload(self):
        frontend_timeouts = []
        for step in (self.first, self.retry):
            # This is deliberately the frontend-specific policy, not a general
            # GitHub expression evaluator. Other image budgets must not shrink.
            match = re.fullmatch(
                r"\$\{\{\s*matrix\.image == 'frontend' && (\d+) \|\| (\d+)\s*\}\}",
                str(step.get('timeout-minutes')))
            self.assertIsNotNone(match, step)
            frontend_timeout, other_timeout = map(int, match.groups())
            self.assertGreater(frontend_timeout, 0)
            self.assertEqual(other_timeout, self.job['timeout-minutes'])
            frontend_timeouts.append(frontend_timeout)

        wait = next(
            s for s in self.steps if s['name'] == 'Wait before rebuilding')
        wait_command = re.fullmatch(r'sleep (\d+)', wait['run'].strip())
        self.assertIsNotNone(wait_command)
        reset_timeout = self.reset['timeout-minutes']
        self.assertGreater(reset_timeout, 0)
        recovery_minutes = (
            sum(frontend_timeouts) + reset_timeout +
            int(wait_command.group(1)) / 60)
        self.assertLessEqual(recovery_minutes + 5, self.job['timeout-minutes'])
        self.assertEqual(self.job['timeout-minutes'], 30)

    def test_timeout_uses_existing_failure_retry_without_swallowing_retry(self):
        self.assertTrue(self.first['continue-on-error'])
        self.assertEqual(self.retry['if'], FAILED_ATTEMPT)
        self.assertFalse(self.retry.get('continue-on-error', False))
        self.assertTrue(self.retry['with']['no-cache'])
        self.assertFalse(self.job['strategy']['fail-fast'])
        self.assertEqual(
            self.reset['if'],
            "${{ matrix.image == 'frontend' && steps.save-image.outcome == 'failure' }}"
        )
        self.assertFalse(self.reset.get('continue-on-error', False))
        self.assertLess(
            self.steps.index(self.first), self.steps.index(self.reset))
        self.assertLess(
            self.steps.index(self.reset), self.steps.index(self.retry))

    def test_initial_reset_and_retry_use_the_same_explicit_builder(self):
        setup = next(s for s in self.steps if s.get('id') == 'buildx')
        self.assertIn('setup-buildx-with-retry.sh', setup['run'])
        self.assertEqual(self.reset['env']['BUILDER_NAME'], BUILDER_OUTPUT)
        for step in (self.first, self.retry):
            self.assertEqual(step['with']['builder'], BUILDER_OUTPUT)
        self.assertLess(self.steps.index(setup), self.steps.index(self.first))

    def test_only_a_successful_build_can_publish_the_image(self):
        self.assertEqual(
            self.upload['if'],
            "${{ steps.save-image.outcome == 'success' || steps.rebuild.outcome == 'success' }}"
        )
        self.assertEqual(self.upload['with']['if-no-files-found'], 'error')
        self.assertLess(
            self.steps.index(self.retry), self.steps.index(self.upload))

    def _run_reset(self, inspect_fails=False, remove_fails=False):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            commands = directory / 'commands.log'
            output = directory / 'github-output'
            scripts = {
                'docker':
                    '''#!/usr/bin/env bash
printf 'docker %s\\n' "$*" >> "$COMMAND_LOG"
# The diagnostics command fails, as it can when the builder is unhealthy.
[[ "$1" == logs ]] && exit 1
if [[ "$1 $2" == "buildx rm" && "$REMOVE_FAILS" == true ]]; then
  exit 1
fi
if [[ "$1 $2" == "buildx inspect" && "$INSPECT_FAILS" == true ]]; then
  exit 1
fi
exit 0
''',
                'timeout':
                    '''#!/usr/bin/env bash
printf 'timeout %s\\n' "$*" >> "$COMMAND_LOG"
shift
exec "$@"
''',
                'sleep':
                    '#!/usr/bin/env bash\nexit 0\n',
            }
            for name, script in scripts.items():
                executable = directory / name
                executable.write_text(script)
                executable.chmod(0o755)
            result = subprocess.run(
                [
                    'bash', '--noprofile', '--norc', '-eo', 'pipefail', '-c',
                    self.reset['run']
                ],
                cwd=ROOT,
                env={
                    **os.environ,
                    'PATH':
                        f'{directory}:{os.environ["PATH"]}',
                    'BUILDER_NAME':
                        'frontend-retry-test',
                    'COMMAND_LOG':
                        str(commands),
                    'GITHUB_OUTPUT':
                        str(output),
                    'INSPECT_FAILS':
                        str(inspect_fails).lower(),
                    'REMOVE_FAILS':
                        str(remove_fails).lower(),
                    'BUILDKIT_IMAGE':
                        'moby/buildkit:buildx-stable-1',
                    'BUILDKIT_MIRROR_IMAGE':
                        'mirror.gcr.io/moby/buildkit:buildx-stable-1',
                },
                capture_output=True,
                text=True,
                check=False,
                timeout=10)
            return (result, commands.read_text().splitlines(),
                    output.read_text() if output.exists() else '')

    def test_failed_diagnostics_do_not_prevent_removing_and_recreating_builder(
            self):
        result, commands, output = self._run_reset()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            commands[0],
            'timeout 15s docker logs --tail 200 buildx_buildkit_frontend-retry-test0'
        )
        remove = 'docker buildx rm --force frontend-retry-test'
        self.assertIn('timeout 30s ' + remove, commands)
        first_pull = next(c for c in commands if c.startswith('docker pull '))
        self.assertLess(commands.index(remove), commands.index(first_pull))
        create = next(
            c for c in commands if c.startswith('docker buildx create '))
        self.assertIn('--name frontend-retry-test ', create)
        self.assertLess(commands.index(remove), commands.index(create))
        self.assertIn('docker buildx inspect frontend-retry-test --bootstrap',
                      commands)
        self.assertEqual(output, 'builder_name=frontend-retry-test\n')

    def test_required_removal_failure_stops_before_fresh_setup(self):
        result, commands, output = self._run_reset(remove_fails=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(commands, [
            'timeout 15s docker logs --tail 200 buildx_buildkit_frontend-retry-test0',
            'docker logs --tail 200 buildx_buildkit_frontend-retry-test0',
            'timeout 30s docker buildx rm --force frontend-retry-test',
            'docker buildx rm --force frontend-retry-test',
        ])
        self.assertEqual(output, '')

    def test_reset_failure_is_not_hidden_by_optional_diagnostics(self):
        result, commands, output = self._run_reset(inspect_fails=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(
            commands.count(
                'docker buildx inspect frontend-retry-test --bootstrap'), 3)
        self.assertEqual(output, '')


if __name__ == '__main__':
    unittest.main()
