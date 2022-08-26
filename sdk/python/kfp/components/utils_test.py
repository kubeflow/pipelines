# Copyright 2020 The Kubeflow Authors
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
"""Tests for kfp.components.utils."""

import unittest

from absl.testing import parameterized
from kfp.components import utils


class UtilsTest(parameterized.TestCase):

    @parameterized.parameters(
        {
            'original': 'name',
            'expected': 'name',
        },
        {
            'original': ' Some name',
            'expected': 'some-name',
        },
        {
            'original': 'name_123*',
            'expected': 'name-123',
        },
    )
    def test_maybe_rename_for_k8s(self, original, expected):
        self.assertEqual(utils.maybe_rename_for_k8s(original), expected)

    def test_sanitize_component_name(self):
        self.assertEqual('comp-my-component',
                         utils.sanitize_component_name('My component'))

    def test_sanitize_executor_label(self):
        self.assertEqual('exec-my-component',
                         utils.sanitize_executor_label('My component'))

    def test_sanitize_task_name(self):
        self.assertEqual('my-component-1',
                         utils.sanitize_task_name('My component 1'))

    @parameterized.parameters(
        {
            'name': 'some-name',
            'collection': [],
            'delimiter': '-',
            'expected': 'some-name'
        },
        {
            'name': 'some-name',
            'collection': ['some-name'],
            'delimiter': '+',
            'expected': 'some-name+2'
        },
        {
            'name': 'some-name',
            'collection': ['some-name', 'some-name-2'],
            'delimiter': '-',
            'expected': 'some-name-3'
        },
        {
            'name': 'some-name-2',
            'collection': ['some-name', 'some-name-2'],
            'delimiter': '-',
            'expected': 'some-name-2-2'
        },
    )
    def test_make_name_unique_by_adding_index(self, name, collection, delimiter,
                                              expected):
        self.assertEqual(
            expected,
            utils.make_name_unique_by_adding_index(
                name=name,
                collection=collection,
                delimiter=delimiter,
            ))

    @parameterized.parameters(
        {
            'pipeline_name': 'my-pipeline',
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 128,
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 129,
            'is_valid': False,
        },
        {
            'pipeline_name': 'my_pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': '-my-pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': 'My pipeline',
            'is_valid': False,
        },
    )
    def test(self, pipeline_name, is_valid):

        if is_valid:
            utils.validate_pipeline_name(pipeline_name)
        else:
            with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
                utils.validate_pipeline_name('my_pipeline')


if __name__ == '__main__':
    unittest.main()
