"""Image bundles reduce downloads without weakening producer evidence."""

import copy
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import shared_ci_images as shared
from shared_ci_images_test import FakeAPI, identity


class BundleTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.directory = Path(temporary.name)
        self.consumer = identity(20)
        self.api = FakeAPI(identity())
        self.api.images('amd64', self.directory)

    def resolve(self):
        return shared.resolve_once(
            self.api,
            self.consumer,
            'amd64',
            'images',
            None,
            require_bundle=True,
        )

    def test_individual_images_do_not_satisfy_required_bundle(self):
        with self.assertRaises(shared.NotReady):
            self.resolve()

    def test_one_bundle_id_preserves_all_provenance_and_live_validation(self):
        bundle = self.api.add('ci-images-amd64-attempt-1')
        state = self.resolve()
        output = self.directory / 'output'
        shared.emit_outputs(output, state, 'images')
        self.assertIn(f'artifact-ids={bundle}\n', output.read_text())
        imported = shared.import_ready_state(
            json.dumps(state), self.consumer, 'amd64', 'images'
        )
        with mock.patch.object(
            self.api, 'artifacts', wraps=self.api.artifacts
        ) as artifacts:
            with mock.patch.object(self.api, 'run', wraps=self.api.run) as run:
                shared.verify(
                    self.api, imported, self.consumer, self.directory, 'flat'
                )
        self.assertEqual(artifacts.call_count, 1)
        self.assertEqual(run.call_count, 1)
        self.api.items[bundle]['expired'] = True
        with self.assertRaises(shared.InvalidEvidence):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )

    def test_bundle_rejects_corrupt_file_and_changed_attempt(self):
        self.api.add('ci-images-amd64-attempt-1')
        state = self.resolve()
        self.api.current_run['run_attempt'] = 2
        with self.assertRaises(shared.InvalidEvidence):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )
        self.api.current_run['run_attempt'] = 1
        (self.directory / 'frontend.tar').write_bytes(b'corrupt')
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'Checksum mismatch'
        ):
            shared.verify(
                self.api, state, self.consumer, self.directory, 'flat'
            )

    def test_bundle_rejects_mixed_artifact_ids(self):
        self.api.add('ci-images-amd64-attempt-1')
        state = copy.deepcopy(self.resolve())
        state['sets']['images'][0]['artifact_id'] += 1
        with self.assertRaises(shared.InvalidEvidence):
            shared.import_ready_state(
                json.dumps(state), self.consumer, 'amd64', 'images'
            )

    def test_producer_rejects_missing_extra_and_symlink_files(self):
        shared.validate_bundle(self.api.source, 'amd64', self.directory)
        (self.directory / 'unexpected').write_text('bad')
        with self.assertRaises(shared.InvalidEvidence):
            shared.validate_bundle(self.api.source, 'amd64', self.directory)
        (self.directory / 'unexpected').unlink()
        target = self.directory / 'frontend.tar'
        target.unlink()
        with self.assertRaises(shared.InvalidEvidence):
            shared.validate_bundle(self.api.source, 'amd64', self.directory)
        target.symlink_to(self.directory / 'driver.tar')
        with self.assertRaises(shared.InvalidEvidence):
            shared.validate_bundle(self.api.source, 'amd64', self.directory)

    def test_runtime_and_arm_bundles_download_only_selected_ids(self):
        self.api.runtime(self.directory)
        bundle = self.api.add('ci-images-amd64-attempt-1')
        state = shared.resolve_once(
            self.api, self.consumer, 'amd64', 'all', None, require_bundle=True
        )
        output = self.directory / 'combined-output'
        shared.emit_outputs(output, state, 'all')
        ids = (
            next(
                line
                for line in output.read_text().splitlines()
                if line.startswith('artifact-ids=')
            )
            .split('=')[1]
            .split(',')
        )
        self.assertEqual(len(ids), 2)
        self.assertIn(str(bundle), ids)
        shared.verify(
            self.api, state, self.consumer, self.directory, 'flat', 'all'
        )
        arm = self.directory / 'arm'
        arm.mkdir()
        api = FakeAPI(identity())
        api.images('arm64', arm)
        api.add('ci-images-arm64-attempt-1')
        state = shared.resolve_once(
            api, self.consumer, 'arm64', 'images', None, require_bundle=True
        )
        self.assertEqual(
            len({record['artifact_id'] for record in state['sets']['images']}),
            1,
        )
        shared.verify(api, state, self.consumer, arm, 'flat')

    def test_producer_rejects_previous_attempt_provenance(self):
        with self.assertRaises(shared.InvalidEvidence):
            shared.validate_bundle(
                dict(self.api.source, run_attempt=2), 'amd64', self.directory
            )

    def test_cli_readiness_and_direct_resolver_require_bundle(self):
        for command in ('prepare', 'resolve'):
            args = [
                'shared_ci_images.py',
                command,
                '--architecture',
                'amd64',
                '--state',
                str(self.directory / 'state'),
                '--output',
                str(self.directory / 'output'),
            ]
            if command == 'resolve':
                args += ['--kind', 'images']
            with (
                mock.patch('sys.argv', args),
                mock.patch.object(
                    shared, 'environment_identity', return_value=self.consumer
                ),
            ):
                with mock.patch.object(
                    shared,
                    'resolve',
                    side_effect=shared.InvalidEvidence('test stop'),
                ) as resolve:
                    self.assertEqual(shared.main(), 1)
                    self.assertTrue(resolve.call_args.kwargs['require_bundle'])


if __name__ == '__main__':
    unittest.main()
