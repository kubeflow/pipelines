#!/usr/bin/env python3

import io
import unittest
from unittest import mock

import prune_runtime_base_image_caches as pruner


CURRENT_KEY = 'runtime-base-images-list-hash-2026-07-15'
MASTER_REF = 'refs/heads/master'


def cache(cache_id: int, key: str, ref: str) -> dict[str, object]:
    return {'id': cache_id, 'key': key, 'ref': ref}


class CacheSelectionTest(unittest.TestCase):

    def test_preserves_only_current_master_generation(self):
        pages = [{
            'actions_caches': [
                cache(1, CURRENT_KEY, MASTER_REF),
                cache(2, 'runtime-base-images-list-hash-2026-07-14', MASTER_REF),
                cache(3, CURRENT_KEY, 'refs/pull/13640/merge'),
                cache(4, 'runtime-base-images-old-hash-2026-07-14', 'refs/pull/1/merge'),
                cache(5, 'kind-node-Linux-v1.36.1', MASTER_REF),
            ]
        }]

        self.assertEqual(
            pruner.cache_ids_to_delete(pages, CURRENT_KEY, MASTER_REF),
            [2, 3, 4],
        )

    def test_combines_pages_and_deduplicates_ids(self):
        pages = [
            {
                'actions_caches': [
                    cache(1, CURRENT_KEY, MASTER_REF),
                    cache(2, 'runtime-base-images-old-1', MASTER_REF),
                ]
            },
            {
                'actions_caches': [
                    cache(2, 'runtime-base-images-old-1', MASTER_REF),
                    cache(3, 'runtime-base-images-old-2', MASTER_REF),
                ]
            },
        ]

        self.assertEqual(
            pruner.cache_ids_to_delete(pages, CURRENT_KEY, MASTER_REF),
            [2, 3],
        )

    def test_refuses_to_delete_when_current_master_cache_is_missing(self):
        pages = [{
            'actions_caches': [
                cache(2, 'runtime-base-images-old', MASTER_REF),
                cache(3, CURRENT_KEY, 'refs/pull/13640/merge'),
            ]
        }]

        with self.assertRaisesRegex(ValueError, 'was not found'):
            pruner.cache_ids_to_delete(pages, CURRENT_KEY, MASTER_REF)

    def test_accepts_single_page_response(self):
        page = {'actions_caches': [cache(1, CURRENT_KEY, MASTER_REF)]}

        self.assertEqual(
            pruner.cache_ids_to_delete(page, CURRENT_KEY, MASTER_REF),
            [],
        )


class MainTest(unittest.TestCase):

    @mock.patch('sys.argv', [
        'prune_runtime_base_image_caches.py',
        '--current-key',
        CURRENT_KEY,
        '--current-ref',
        MASTER_REF,
    ])
    @mock.patch('sys.stdin', io.StringIO('{"actions_caches": []}'))
    def test_main_fails_without_emitting_cache_ids(self):
        output = io.StringIO()
        errors = io.StringIO()
        with mock.patch('sys.stdout', output), mock.patch('sys.stderr', errors):
            exit_code = pruner.main()

        self.assertEqual(exit_code, 1)
        self.assertEqual(output.getvalue(), '')
        self.assertIn('was not found', errors.getvalue())


if __name__ == '__main__':
    unittest.main()
