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
        artifacts_api_fails_on: tuple[int, ...] = (),
        use_callback: bool = False,
        callback_exit: int = 0,
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
                    if [[ " $ARTIFACTS_API_FAILS_ON " == *" $count "* ]]; then
                      exit 1
                    fi
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
                'ARTIFACTS_API_FAILS_ON':
                    ' '.join(
                        str(attempt) for attempt in artifacts_api_fails_on),
                'ARTIFACT_NAMES':
                    ' '.join(ARTIFACTS),
                'GH_COUNTER':
                    str(counter),
                'GITHUB_REPOSITORY':
                    'kubeflow/pipelines',
                'GITHUB_RUN_ID':
                    '123',
                'JOBS_API_FAILS':
                    str(jobs_api_fails).lower(),
                'MISSING_ARTIFACT':
                    missing_artifact,
                'PATH':
                    f'{fake_bin}{os.pathsep}{environment["PATH"]}',
                'PRODUCER_JOBS':
                    '\n'.join('\t'.join(producer_job)
                              for producer_job in producer_jobs),
                'READY_AFTER':
                    str(ready_after),
                'WAIT_INTERVAL_SECONDS':
                    '0',
            })
            if attempts is not None:
                environment['WAIT_ATTEMPTS'] = str(attempts)
            if publication_grace_attempts is not None:
                environment['PUBLICATION_GRACE_ATTEMPTS'] = str(
                    publication_grace_attempts)
            if producer_state_unavailable_extensions is not None:
                environment['PRODUCER_STATE_UNAVAILABLE_EXTENSIONS'] = str(
                    producer_state_unavailable_extensions)
            command = ['bash', str(SCRIPT)]
            if use_callback:
                callback = root / 'callback'
                callback.write_text('#!/usr/bin/env bash\n'
                                    'echo "CALLBACK $(cat "$GH_COUNTER") $*"\n'
                                    f'exit {callback_exit}\n')
                callback.chmod(0o755)
                command.append(str(callback))
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                check=False,
                env=environment,
            )
            return result, int(counter.read_text(encoding='utf-8'))

    def test_prepares_visible_artifacts_before_last_producer_finishes(self):
        result, attempts = self._run(
            ready_after=3, use_callback=True, attempts=4)
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = [
            line.split()
            for line in result.stdout.splitlines()
            if line.startswith('CALLBACK ')
        ]
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[0][1], '1')
        self.assertCountEqual(
            calls[0][2:], [name for name in ARTIFACTS if name != 'frontend'])
        self.assertEqual(calls[1], ['CALLBACK', '3', 'frontend'])
        self.assertEqual(attempts, 3)

    def test_callback_failure_stops_before_readiness_success(self):
        for ready_after in (1, 3):
            with self.subTest(ready_after=ready_after):
                result, attempts = self._run(
                    ready_after=ready_after, use_callback=True, callback_exit=7)
                self.assertEqual(result.returncode, 7)
                self.assertEqual(attempts, 1)
                self.assertNotIn('All 8 branch image artifacts are available',
                                 result.stdout)

    def test_listing_failure_does_not_repeat_already_prepared_artifacts(self):
        result, attempts = self._run(
            ready_after=3,
            attempts=4,
            use_callback=True,
            artifacts_api_fails_on=(2,))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('Could not list image artifacts on attempt 2/4',
                      result.stdout)
        calls = [
            line.split()
            for line in result.stdout.splitlines()
            if line.startswith('CALLBACK ')
        ]
        self.assertEqual(len(calls), 2)
        self.assertCountEqual(
            calls[0][2:], [name for name in ARTIFACTS if name != 'frontend'])
        self.assertEqual(calls[1], ['CALLBACK', '3', 'frontend'])
        self.assertEqual(attempts, 3)

    def test_partial_import_does_not_hide_missing_producer_failure(self):
        result, attempts = self._run(
            ready_after=99,
            use_callback=True,
            producer_jobs=(('build / image-build (frontend, frontend)',
                            'completed', 'failure'),))
        self.assertEqual(result.returncode, 1)
        self.assertEqual(attempts, 2)
        self.assertEqual(result.stdout.count('CALLBACK '), 1)
        self.assertIn('frontend:failure', result.stderr)

    def test_succeeds_when_all_artifacts_are_available(self):
        result, attempts = self._run(ready_after=1)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('All 8 branch image artifacts are available',
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
        self.assertIn(
            'Missing branch image artifacts after producer completion grace',
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
