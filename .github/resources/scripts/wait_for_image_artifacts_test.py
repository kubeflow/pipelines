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
from unittest import mock

SCRIPT = Path(__file__).with_name('wait-for-image-artifacts.sh')
ARTIFACTS = (
    'apiserver',
    'scheduledworkflow',
    'persistenceagent',
    'frontend',
    'viewer-crd-controller',
    'visualization-server',
    'cache-deployer',
    'cache-server',
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
        producer_jobs: tuple[tuple[str, str, str], ...] = (),
        jobs_api_fails: bool = False,
        missing_artifact: str = 'frontend',
        publication_grace_attempts: Optional[int] = None,
        producer_state_unavailable_extensions: Optional[int] = None,
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
                    if [[ "$*" == *"/jobs?"* ]]; then
                      if [[ "$JOBS_API_FAILS" == "true" ]]; then
                        exit 1
                      fi
                      printf '%s\n' "$PRODUCER_JOBS"
                      exit 0
                    fi
                    count=$(cat "$GH_COUNTER")
                    count=$((count + 1))
                    echo "$count" > "$GH_COUNTER"
                    if (( count >= READY_AFTER )); then
                      printf '%s\n' $ARTIFACT_NAMES
                    else
                      for artifact in $ARTIFACT_NAMES; do
                        if [[ "$artifact" != "$MISSING_ARTIFACT" ]]; then
                          echo "$artifact"
                        fi
                      done
                    fi
                '''),
                encoding='utf-8',
            )
            fake_gh.chmod(0o755)

            environment = os.environ.copy()
            environment.pop('WAIT_ATTEMPTS', None)
            environment.update({
                'ARTIFACT_NAMES': ' '.join(ARTIFACTS),
                'GH_COUNTER': str(counter),
                'GITHUB_REPOSITORY': 'kubeflow/pipelines',
                'GITHUB_RUN_ID': '123',
                'JOBS_API_FAILS': str(jobs_api_fails).lower(),
                'MISSING_ARTIFACT': missing_artifact,
                'PATH': f'{fake_bin}{os.pathsep}{environment["PATH"]}',
                'PRODUCER_JOBS': '\n'.join(
                    '\t'.join(producer_job) for producer_job in producer_jobs),
                'READY_AFTER': str(ready_after),
                'WAIT_INTERVAL_SECONDS': '0',
            })
            if attempts is not None:
                environment['WAIT_ATTEMPTS'] = str(attempts)
            if publication_grace_attempts is not None:
                environment['PUBLICATION_GRACE_ATTEMPTS'] = str(
                    publication_grace_attempts)
            if producer_state_unavailable_extensions is not None:
                environment['PRODUCER_STATE_UNAVAILABLE_EXTENSIONS'] = str(
                    producer_state_unavailable_extensions)
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
        self.assertIn('All 11 branch image artifacts are available',
                      result.stdout)
        self.assertEqual(attempts, 1)

    def test_rejects_invalid_publication_grace(self):
        result, attempts = self._run(
            ready_after=1,
            publication_grace_attempts=0,
        )

        self.assertEqual(result.returncode, 2)
        self.assertIn('PUBLICATION_GRACE_ATTEMPTS must be a positive integer',
                      result.stderr)
        self.assertEqual(attempts, 0)

    def test_retries_until_artifacts_are_available(self):
        result, attempts = self._run(ready_after=2)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Waiting for branch image artifacts', result.stdout)
        self.assertEqual(attempts, 2)

    @mock.patch.dict(os.environ, {'WAIT_ATTEMPTS': '1'})
    def test_default_wait_exceeds_previous_ten_minute_budget(self):
        result, attempts = self._run(ready_after=21, attempts=None)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Waiting for branch image artifacts (20/40)',
                      result.stdout)
        self.assertEqual(attempts, 21)

    def test_fails_with_missing_artifact_names(self):
        result, attempts = self._run(ready_after=99, attempts=2)

        self.assertEqual(result.returncode, 1)
        self.assertIn('Missing branch image artifacts after producer completion grace',
                      result.stderr)
        self.assertIn('frontend', result.stderr)
        self.assertEqual(attempts, 5)

    def test_extends_wait_while_missing_producer_is_active(self):
        result, attempts = self._run(
            ready_after=3,
            attempts=2,
            producer_jobs=((
                'build / image-build (frontend, frontend/Dockerfile, .)',
                'in_progress',
                '',
            ),),
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(
            'Extending image artifact wait; active producers: frontend',
            result.stdout)
        self.assertEqual(attempts, 3)

    def test_does_not_extend_for_an_unrelated_active_producer(self):
        result, attempts = self._run(
            ready_after=99,
            attempts=2,
            producer_jobs=((
                'build / image-build (apiserver, backend/Dockerfile, .)',
                'in_progress',
                '',
            ),),
        )

        self.assertEqual(result.returncode, 1)
        self.assertNotIn('Extending image artifact wait', result.stdout)
        self.assertEqual(attempts, 5)

    def test_fails_immediately_when_producer_completed_unsuccessfully(self):
        result, attempts = self._run(
            ready_after=99,
            attempts=2,
            producer_jobs=((
                'build / image-build (frontend, frontend/Dockerfile, .)',
                'completed',
                'failure',
            ),),
        )

        self.assertEqual(result.returncode, 1)
        self.assertIn('frontend:failure', result.stderr)
        self.assertEqual(attempts, 2)

    def test_allows_publication_grace_after_successful_producer(self):
        result, attempts = self._run(
            ready_after=4,
            attempts=2,
            producer_jobs=((
                'build / image-build (frontend, frontend/Dockerfile, .)',
                'completed',
                'success',
            ),),
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Allowing 3 publication grace attempts', result.stdout)
        self.assertEqual(attempts, 4)

    def test_matches_runtime_base_image_producer(self):
        result, attempts = self._run(
            ready_after=3,
            attempts=2,
            missing_artifact='runtime-base-images',
            producer_jobs=((
                'build / runtime-base-images',
                'queued',
                '',
            ),),
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('active producers: runtime-base-images', result.stdout)
        self.assertEqual(attempts, 3)

    def test_extends_conservatively_when_producer_state_is_unavailable(self):
        result, attempts = self._run(
            ready_after=2,
            attempts=1,
            jobs_api_fails=True,
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(
            'Extending image artifact wait because producer state is unavailable',
            result.stdout)
        self.assertEqual(attempts, 2)

    def test_fails_after_bounded_producer_state_api_fallback(self):
        result, attempts = self._run(
            ready_after=99,
            attempts=1,
            jobs_api_fails=True,
            producer_state_unavailable_extensions=1,
        )

        self.assertEqual(result.returncode, 1)
        self.assertIn('producer state remains unavailable', result.stderr)
        self.assertIn('frontend', result.stderr)
        self.assertEqual(attempts, 2)


if __name__ == '__main__':
    unittest.main()
