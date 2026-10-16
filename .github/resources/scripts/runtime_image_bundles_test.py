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
"""Verify runtime bundle completeness and lane selection."""
import unittest

import runtime_image_bundles as bundles


class RuntimeImageBundlesTest(unittest.TestCase):

    def test_inventory_partition_has_no_duplicates(self):
        inventory = (bundles.ROOT / '.github/resources/runtime-base-images.txt'
                    ).read_text().splitlines()
        expected = [
            line.strip()
            for line in inventory
            if line.strip() and not line.strip().startswith('#')
        ]
        actual = [
            image for group in bundles.BUNDLES
            for image in bundles.bundle_images(group)
        ]
        self.assertCountEqual(actual, expected)
        self.assertEqual(len(actual), len(set(actual)))

    def test_seven_core_fixture_images_preserve_digests(self):
        images = bundles.bundle_images('fixtures')
        self.assertEqual(len(images), 7)
        self.assertIn('python:3.11', images)
        self.assertTrue(any('@sha256:' in image for image in images))
        self.assertFalse(
            any('argoproj/' in image or '/mysql:' in image for image in images))

    def test_argo_pairs(self):
        for version in ('v4.0.13', 'v4.1.5'):
            with self.subTest(version=version):
                self.assertCountEqual(
                    bundles.bundle_images('argo-' + version), [
                        'quay.io/argoproj/workflow-controller:' + version,
                        'quay.io/argoproj/argoexec:' + version
                    ])
                self.assertEqual(
                    bundles.selected_bundles(version, 'pgx'),
                    ['fixtures', 'argo-' + version])

    def test_default_argo_uses_version_file(self):
        version = (bundles.ROOT /
                   'third_party/argo/VERSION').read_text().strip()
        self.assertEqual(bundles.selected_bundles(),
                         ['fixtures', 'argo-' + version, 'mysql'])

    def test_database_selection(self):
        for database, mysql in (('', True), ('mysql', True), ('pgx', False)):
            with self.subTest(database=database):
                self.assertEqual(
                    'mysql' in bundles.selected_bundles('v4.1.5', database),
                    mysql)

    def test_modelcar_is_optional_and_built_separately(self):
        self.assertNotIn('modelcar', bundles.selected_bundles('v4.1.5'))
        self.assertEqual(
            bundles.selected_bundles('v4.1.5', modelcar=True),
            ['fixtures', 'argo-v4.1.5', 'mysql', 'modelcar'])
        self.assertEqual(bundles.bundle_images('modelcar'), [])

    def test_custom_argo_falls_back_to_full_inventory(self):
        expected = ['fixtures', 'mysql', 'argo-v4.0.13', 'argo-v4.1.5']
        self.assertCountEqual(
            bundles.selected_bundles('v9.9.9', 'pgx'), expected)
        self.assertCountEqual(
            bundles.selected_bundles('v9.9.9', 'pgx', modelcar=True),
            expected + ['modelcar'])

    def test_unknown_bundle_rejected(self):
        with self.assertRaisesRegex(ValueError, 'Unknown runtime image bundle'):
            bundles.bundle_images('missing')

    def test_unknown_inventory_argo_rejected(self):
        with self.assertRaisesRegex(ValueError, 'no supported bundle'):
            bundles.bundle_images('fixtures',
                                  ['quay.io/argoproj/argoexec:v9.9.9'])

    def test_unknown_database_rejected(self):
        with self.assertRaisesRegex(ValueError, 'Unsupported runtime database'):
            bundles.selected_bundles('v4.1.5', 'unknown')

    def test_comments_and_empty_lines_ignored(self):
        self.assertEqual(
            bundles.bundle_images(
                'fixtures',
                ['# comment', '', '  python:3.11  ', '  # comment']),
            ['python:3.11'])


if __name__ == '__main__':
    unittest.main()
