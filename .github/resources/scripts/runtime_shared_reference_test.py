#!/usr/bin/env python3
"""Validate exact runtime artifact references before cross-run downloads."""

import json
from pathlib import Path
import tempfile
import unittest

import runtime_base_image_artifacts as artifacts


class SharedReferenceTest(unittest.TestCase):

    def candidate(self, **changes):
        candidate = {
            'id': 12,
            'name': 'runtime-base-images-fingerprint',
            'expired': False,
            'created_at': '2026-01-01T00:00:00Z',
            'expires_at': '2099-01-01T00:00:00Z',
            'workflow_run': {
                'id': 34,
                'repository_id': 1,
                'head_repository_id': 1,
                'head_branch': 'master',
                'head_sha': 'old-source'
            },
        }
        candidate.update(changes)
        return candidate

    def eligible(self, candidate):
        return artifacts.eligible_artifact(candidate,
                                           'runtime-base-images-fingerprint',
                                           'current-source', 1)

    def test_trusted_master_and_exact_current_source_are_eligible(self):
        self.assertTrue(self.eligible(self.candidate()))
        candidate = self.candidate()
        candidate['workflow_run'].update(
            head_repository_id=2,
            head_branch='feature',
            head_sha='current-source')
        self.assertTrue(self.eligible(candidate))

    def test_rejects_wrong_fingerprint_expired_and_malformed_metadata(self):
        for change in ({
                'name': 'runtime-base-images-other'
        }, {
                'expired': True
        }, {
                'expires_at': '2020-01-01T00:00:00Z'
        }, {
                'expires_at': 'invalid'
        }, {
                'created_at': 'invalid'
        }, {
                'created_at': '2026-01-01T00:00:00'
        }, {
                'id': 0
        }, {
                'id': -1
        }, {
                'id': True
        }, {
                'id': '12'
        }):
            with self.subTest(change=change):
                self.assertFalse(self.eligible(self.candidate(**change)))

    def test_rejects_cross_repository_and_unrelated_fork_master(self):
        for change in ({
                'repository_id': 2
        }, {
                'head_repository_id': 2
        }, {
                'id': True
        }, {
                'id': 0
        }, {
                'repository_id': True
        }, {
                'head_repository_id': True
        }, {
                'head_repository_id': 0
        }):
            with self.subTest(change=change):
                candidate = self.candidate()
                candidate['workflow_run'].update(change)
                self.assertFalse(self.eligible(candidate))

    def test_selects_exact_newest_artifact_across_pages(self):
        newer = self.candidate(id=14, created_at='2026-02-01T00:00:00Z')
        expired = self.candidate(
            id=15, created_at='2026-03-01T00:00:00Z', expired=True)
        selected = artifacts.select_artifact([{
            'artifacts': [self.candidate()]
        }, {
            'artifacts': [newer, expired]
        }], 'runtime-base-images-fingerprint', 'current-source', 1)
        self.assertEqual(selected['id'], 14)

    def test_rejects_missing_source_or_invalid_repository_identity(self):
        for source, repository in (('', 1), (None, 1), ('current', True),
                                   ('current', 0)):
            self.assertFalse(
                artifacts.eligible_artifact(self.candidate(),
                                            'runtime-base-images-fingerprint',
                                            source, repository))

    def test_selection_compares_timestamp_instants(self):
        earlier = self.candidate(id=15, created_at='2026-02-01T02:00:00+02:00')
        later = self.candidate(id=14, created_at='2026-02-01T01:00:00Z')
        selected = artifacts.select_artifact([{
            'artifacts': [earlier, later]
        }], 'runtime-base-images-fingerprint', 'current-source', 1)
        self.assertEqual(selected['id'], 14)

    def test_reference_accepts_only_a_positive_exact_id(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'reference.json'
            path.write_text('{"artifact_id": 12}')
            self.assertEqual(artifacts.reference_id(path), 12)
            for value in ({
                    'artifact_id': True
            }, {
                    'artifact_id': 0
            }, {
                    'artifact_id': -1
            }, {
                    'artifact_id': '12'
            }, {
                    'artifact_id': 12,
                    'repository': 'attacker/repo'
            }, [], {}):
                with self.subTest(value=value):
                    path.write_text(json.dumps(value))
                    with self.assertRaises(ValueError):
                        artifacts.reference_id(path)


if __name__ == '__main__':
    unittest.main()
