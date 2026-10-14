#!/usr/bin/env python3

import contextlib
import json
from pathlib import Path
import re
import tempfile
import unittest
from unittest import mock

import runtime_base_image_artifacts as artifacts

SOURCE_SHA = 'current-source-sha'


def artifact(
    artifact_id: int,
    created_at: str,
    *,
    head_sha: str = 'other-sha',
    head_branch: str = 'feature',
    head_repository_id: int = 1,
    repository_id: int = 1,
    expired: bool = False,
) -> dict[str, object]:
    return {
        'id': artifact_id,
        'created_at': created_at,
        'expired': expired,
        'workflow_run': {
            'id': artifact_id * 10,
            'head_sha': head_sha,
            'head_branch': head_branch,
            'head_repository_id': head_repository_id,
            'repository_id': repository_id,
        },
    }


class FingerprintTest(unittest.TestCase):

    def test_changes_when_any_generation_input_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            paths = [
                Path(directory) / name
                for name in ('images', 'helper', 'workflow')
            ]
            for path in paths:
                path.write_text(f'{path.name} contents')
            original = artifacts.fingerprint_files(paths)

            for path in paths:
                original_contents = path.read_text()
                path.write_text(f'{original_contents} changed')
                self.assertNotEqual(
                    artifacts.fingerprint_files(paths), original)
                path.write_text(original_contents)


class ProducerSelectionTest(unittest.TestCase):

    def test_selects_newest_current_source_or_upstream_master(self):
        payload = {
            'artifacts': [
                artifact(1, '2026-07-16T00:00:00Z', head_sha=SOURCE_SHA),
                artifact(2, '2026-07-16T01:00:00Z', head_branch='master'),
                artifact(3, '2026-07-16T02:00:00Z'),
            ]
        }

        self.assertEqual(
            artifacts.select_producer_run_id(payload, SOURCE_SHA),
            20,
        )

    def test_rejects_newer_fork_branch_named_master(self):
        payload = {
            'artifacts': [
                artifact(1, '2026-07-16T00:00:00Z', head_branch='master'),
                artifact(
                    2,
                    '2026-07-16T01:00:00Z',
                    head_branch='master',
                    head_repository_id=2,
                    repository_id=1,
                ),
            ]
        }

        self.assertEqual(
            artifacts.select_producer_run_id(payload, SOURCE_SHA),
            10,
        )

    def test_accepts_current_source_from_fork(self):
        payload = {
            'artifacts': [
                artifact(
                    1,
                    '2026-07-16T00:00:00Z',
                    head_sha=SOURCE_SHA,
                    head_repository_id=2,
                    repository_id=1,
                )
            ]
        }

        self.assertEqual(
            artifacts.select_producer_run_id(payload, SOURCE_SHA),
            10,
        )

    def test_rejects_master_without_repository_identity(self):
        candidate = artifact(1, '2026-07-16T00:00:00Z', head_branch='master')
        workflow_run = candidate['workflow_run']
        self.assertIsInstance(workflow_run, dict)
        workflow_run['head_repository_id'] = None
        workflow_run['repository_id'] = None

        self.assertIsNone(
            artifacts.select_producer_run_id(
                {'artifacts': [candidate]},
                SOURCE_SHA,
            ))

    def test_ignores_expired_artifact(self):
        payload = {
            'artifacts': [
                artifact(
                    1,
                    '2026-07-16T00:00:00Z',
                    head_sha=SOURCE_SHA,
                    expired=True,
                )
            ]
        }

        self.assertIsNone(artifacts.select_producer_run_id(payload, SOURCE_SHA))


if __name__ == '__main__':
    unittest.main()


class BundleReferencesTest(unittest.TestCase):

    def bundle_set(self, run, stamp='2026-01-01T00:00:00Z'):
        return [
            dict(
                artifact(run * 10 + index, stamp, head_branch='master'),
                name='generation-' + bundle,
                expires_at='2099-01-01T00:00:00Z',
                workflow_run=dict(
                    id=run,
                    repository_id=1,
                    head_repository_id=1,
                    head_branch='master'))
            for index, bundle in enumerate(artifacts.BUNDLES)
        ]

    @mock.patch.object(
        artifacts, 'expected_artifact_name', return_value='generation')
    def test_partial_newer_producer_never_mixes_with_complete_previous(
            self, unused):
        older = self.bundle_set(1)
        newer = self.bundle_set(2, '2026-02-01T00:00:00Z')
        chosen = artifacts.select_bundles([{
            'artifacts': older + newer[:-1]
        }], 'source', 1)
        self.assertEqual(
            set(chosen['bundles'].values()), {a['id'] for a in older})
        chosen = artifacts.select_bundles([{
            'artifacts': older + newer
        }], 'source', 1)
        self.assertEqual(
            set(chosen['bundles'].values()), {a['id'] for a in newer})

    @mock.patch.object(
        artifacts, 'expected_artifact_name', return_value='generation')
    def test_incomplete_or_untrusted_bundle_set_is_cache_miss(self, unused):
        candidates = self.bundle_set(1)
        candidates[-1]['expired'] = True
        self.assertIsNone(
            artifacts.select_bundles([{
                'artifacts': candidates
            }], 'source', 1))

    def test_reference_requires_exact_complete_unique_positive_ids(self):
        good = {
            bundle: index + 1 for index, bundle in enumerate(artifacts.BUNDLES)
        }
        invalid = [
            dict(good, fixtures=True),
            dict(good, fixtures=0),
            dict(good, fixtures='1'),
            dict(good, fixtures=2),
            dict(good, unexpected=10), {
                key: value for key, value in good.items() if key != 'mysql'
            }
        ]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'reference.json'
            for values in invalid:
                with self.subTest(values=values):
                    path.write_text(json.dumps({'bundles': values}))
                    with self.assertRaises(ValueError):
                        artifacts.bundle_reference(path)
            path.write_text(json.dumps({'bundles': good}))
            self.assertEqual(artifacts.bundle_reference(path), good)


class GenerationFingerprintAlignmentTest(unittest.TestCase):

    def test_producer_and_consumer_cli_fingerprint_inputs_match_validator(self):
        root = Path(__file__).resolve().parents[3]
        for filename in ('image-builds.yml', 'runtime-base-images.yml'):
            source = (root / '.github/workflows' / filename).read_text()
            invocation = source.split(
                'runtime_base_image_artifacts.py fingerprint',
                1)[1].split(')', 1)[0]
            paths = re.findall(r'(?:\.github|test_data)/[^\s\\]+', invocation)
            self.assertEqual(paths, list(artifacts.GENERATION_INPUTS))
            with contextlib.chdir(root):
                self.assertEqual(
                    'runtime-base-images-' +
                    artifacts.fingerprint_files(map(Path, paths)),
                    artifacts.expected_artifact_name())
