"""Combined downloads retain fresh evidence while sharing metadata requests."""

import copy
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import shared_ci_images as shared
from shared_ci_images_test import FakeAPI, identity


class CombinedVerificationTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.directory = Path(temp.name)
        self.consumer = identity(20)
        self.api = FakeAPI(identity())
        self.api.runtime(self.directory)
        self.api.images('amd64', self.directory)
        self.state = shared.resolve_once(
            self.api, self.consumer, 'amd64', 'all', None
        )

    def verify(self):
        shared.verify(
            self.api, self.state, self.consumer, self.directory, 'nested', 'all'
        )

    def test_combined_verification_reads_one_fresh_snapshot_and_run(self):
        with mock.patch.object(
            self.api, 'artifacts', wraps=self.api.artifacts
        ) as artifacts:
            with mock.patch.object(self.api, 'run', wraps=self.api.run) as run:
                for kind in ('runtime', 'images'):
                    shared.verify(
                        self.api,
                        copy.deepcopy(self.state),
                        self.consumer,
                        self.directory,
                        'flat',
                        kind,
                    )
        self.assertEqual(artifacts.call_count + run.call_count, 4)
        with mock.patch.object(
            self.api, 'artifacts', wraps=self.api.artifacts
        ) as artifacts:
            with mock.patch.object(self.api, 'run', wraps=self.api.run) as run:
                self.verify()
        self.assertEqual(artifacts.call_count, 1)
        self.assertEqual(run.call_count, 1)
        for name in ('runtime-base-images', 'frontend'):
            self.assertTrue((self.directory / name / f'{name}.tar').is_file())
        self.assertEqual(set(self.state['sets']), {'runtime', 'images'})

    def test_import_and_outputs_cover_both_sets_without_api_reads(self):
        state = shared.import_ready_state(
            json.dumps(self.state), self.consumer, 'amd64', 'all'
        )
        output = self.directory / 'outputs'
        shared.emit_outputs(output, state, 'all')
        expected = ','.join(
            str(record['artifact_id'])
            for kind in ('runtime', 'images')
            for record in state['sets'][kind]
        )
        self.assertIn('artifact-ids=' + expected, output.read_text())
        self.assertIn('runtime-base-images.tar', output.read_text())
        self.assertIn('frontend.tar', output.read_text())

    def test_corrupt_runtime_prevents_all_remote_reads_and_rearrangement(self):
        (self.directory / 'runtime-base-images.tar').write_text('corrupt')
        with mock.patch.object(self.api, 'artifacts') as artifacts:
            with self.assertRaisesRegex(
                shared.InvalidEvidence, 'Checksum mismatch'
            ):
                self.verify()
        artifacts.assert_not_called()
        self.assertFalse((self.directory / 'frontend').exists())

    def test_changed_attempt_after_hashing_rejects_all_files(self):
        original = shared.checksum

        def checksum(path):
            result = original(path)
            self.api.current_run['run_attempt'] = 2
            return result

        with mock.patch.object(shared, 'checksum', side_effect=checksum):
            with self.assertRaises(shared.InvalidEvidence):
                self.verify()
        self.assertFalse((self.directory / 'frontend').exists())
        self.assertFalse((self.directory / 'runtime-base-images').exists())

    def test_expired_runtime_or_branch_artifact_rejected_after_hashing(self):
        for kind in ('runtime', 'images'):
            with self.subTest(kind=kind):
                artifact = self.api.items[
                    self.state['sets'][kind][0]['artifact_id']
                ]
                artifact['expired'] = True
                with self.assertRaises(shared.InvalidEvidence):
                    self.verify()
                self.assertFalse((self.directory / 'frontend').exists())
                artifact['expired'] = False

    def test_failed_producer_prevents_loading_either_set(self):
        self.api.current_run.update(status='completed', conclusion='failure')
        with self.assertRaises(shared.InvalidEvidence):
            self.verify()
        self.assertFalse((self.directory / 'frontend').exists())
        self.assertFalse((self.directory / 'runtime-base-images').exists())

    def test_missing_set_cannot_be_accepted_as_combined(self):
        for kind in ('runtime', 'images'):
            state = copy.deepcopy(self.state)
            del state['sets'][kind]
            with self.assertRaises(shared.InvalidEvidence):
                shared.import_ready_state(
                    json.dumps(state), self.consumer, 'amd64', 'all'
                )
            with self.assertRaises(shared.InvalidEvidence):
                shared.verify(
                    self.api,
                    state,
                    self.consumer,
                    self.directory,
                    'nested',
                    'all',
                )


if __name__ == '__main__':
    unittest.main()
