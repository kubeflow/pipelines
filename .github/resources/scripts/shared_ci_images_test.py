"""Exact-revision, attempt and archive integrity contracts for shared CI images."""

import copy
import io
import json
import tempfile
import unittest
import zipfile
from pathlib import Path
from unittest import mock

import shared_ci_images as shared


def identity(run_id=10):
    return {
        'version': 1,
        'repository': 'upstream/pipelines',
        'repository_id': 100,
        'event': 'pull_request',
        'ref': 'refs/pull/23/merge',
        'source_sha': 'a' * 40,
        'run_id': run_id,
        'run_attempt': 1,
        'pr': {
            'number': 23,
            'head_repository': 'fork/pipelines',
            'head_repository_id': 200,
            'head_sha': 'b' * 40,
            'base_repository': 'upstream/pipelines',
            'base_repository_id': 100,
            'base_ref': 'main',
            'base_sha': 'c' * 40,
        },
    }


def run_for(source):
    pr = source['pr']
    return {
        'id': source['run_id'],
        'run_attempt': source['run_attempt'],
        'event': source['event'],
        'path': shared.WORKFLOW,
        'head_sha': pr['head_sha'] if pr else source['source_sha'],
        'repository': {
            'id': source['repository_id'],
            'full_name': source['repository'],
        },
        'head_repository': {
            'id': pr['head_repository_id'] if pr else source['repository_id'],
            'full_name': pr['head_repository'] if pr else source['repository'],
        },
        'status': 'in_progress',
        'conclusion': None,
        'pull_requests': [],
    }


class FakeAPI:
    def __init__(self, source):
        self.source = source
        self.current_run = run_for(source)
        self.items = {}
        self.documents = {}
        self.run_error = None
        self.add('ci-image-source-attempt-1', source)

    def add(self, name, document=None):
        artifact_id = len(self.items) + 1
        pr = self.source['pr']
        self.items[artifact_id] = {
            'id': artifact_id,
            'name': name,
            'expired': False,
            'size_in_bytes': 100,
            'workflow_run': {
                'id': self.source['run_id'],
                'repository_id': self.source['repository_id'],
                'head_repository_id': pr['head_repository_id']
                if pr
                else self.source['repository_id'],
                'head_sha': pr['head_sha'] if pr else self.source['source_sha'],
            },
        }
        if document is not None:
            self.documents[artifact_id] = document
        return artifact_id

    def runs(self, sha):
        if self.run_error:
            raise self.run_error
        return [self.current_run]

    def run(self, run_id):
        return self.current_run

    def artifacts(self, run_id):
        return list(self.items.values())

    def artifact(self, artifact_id):
        return self.items[artifact_id]

    def document(self, artifact, filename):
        return copy.deepcopy(self.documents[artifact['id']])

    def images(self, architecture, directory):
        records = []
        for image in shared.IMAGES:
            artifact_id = self.add(shared.artifact_name(image, architecture, 1))
            for filename in shared.filenames(image, architecture):
                (directory / filename).write_bytes(filename.encode())
            records.append(
                shared.make_record(
                    self.source,
                    architecture,
                    image,
                    artifact_id,
                    directory,
                    self,
                )
            )
        manifest = shared.make_manifest(
            self.source, architecture, records, self
        )
        self.add(f'ci-image-set-{architecture}-attempt-1', manifest)
        return records

    def runtime(self, directory):
        artifact_id = self.add('runtime-base-images-attempt-1')
        for filename in shared.filenames('runtime-base-images', 'amd64'):
            (directory / filename).write_bytes(filename.encode())
        record = shared.make_record(
            self.source,
            'amd64',
            'runtime-base-images',
            artifact_id,
            directory,
            self,
        )
        self.add('runtime-base-images-record-amd64-attempt-1', record)


class SharedImageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.producer = identity()
        self.consumer = identity(20)
        self.api = FakeAPI(self.producer)
        self.state_path = self.directory / 'state.json'

    def resolve(self, kind='images', architecture='amd64'):
        return shared.resolve(
            self.api,
            self.consumer,
            architecture,
            kind,
            self.state_path,
            timeout=1,
            poll=1,
        )

    def test_actual_merge_sha_not_head_sha_controls_reuse(self):
        self.api.images('amd64', self.directory)
        self.consumer['source_sha'] = 'd' * 40
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'source_sha mismatch'
        ):
            self.resolve()

    def test_identity_rejects_other_base_and_fork(self):
        for key, value in [
            ('base_sha', 'd' * 40),
            ('base_ref', 'release-3'),
            ('head_repository', 'other/pipelines'),
            ('number', 99),
        ]:
            with self.subTest(key=key):
                changed = copy.deepcopy(self.producer)
                changed['pr'][key] = value
                with self.assertRaises(shared.InvalidEvidence):
                    shared.same_source(changed, self.consumer)

    def test_empty_api_pr_array_is_supported(self):
        shared.validate_run(self.api.current_run, self.producer)

    def test_wrong_repo_head_repo_event_workflow_or_attempt_rejected(self):
        for path, value in [
            ('path', 'other.yml'),
            ('event', 'push'),
            ('run_attempt', 2),
            ('head_sha', 'e' * 40),
            ('repository', {'id': 999}),
            ('head_repository', {'id': 999}),
        ]:
            with self.subTest(path=path):
                run = dict(self.api.current_run, **{path: value})
                with self.assertRaises(shared.InvalidEvidence):
                    shared.validate_run(run, self.producer)

    def test_runtime_then_images_pin_one_attempt_and_nested_layout(self):
        self.api.runtime(self.directory)
        runtime = self.resolve('runtime')
        shared.verify(self.api, runtime, self.consumer, self.directory, 'flat')
        self.api.images('amd64', self.directory)
        state = self.resolve()
        self.assertEqual(set(state['sets']), {'runtime', 'images'})
        shared.verify(self.api, state, self.consumer, self.directory, 'nested')
        self.assertTrue((self.directory / 'frontend/frontend.tar').is_file())
        self.assertFalse((self.directory / 'frontend.tar').exists())

    def test_arm_checksums_include_existing_identity_record(self):
        self.api.images('arm64', self.directory)
        state = self.resolve(architecture='arm64')
        shared.verify(self.api, state, self.consumer, self.directory, 'flat')
        self.assertTrue((self.directory / 'frontend-arm64.json').is_file())
        (self.directory / 'frontend-arm64.json').write_text('wrong')
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'Checksum mismatch'
        ):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )

    def test_rerun_between_phases_fails_instead_of_switching(self):
        self.api.runtime(self.directory)
        self.resolve('runtime')
        self.api.current_run['run_attempt'] = 2
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'run_attempt mismatch'
        ):
            self.resolve()

    def test_rerun_during_download_is_rejected(self):
        self.api.images('amd64', self.directory)
        state = self.resolve()
        self.api.current_run['run_attempt'] = 2
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'run_attempt mismatch'
        ):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )

    def test_terminal_failed_producer_rejects_available_artifacts(self):
        self.api.images('amd64', self.directory)
        self.api.current_run.update(status='completed', conclusion='failure')
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'rerun the complete producer'
        ):
            self.resolve()

    def test_expired_image_artifact_is_rejected(self):
        records = self.api.images('amd64', self.directory)
        self.api.items[records[0]['artifact_id']]['expired'] = True
        with self.assertRaisesRegex(shared.InvalidEvidence, 'expired'):
            self.resolve()

    def test_mixed_attempt_record_rejected(self):
        records = self.api.images('amd64', self.directory)
        records[0] = copy.deepcopy(records[0])
        records[0]['identity']['run_attempt'] = 2
        with self.assertRaisesRegex(shared.InvalidEvidence, 'Mixed source'):
            shared.make_manifest(self.producer, 'amd64', records, self.api)

    def test_seven_distinct_images_are_required(self):
        records = self.api.images('amd64', self.directory)
        for invalid in (records[:-1], records[:-1] + [records[0]]):
            with self.assertRaises(shared.InvalidEvidence):
                shared.make_manifest(self.producer, 'amd64', invalid, self.api)

    def test_path_traversal_record_is_rejected(self):
        records = self.api.images('amd64', self.directory)
        records[0]['files'] = {'../escape.tar': 'a' * 64}
        with self.assertRaisesRegex(shared.InvalidEvidence, 'inventory'):
            shared.make_manifest(self.producer, 'amd64', records, self.api)

    def test_corruption_and_symlink_rejected_before_rearrangement(self):
        self.api.images('amd64', self.directory)
        state = self.resolve()
        (self.directory / 'frontend.tar').write_text('corrupt')
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'Checksum mismatch'
        ):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'nested'
            )
        self.assertFalse((self.directory / 'apiserver').exists())
        (self.directory / 'frontend.tar').unlink()
        (self.directory / 'frontend.tar').symlink_to(
            self.directory / 'launcher.tar'
        )
        with self.assertRaisesRegex(shared.InvalidEvidence, 'unsafe'):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )

    def test_manual_consumers_allow_push_but_never_pr(self):
        manual = dict(
            self.consumer,
            event='workflow_dispatch',
            ref='refs/heads/main',
            pr=None,
        )
        push = dict(manual, event='push', run_id=10)
        shared.same_source(push, manual)
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(self.producer, manual)
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(dict(push, ref='refs/heads/release-3'), manual)

    def test_output_uses_immutable_ids_and_flat_required_files(self):
        records = self.api.images('amd64', self.directory)
        state = self.resolve()
        output = self.directory / 'outputs'
        shared.emit_outputs(output, state, 'images')
        text = output.read_text()
        self.assertIn('run-id=10\n', text)
        self.assertIn(
            'artifact-ids=' + ','.join(str(r['artifact_id']) for r in records),
            text,
        )
        self.assertIn('\nfrontend.tar\n', text)

    def test_bounded_wait_retries_api_errors_and_missing_artifacts(self):
        now = [0]

        def sleep(seconds):
            now[0] += seconds

        for error in (None, shared.APIError('unavailable')):
            self.api.run_error = error
            now[0] = 0
            with self.assertRaisesRegex(shared.InvalidEvidence, 'Timed out'):
                shared.resolve(
                    self.api,
                    self.consumer,
                    'amd64',
                    'images',
                    self.state_path,
                    timeout=2,
                    poll=1,
                    clock=lambda: now[0],
                    sleep=sleep,
                )
            self.assertEqual(now[0], 2)

    def test_delayed_publication_eventually_resolves(self):
        now = [0]

        def sleep(seconds):
            now[0] += seconds
            self.api.images('amd64', self.directory)

        state = shared.resolve(
            self.api,
            self.consumer,
            'amd64',
            'images',
            self.state_path,
            timeout=2,
            poll=1,
            clock=lambda: now[0],
            sleep=sleep,
        )
        self.assertEqual(state['identity']['run_id'], 10)

    def test_malformed_matching_run_cannot_hide_behind_older_success(self):
        malformed = dict(self.api.current_run, id='10')
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'Malformed matching'
        ):
            shared.candidate_runs(
                [malformed, self.api.current_run], self.consumer
            )

    def test_artifact_from_wrong_run_or_repository_rejected(self):
        artifact = copy.deepcopy(self.api.items[1])
        for field in ('id', 'repository_id', 'head_repository_id'):
            modified = copy.deepcopy(artifact)
            modified['workflow_run'][field] = 999
            with self.assertRaises(shared.InvalidEvidence):
                shared.validate_artifact(
                    modified, self.producer, artifact['name']
                )

    def test_download_rejects_extra_archive_entries(self):
        stream = io.BytesIO()
        with zipfile.ZipFile(stream, 'w') as archive:
            archive.writestr('source.json', json.dumps(self.producer))
            archive.writestr('../other', 'bad')
        api = shared.GitHub('upstream/pipelines')
        with mock.patch.object(api, 'request', return_value=stream.getvalue()):
            with self.assertRaises(shared.InvalidEvidence):
                api.document({'id': 1}, 'source.json')

    def test_sha_regression_guard_rejects_head_only_implementation(self):
        changed = dict(self.producer, source_sha='d' * 40)
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(changed, self.consumer)

    def test_consumer_validation_uses_bulk_snapshots_not_per_image_requests(
        self,
    ):
        self.api.images('amd64', self.directory)
        with mock.patch.object(
            self.api,
            'artifact',
            side_effect=AssertionError('N+1 artifact request'),
        ):
            with mock.patch.object(
                self.api, 'artifacts', wraps=self.api.artifacts
            ) as listing:
                state = self.resolve()
                self.assertEqual(listing.call_count, 1)
            with mock.patch.object(
                self.api, 'artifacts', wraps=self.api.artifacts
            ) as listing:
                shared.verify(
                    self.api, state, self.consumer, self.directory, 'flat'
                )
                self.assertEqual(listing.call_count, 1)

    def test_completed_missing_manifest_has_short_publication_grace(self):
        self.api.current_run.update(status='completed', conclusion='success')
        now = [0]

        def sleep(seconds):
            now[0] += seconds

        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'rerun the complete producer'
        ):
            shared.resolve(
                self.api,
                self.consumer,
                'amd64',
                'images',
                self.state_path,
                timeout=2400,
                poll=30,
                clock=lambda: now[0],
                sleep=sleep,
            )
        self.assertLessEqual(now[0], 60)

    def test_first_pr_workflow_not_registered_retries_404(self):
        api = shared.GitHub('upstream/pipelines')
        response = mock.Mock(returncode=1, stdout=b'', stderr=b'HTTP 404')
        with mock.patch.object(shared.subprocess, 'run', return_value=response):
            with self.assertRaises(shared.APIError):
                api.runs(self.consumer['pr']['head_sha'])

    def test_failed_only_rerun_with_old_source_expires_publication_grace(self):
        self.api.current_run.update(
            run_attempt=2, status='completed', conclusion='success'
        )
        now = [0]

        def sleep(seconds):
            now[0] += seconds

        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'rerun the complete producer'
        ):
            shared.resolve(
                self.api,
                self.consumer,
                'amd64',
                'images',
                self.state_path,
                timeout=2400,
                poll=30,
                clock=lambda: now[0],
                sleep=sleep,
            )
        self.assertEqual(now[0], 60)

    def test_expiry_during_checksum_is_detected_by_fresh_snapshot(self):
        records = self.api.images('amd64', self.directory)
        state = self.resolve()
        original = shared.checksum

        def checksum_and_expire(path):
            result = original(path)
            if path.name == 'launcher.tar':
                self.api.items[records[0]['artifact_id']]['expired'] = True
            return result

        with mock.patch.object(
            shared, 'checksum', side_effect=checksum_and_expire
        ):
            with self.assertRaisesRegex(shared.InvalidEvidence, 'expired'):
                shared.verify(
                    self.api, state, self.consumer, self.directory, 'flat'
                )

    def test_api_request_timeout_respects_remaining_wait_budget(self):
        api = shared.GitHub('upstream/pipelines')
        api.deadline = 15
        response = mock.Mock(returncode=0, stdout=b'{}')
        with mock.patch.object(shared.time, 'monotonic', return_value=10):
            with mock.patch.object(
                shared.subprocess, 'run', return_value=response
            ) as request:
                api.run(10)
                self.assertEqual(request.call_args.kwargs['timeout'], 5)
        with mock.patch.object(shared.time, 'monotonic', return_value=16):
            with self.assertRaisesRegex(shared.APIError, 'deadline'):
                api.run(10)

    def test_source_is_pinned_before_wait_and_backoff_is_bounded(self):
        now = [0]
        sleeps = []

        def sleep(seconds):
            sleeps.append(seconds)
            now[0] += seconds
            state = json.loads(self.state_path.read_text())
            self.assertEqual(state['identity'], self.producer)
            self.assertEqual(state['sets'], {})

        with mock.patch.object(self.api, 'runs', wraps=self.api.runs) as runs:
            with mock.patch.object(
                self.api, 'document', wraps=self.api.document
            ) as document:
                with self.assertRaisesRegex(
                    shared.InvalidEvidence, 'Timed out'
                ):
                    shared.resolve(
                        self.api,
                        self.consumer,
                        'amd64',
                        'images',
                        self.state_path,
                        timeout=400,
                        poll=30,
                        clock=lambda: now[0],
                        sleep=sleep,
                    )
                self.assertEqual(runs.call_count, 1)
                self.assertEqual(document.call_count, 1)
        self.assertEqual(sleeps, [30, 60, 120, 120, 70])

    def test_attempt_change_during_initial_wait_rejects_reselection(self):
        now = [0]

        def sleep(seconds):
            now[0] += seconds
            self.api.current_run['run_attempt'] = 2

        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'run_attempt mismatch'
        ):
            shared.resolve(
                self.api,
                self.consumer,
                'amd64',
                'images',
                self.state_path,
                timeout=120,
                poll=30,
                clock=lambda: now[0],
                sleep=sleep,
            )


if __name__ == '__main__':
    unittest.main()
