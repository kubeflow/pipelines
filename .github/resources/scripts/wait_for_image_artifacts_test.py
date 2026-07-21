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

import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
from typing import Optional
import unittest

SCRIPT = Path(__file__).with_name('wait-for-image-artifacts.sh')
ARTIFACTS = (
    'apiserver',
    'scheduledworkflow',
    'persistenceagent',
    'frontend',
    'metadata-writer',
    'viewer-crd-controller',
    'visualization-server',
    'cache-deployer',
    'cache-server',
    'metadata-envoy',
    'driver',
    'launcher',
    'runtime-base-images',
)


class WaitForImageArtifactsTest(unittest.TestCase):

    def _run(
        self,
        *,
        ready_after: int,
        attempts: Optional[int] = 2,
    ) -> tuple[subprocess.CompletedProcess[str], int]:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            fake_bin = root / 'bin'
            fake_bin.mkdir()
            counter = root / 'counter'
            counter.write_text('0', encoding='utf-8')
            fake_gh = fake_bin / 'gh'
            fake_gh.write_text(
                textwrap.dedent('''\
                    #!/usr/bin/env bash
                    count=$(cat "$GH_COUNTER")
                    count=$((count + 1))
                    echo "$count" > "$GH_COUNTER"
                    if (( count >= READY_AFTER )); then
                      printf '%s\n' $ARTIFACT_NAMES
                    else
                      echo apiserver
                    fi
                '''),
                encoding='utf-8',
            )
            fake_gh.chmod(0o755)

            environment = os.environ.copy()
            environment.update({
                'ARTIFACT_NAMES': ' '.join(ARTIFACTS),
                'GH_COUNTER': str(counter),
                'GITHUB_REPOSITORY': 'kubeflow/pipelines',
                'GITHUB_RUN_ID': '123',
                'PATH': f'{fake_bin}{os.pathsep}{environment["PATH"]}',
                'READY_AFTER': str(ready_after),
                'WAIT_INTERVAL_SECONDS': '0',
            })
            if attempts is not None:
                environment['WAIT_ATTEMPTS'] = str(attempts)
            result = subprocess.run(
                ['bash', str(SCRIPT)],
                capture_output=True,
                text=True,
                check=False,
                env=environment,
            )
            return result, int(counter.read_text(encoding='utf-8'))

    def test_succeeds_when_all_artifacts_are_available(self):
        result, attempts = self._run(ready_after=1)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('All 13 branch image artifacts are available',
                      result.stdout)
        self.assertEqual(attempts, 1)

    def test_retries_until_artifacts_are_available(self):
        result, attempts = self._run(ready_after=2)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Waiting for branch image artifacts', result.stdout)
        self.assertEqual(attempts, 2)

    def test_default_wait_exceeds_previous_ten_minute_budget(self):
        result, attempts = self._run(ready_after=21, attempts=None)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Waiting for branch image artifacts (20/40)',
                      result.stdout)
        self.assertEqual(attempts, 21)

    def test_fails_with_missing_artifact_names(self):
        result, attempts = self._run(ready_after=3, attempts=2)

        self.assertEqual(result.returncode, 1)
        self.assertIn('Missing branch image artifacts after 2 attempts',
                      result.stderr)
        self.assertIn('runtime-base-images', result.stderr)
        self.assertEqual(attempts, 2)


if __name__ == '__main__':
    unittest.main()
