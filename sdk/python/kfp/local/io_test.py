# Copyright 2024 The Kubeflow Authors
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
"""Tests for io.py."""

import unittest

from kfp import dsl
from kfp.local import io


class IOStoreTest(unittest.TestCase):

    def test_task_not_found(self):
        store = io.IOStore()
        with self.assertRaisesRegex(
                ValueError,
                r"Tried to get output 'foo' from task 'my-task', but task 'my-task' not found\."
        ):
            store.get_task_output('my-task', 'foo')

    def test_output_not_found(self):
        store = io.IOStore()
        store.put_task_output('my-task', 'bar', 'baz')
        with self.assertRaisesRegex(
                ValueError,
                r"Tried to get output 'foo' from task 'my-task', but task 'my-task' has no output named 'foo'\."
        ):
            store.get_task_output('my-task', 'foo')

    def test_parent_input_not_found(self):
        store = io.IOStore()
        with self.assertRaisesRegex(
                ValueError, r"Parent pipeline input argument 'foo' not found."):
            store.get_parent_input('foo')

    def test_put_and_get_task_output(self):
        store = io.IOStore()
        store.put_task_output('my-task', 'foo', 'bar')
        store.put_task_output('my-task', 'baz', 'bat')
        self.assertEqual(
            store.get_task_output('my-task', 'foo'),
            'bar',
        )
        self.assertEqual(
            store.get_task_output('my-task', 'baz'),
            'bat',
        )
        # test getting doesn't remove by getting twice
        self.assertEqual(
            store.get_task_output('my-task', 'baz'),
            'bat',
        )

    def test_put_and_get_parent_input(self):
        store = io.IOStore()
        store.put_parent_input('foo', 'bar')
        store.put_parent_input('baz', 'bat')
        self.assertEqual(
            store.get_parent_input('foo'),
            'bar',
        )
        self.assertEqual(
            store.get_parent_input('baz'),
            'bat',
        )
        # test getting doesn't remove by getting twice
        self.assertEqual(
            store.get_parent_input('baz'),
            'bat',
        )

    def test_put_and_get_task_output_with_artifact(self):
        artifact = dsl.Artifact(
            name='foo', uri='/my/uri', metadata={'foo': 'bar'})
        store = io.IOStore()
        store.put_task_output('my-task', 'foo', artifact)
        self.assertEqual(
            store.get_task_output('my-task', 'foo'),
            artifact,
        )


if __name__ == '__main__':
    unittest.main()
