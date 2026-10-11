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

import json
import os
from pathlib import Path
import subprocess
import sys
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


def artifact_record(name,
                    *,
                    artifact_id=None,
                    created_at='2026-10-11T12:01:00Z',
                    expired=False):
    return {
        'name':
            name,
        'id':
            artifact_id if artifact_id is not None else ARTIFACTS.index(name) +
            1,
        'created_at':
            created_at,
        'expired':
            expired,
    }


def producer_record(name,
                    *,
                    attempt=1,
                    status='completed',
                    conclusion='success',
                    started_at='2026-10-11T12:00:00Z'):
    return {
        'name': ('build / runtime-base-images' if name == 'runtime-base-images'
                 else f'build / image-build ({name}, Dockerfile, .)'),
        'id': 100 * attempt + ARTIFACTS.index(name),
        'run_attempt': attempt,
        'status': status,
        'conclusion': conclusion,
        'started_at': started_at,
    }


def callback_calls(result):
    return [
        line.split()
        for line in result.stdout.splitlines()
        if line.startswith('CALLBACK ')
    ]


class WaitForImageArtifactsTest(unittest.TestCase):

    def _run(
        self,
        *,
        ready_after: int,
        attempts: Optional[int] = 2,
        producer_jobs: tuple[tuple[str, str, str], ...] = (),
        jobs_api_fails: bool = False,
        jobs_api_fails_on: tuple[int, ...] = (),
        artifacts_api_fails_on: tuple[int, ...] = (),
        artifact_snapshots=None,
        job_snapshots=None,
        run_attempt: str = '1',
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
            jobs = [
                producer_record(name, attempt=int(run_attempt))
                for name in ARTIFACTS
            ]
            for name, status, conclusion in producer_jobs:
                matches = [
                    job for job in jobs
                    if job['name'].split(',')[0] == name.split(',')[0]
                ]
                job = matches[0] if matches else {
                    'id': 999,
                    'run_attempt': int(run_attempt),
                    'started_at': '2026-10-11T12:00:00Z'
                }
                job.update(
                    name=name, status=status, conclusion=conclusion or None)
                if not matches:
                    jobs.append(job)
            fixtures = root / 'fixtures.json'
            fixtures.write_text(
                json.dumps({
                    'artifacts': [artifact_record(name) for name in ARTIFACTS],
                    'jobs': jobs,
                    'artifact_snapshots': artifact_snapshots,
                    'job_snapshots': job_snapshots,
                }))
            fake_gh = fake_bin / 'gh'
            fake_gh.write_text(
                f'#!{sys.executable}\n' + textwrap.dedent('''\
                    import json
                    import os
                    from pathlib import Path
                    import sys

                    data = json.loads(Path(os.environ['FIXTURES']).read_text())
                    counter = Path(os.environ['GH_COUNTER'])
                    count = int(counter.read_text())
                    ready = int(os.environ['READY_AFTER'])
                    if '/jobs?' in ' '.join(sys.argv):
                        expected = f"/attempts/{os.environ['GITHUB_RUN_ATTEMPT']}/jobs?"
                        if expected not in ' '.join(sys.argv):
                            sys.exit('Expected attempt-specific producer jobs')
                        if os.environ['JOBS_API_FAILS'] == 'true' or str(count) in os.environ['JOBS_API_FAILS_ON'].split():
                            sys.exit(1)
                        snapshots = data['job_snapshots']
                        records = snapshots[min(count - 1, len(snapshots) - 1)] if snapshots else data['jobs']
                        if snapshots is None and count >= ready:
                            for record in records:
                                if record['status'] in ('queued', 'in_progress'):
                                    record.update(status='completed', conclusion='success')
                    else:
                        count += 1
                        counter.write_text(str(count))
                        if str(count) in os.environ['ARTIFACTS_API_FAILS_ON'].split():
                            sys.exit(1)
                        snapshots = data['artifact_snapshots']
                        records = snapshots[min(count - 1, len(snapshots) - 1)] if snapshots else data['artifacts']
                        if snapshots is None and count < ready:
                            records = [record for record in records if record['name'] != os.environ['MISSING_ARTIFACT']]
                    for record in records:
                        print(json.dumps(record))
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
                'FIXTURES':
                    str(fixtures),
                'GH_COUNTER':
                    str(counter),
                'GITHUB_REPOSITORY':
                    'kubeflow/pipelines',
                'GITHUB_RUN_ID':
                    '123',
                'GITHUB_RUN_ATTEMPT':
                    run_attempt,
                'JOBS_API_FAILS':
                    str(jobs_api_fails).lower(),
                'JOBS_API_FAILS_ON':
                    ' '.join(str(attempt) for attempt in jobs_api_fails_on),
                'MISSING_ARTIFACT':
                    missing_artifact,
                'PATH':
                    f'{fake_bin}{os.pathsep}{environment["PATH"]}',
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

    def test_duplicate_names_select_newest_nonexpired_artifact_id(self):
        artifacts = [artifact_record(name) for name in ARTIFACTS]
        artifacts.extend([
            artifact_record(
                'frontend',
                artifact_id=40,
                created_at='2026-10-11T12:02:00.500Z'),
            artifact_record(
                'frontend',
                artifact_id=41,
                created_at='2026-10-11T14:02:00.500+02:00'),
            artifact_record(
                'frontend', artifact_id=900, created_at='2026-10-11T12:02:00Z'),
            artifact_record(
                'frontend',
                artifact_id=901,
                created_at='2026-10-11T13:00:00Z',
                expired=True),
        ])
        result, attempts = self._run(
            ready_after=1, use_callback=True, artifact_snapshots=[artifacts])
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = callback_calls(result)
        self.assertEqual(attempts, 1)
        self.assertEqual(len(calls), 1)
        self.assertEqual(
            [token for token in calls[0][2:] if token.startswith('frontend:')],
            ['frontend:41'])

    def test_rerun_waits_for_active_producer_and_fresh_publication(self):
        old_artifacts = [artifact_record(name) for name in ARTIFACTS]
        new_artifacts = old_artifacts + [
            artifact_record(
                'frontend', artifact_id=404, created_at='2026-10-11T13:01:00Z')
        ]
        snapshots = []
        for status, conclusion, started in (('queued', None,
                                             None), ('in_progress', None,
                                                     '2026-10-11T13:00:00Z'),
                                            ('completed', 'success',
                                             '2026-10-11T13:00:00Z')):
            jobs = [producer_record(name, attempt=2) for name in ARTIFACTS]
            jobs[3] = producer_record(
                'frontend',
                attempt=2,
                status=status,
                conclusion=conclusion,
                started_at=started)
            snapshots.append(jobs)
        result, attempts = self._run(
            ready_after=1,
            use_callback=True,
            run_attempt='2',
            artifact_snapshots=[
                old_artifacts, old_artifacts, old_artifacts, new_artifacts
            ],
            job_snapshots=snapshots)
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = callback_calls(result)
        self.assertEqual(attempts, 4)
        self.assertEqual(len(calls), 2)
        self.assertEqual(calls[0][1], '1')
        self.assertFalse(
            any(token.startswith('frontend:') for token in calls[0][2:]))
        self.assertEqual(calls[1], ['CALLBACK', '4', 'frontend:404'])
        self.assertIn(
            'Extending image artifact wait; active producers: frontend',
            result.stdout)

    def test_failed_job_only_rerun_reuses_retained_successful_producers(self):
        # GitHub copies successful jobs into attempt 2 with new IDs but keeps
        # their original started_at. Their attempt-1 artifacts remain valid.
        jobs = [producer_record(name, attempt=2) for name in ARTIFACTS]
        result, attempts = self._run(
            ready_after=1,
            use_callback=True,
            run_attempt='2',
            job_snapshots=[jobs])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(attempts, 1)
        self.assertCountEqual(
            callback_calls(result)[0][2:],
            [f'{name}:{index + 1}' for index, name in enumerate(ARTIFACTS)])

    def test_old_attempt_job_snapshot_cannot_admit_existing_artifacts(self):
        jobs = [[producer_record(name, attempt=attempt)
                 for name in ARTIFACTS]
                for attempt in (1, 2)]
        result, attempts = self._run(
            ready_after=1,
            use_callback=True,
            run_attempt='2',
            job_snapshots=jobs)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(attempts, 2)
        calls = callback_calls(result)
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0][1], '2')

    def test_failed_producer_cannot_admit_an_existing_artifact(self):
        result, attempts = self._run(
            ready_after=1,
            use_callback=True,
            producer_jobs=(('build / image-build (frontend, Dockerfile, .)',
                            'completed', 'failure'),))
        self.assertEqual(result.returncode, 1)
        self.assertEqual(attempts, 2)
        self.assertIn('frontend:failure', result.stderr)
        self.assertFalse(
            any(
                token.startswith('frontend:')
                for call in callback_calls(result)
                for token in call[2:]))

    def test_existing_artifacts_do_not_bypass_jobs_api_failure(self):
        result, attempts = self._run(
            ready_after=1, attempts=1, use_callback=True, jobs_api_fails=True)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(attempts, 2)
        self.assertEqual(callback_calls(result), [])

    def test_malformed_metadata_never_emits_a_partial_ready_batch(self):
        for kind, field, value in (('artifact', 'id', '8'),
                                   ('artifact', 'created_at',
                                    'not-a-date'), ('artifact', 'created_at',
                                                    '2026-10-11T12:01:00'),
                                   ('artifact', 'expired',
                                    None), ('job', 'started_at', None),
                                   ('job', 'run_attempt', None)):
            with self.subTest(kind=kind, field=field, value=value):
                artifacts = [artifact_record(name) for name in ARTIFACTS]
                jobs = [producer_record(name) for name in ARTIFACTS]
                records = artifacts if kind == 'artifact' else jobs
                records[-1][field] = value
                result, _ = self._run(
                    ready_after=1,
                    attempts=1,
                    use_callback=True,
                    artifact_snapshots=[artifacts],
                    job_snapshots=[jobs],
                    producer_state_unavailable_extensions=0)
                self.assertEqual(result.returncode, 1)
                self.assertEqual(callback_calls(result), [])
                self.assertIn('Cannot safely select image artifacts',
                              result.stderr)

    def test_changed_artifact_id_is_reimported_while_waiting(self):
        initial = [
            artifact_record(name) for name in ARTIFACTS if name != 'frontend'
        ]
        replaced = [record for record in initial if record['name'] != 'driver']
        replaced.append(
            artifact_record(
                'driver', artifact_id=66, created_at='2026-10-11T12:02:00Z'))
        complete = replaced + [artifact_record('frontend')]
        result, attempts = self._run(
            ready_after=1,
            attempts=4,
            use_callback=True,
            artifact_snapshots=[initial, replaced, complete])
        self.assertEqual(result.returncode, 0, result.stderr)
        calls = callback_calls(result)
        self.assertEqual(attempts, 3)
        self.assertIn('driver:6', calls[0])
        self.assertEqual(calls[1], ['CALLBACK', '2', 'driver:66'])
        self.assertEqual(calls[2], ['CALLBACK', '3', 'frontend:4'])

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
        self.assertCountEqual(calls[0][2:], [
            f'{name}:{ARTIFACTS.index(name) + 1}' for name in ARTIFACTS
            if name != 'frontend'
        ])
        self.assertEqual(calls[1], ['CALLBACK', '3', 'frontend:4'])
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
        self.assertCountEqual(calls[0][2:], [
            f'{name}:{ARTIFACTS.index(name) + 1}' for name in ARTIFACTS
            if name != 'frontend'
        ])
        self.assertEqual(calls[1], ['CALLBACK', '3', 'frontend:4'])
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
                'unrelated workflow job',
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
            jobs_api_fails_on=(1,),
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
