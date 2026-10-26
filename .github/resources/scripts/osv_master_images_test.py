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
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from arm64_smoke import IMAGES
from osv_master_images import (
    merge_published_images,
    resolve_producer,
    select_producer,
)

SHA = 'a' * 40
REPOSITORY = 'kubeflow/pipelines'


def producer(number=2):
    return {
        'id': 100 + number,
        'run_number': number,
        'run_attempt': 1,
        'path': '.github/workflows/image-builds-master.yml',
        'event': 'push',
        'head_branch': 'master',
        'head_sha': SHA,
        'status': 'completed',
        'conclusion': 'success',
        'repository': {'full_name': REPOSITORY},
        'head_repository': {'full_name': REPOSITORY},
    }


class OsvMasterImagesTest(unittest.TestCase):
    def test_selects_newest_producer_not_latest_completion(self):
        self.assertEqual(
            select_producer([producer(2), producer(1)], REPOSITORY), producer(2)
        )
        self.assertEqual(
            select_producer([producer(1), producer(2)], REPOSITORY), producer(2)
        )

    def test_stale_completion_wakes_scan_of_latest_publication(self):
        self.assertEqual(
            select_producer([producer(2)], REPOSITORY, producer(1)), producer(2)
        )

    def test_same_run_older_attempt_resolves_latest_attempt(self):
        latest = producer()
        latest['run_attempt'] = 2
        self.assertEqual(
            select_producer([latest], REPOSITORY, producer()), latest
        )
        self.assertEqual(select_producer([latest], REPOSITORY, latest), latest)

    def test_selection_rejects_untrusted_or_incomplete_provenance(self):
        for key, value in (
            ('path', '.github/workflows/other.yml'),
            ('event', 'pull_request'),
            ('head_branch', 'release-2.17'),
            ('head_sha', 'short'),
            ('status', 'in_progress'),
            ('conclusion', 'failure'),
            ('repository', {'full_name': 'other/pipelines'}),
            ('head_repository', {'full_name': 'fork/pipelines'}),
            ('run_attempt', 0),
            ('id', 0),
            ('run_number', 0),
        ):
            with self.subTest(key=key):
                bad = producer()
                bad[key] = value
                with self.assertRaises(ValueError):
                    select_producer([bad], REPOSITORY)
                with self.assertRaises(ValueError):
                    select_producer([producer()], REPOSITORY, bad)

    def test_empty_successful_history_fails(self):
        with self.assertRaisesRegex(ValueError, 'No successful'):
            select_producer([], REPOSITORY)

    @mock.patch('osv_master_images.github_api')
    def test_resolver_rechecks_selected_attempt(self, api):
        selected = producer()
        api.side_effect = [{'workflow_runs': [selected]}, selected]
        self.assertEqual(resolve_producer(REPOSITORY), selected)
        self.assertIn(
            'branch=master&event=push&status=success',
            api.call_args_list[0].args[0],
        )
        self.assertEqual(
            api.call_args_list[1].args[0],
            'repos/kubeflow/pipelines/actions/runs/102',
        )

    @mock.patch('osv_master_images.github_api')
    def test_resolver_fails_if_attempt_changes_during_selection(self, api):
        current = producer()
        current['run_attempt'] = 2
        api.side_effect = [{'workflow_runs': [producer()]}, current]
        with self.assertRaisesRegex(ValueError, 'Producer changed'):
            resolve_producer(REPOSITORY)

    @mock.patch('osv_master_images.github_api')
    def test_late_completion_selects_latest_records_even_if_newer_wakeup_was_replaced(
        self, api
    ):
        api.side_effect = [{'workflow_runs': [producer(2)]}, producer(2)]
        self.assertEqual(resolve_producer(REPOSITORY, producer(1)), producer(2))
        self.assertEqual(
            api.call_args_list[1].args[0],
            'repos/kubeflow/pipelines/actions/runs/102',
        )

    def make_records(self, directory):
        for name in IMAGES:
            image = f'ghcr.io/kubeflow/{name}'
            record = {
                'image': image,
                'source_sha': SHA,
                'platforms': ['linux/amd64', 'linux/arm64'],
                'reference': image + '@sha256:' + 'b' * 64,
            }
            (directory / f'{name}.json').write_text(json.dumps(record))

    def test_replaces_only_known_first_party_refs_and_adds_runtime_images(self):
        with tempfile.TemporaryDirectory() as directory:
            records = Path(directory)
            self.make_records(records)
            third_party = {
                'mysql:8.4',
                'other.example/kfp-api-server:1',
                'ghcr.io/kubeflow/unrelated:1',
            }
            images = third_party | {'ghcr.io/kubeflow/kfp-api-server:2.17.1'}
            merged = merge_published_images(images, records, SHA, 'kubeflow')
            self.assertEqual(
                merged - third_party,
                {
                    f'ghcr.io/kubeflow/{name}@sha256:' + 'b' * 64
                    for name in IMAGES
                },
            )
            self.assertEqual(len(merged), len(third_party) + len(IMAGES))

    def test_records_fail_closed_without_exact_inventory_and_provenance(self):
        for mutation in (
            'missing',
            'wrong_sha',
            'wrong_owner',
            'tag',
            'platform',
        ):
            with (
                self.subTest(mutation=mutation),
                tempfile.TemporaryDirectory() as directory,
            ):
                records = Path(directory)
                self.make_records(records)
                path = records / 'kfp-api-server.json'
                record = json.loads(path.read_text())
                if mutation == 'missing':
                    path.unlink()
                else:
                    if mutation == 'wrong_sha':
                        record['source_sha'] = 'c' * 40
                    if mutation == 'wrong_owner':
                        record['image'] = 'ghcr.io/attacker/kfp-api-server'
                        record['reference'] = (
                            record['image'] + '@sha256:' + 'b' * 64
                        )
                    if mutation == 'tag':
                        record['reference'] = record['image'] + ':master'
                    if mutation == 'platform':
                        record['platforms'] = ['linux/amd64']
                    path.write_text(json.dumps(record))
                with self.assertRaises(ValueError):
                    merge_published_images(set(), records, SHA, 'kubeflow')


if __name__ == '__main__':
    unittest.main()
