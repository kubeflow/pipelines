#!/usr/bin/env python3

import copy
import json
from pathlib import Path
import subprocess
import sys
import unittest

import image_cache_artifacts as artifacts

NAME = 'image-cache-v1-apiserver-amd64-inputs'


def artifact(artifact_id=1, *, head_repository_id=2, head_branch='feature'):
    return {
        'id': artifact_id,
        'name': NAME,
        'expired': False,
        'created_at': '2026-10-12T00:00:00Z',
        'workflow_run': {
            'id': 100 + artifact_id,
            'repository_id': 1,
            'head_repository_id': head_repository_id,
            'head_branch': head_branch,
        },
    }


class ArtifactSelectionTest(unittest.TestCase):

    def select(self, *candidates):
        return artifacts.select_artifact([{
            'artifacts': list(candidates)
        }], NAME, 2, 'feature')

    def test_selects_exact_artifact_id_across_all_pages(self):
        older = artifact(9)
        newer = artifact(2)
        newer['created_at'] = '2026-10-12T01:00:00Z'
        payload = [{'artifacts': [older]}, {'artifacts': [newer, artifact(99)]}]
        self.assertEqual(
            artifacts.select_artifact(payload, NAME, 2, 'feature'), {
                'artifact_id': 2,
                'run_id': 102
            })

    def test_equal_timestamps_use_highest_artifact_id(self):
        self.assertEqual(
            self.select(artifact(2), artifact(3)), {
                'artifact_id': 3,
                'run_id': 103
            })

    def test_accepts_trusted_master(self):
        self.assertEqual(
            self.select(artifact(head_repository_id=1, head_branch='master')), {
                'artifact_id': 1,
                'run_id': 101
            })

    def test_rejects_other_fork_with_same_branch(self):
        self.assertIsNone(self.select(artifact(head_repository_id=3)))

    def test_rejects_other_branch_in_same_fork(self):
        self.assertIsNone(self.select(artifact(head_branch='another-feature')))

    def test_rejects_fork_named_master(self):
        self.assertIsNone(
            self.select(artifact(head_repository_id=3, head_branch='master')))

    def test_rejects_expired_and_wrong_names(self):
        for field, value in [('expired', True), ('expired', None),
                             ('name', NAME + '-other')]:
            with self.subTest(field=field, value=value):
                candidate = artifact()
                candidate[field] = value
                self.assertIsNone(self.select(candidate))

    def test_rejects_missing_or_malformed_metadata(self):
        for field, values in {
                'id': [True, 0, -1, '1', None],
                'created_at': [None, 123, '2026-99-99T00:00:00Z', 'yesterday'],
                'workflow_run': [None, [], 'run'],
        }.items():
            for value in values:
                with self.subTest(field=field, value=value):
                    candidate = artifact()
                    candidate[field] = value
                    self.assertIsNone(self.select(candidate))
        for field in artifact():
            candidate = artifact()
            del candidate[field]
            with self.subTest(missing=field):
                self.assertIsNone(self.select(candidate))
        self.assertIsNone(self.select(None, [], 'artifact'))

    def test_rejects_invalid_run_ids_including_booleans(self):
        for field in ('id', 'head_repository_id', 'repository_id'):
            for value in (True, False, 0, -1, None, '2'):
                with self.subTest(field=field, value=value):
                    candidate = artifact()
                    candidate['workflow_run'][field] = value
                    self.assertIsNone(self.select(candidate))
            candidate = artifact()
            del candidate['workflow_run'][field]
            self.assertIsNone(self.select(candidate))

    def test_malformed_page_fails_closed_even_after_eligible_page(self):
        for malformed in (None, {}, {'artifacts': None}, {'artifacts': {}}, []):
            with self.subTest(page=malformed):
                with self.assertRaises(ValueError):
                    artifacts.select_artifact([{
                        'artifacts': [artifact()]
                    }, malformed], NAME, 2, 'feature')
        with self.assertRaises(ValueError):
            artifacts.select_artifact({'artifacts': [artifact()]}, NAME, 2,
                                      'feature')

    def test_invalid_current_provenance_fails_closed(self):
        for repository_id, branch in [(True, 'feature'), (0, 'feature'),
                                      (2, '')]:
            with self.assertRaises(ValueError):
                artifacts.select_artifact([], NAME, repository_id, branch)

    def test_does_not_mutate_candidates(self):
        candidate = artifact()
        original = copy.deepcopy(candidate)
        self.select(candidate)
        self.assertEqual(candidate, original)

    def test_cli_returns_exact_id_and_empty_output_on_miss(self):
        for candidates, expected in [([artifact(4)], {
                'artifact_id': 4,
                'run_id': 104
        }), ([], {})]:
            result = self.run_cli(json.dumps([{'artifacts': candidates}]))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(result.stdout), expected)

    def test_cli_invalid_json_has_no_selected_id(self):
        result = self.run_cli('{')
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stdout, '')
        self.assertIn('Cannot select image-cache artifact', result.stderr)

    def run_cli(self, payload):
        return subprocess.run(
            [
                sys.executable,
                str(Path(artifacts.__file__)), '--name', NAME,
                '--head-repository-id', '2', '--head-branch', 'feature'
            ],
            input=payload,
            text=True,
            capture_output=True,
            check=False,
        )


if __name__ == '__main__':
    unittest.main()
