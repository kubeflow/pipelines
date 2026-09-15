#!/usr/bin/env python3

import tempfile
import unittest
from pathlib import Path

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
            paths = [Path(directory) / name for name in ('images', 'helper', 'workflow')]
            for path in paths:
                path.write_text(f'{path.name} contents')
            original = artifacts.fingerprint_files(paths)

            for path in paths:
                original_contents = path.read_text()
                path.write_text(f'{original_contents} changed')
                self.assertNotEqual(artifacts.fingerprint_files(paths), original)
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
            )
        )

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
