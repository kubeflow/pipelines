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

    def test_put_and_get_task_status_success(self):
        from kfp.local import status
        store = io.IOStore()
        store.put_task_status('my-task', status.Status.SUCCESS)
        self.assertEqual(
            store.get_task_status('my-task'),
            status.Status.SUCCESS,
        )

    def test_put_and_get_task_status_failure(self):
        from kfp.local import status
        store = io.IOStore()
        store.put_task_status('my-task', status.Status.FAILURE, 'Task failed')
        self.assertEqual(
            store.get_task_status('my-task'),
            status.Status.FAILURE,
        )
        self.assertEqual(
            store.get_task_error('my-task'),
            'Task failed',
        )

    def test_task_status_not_found(self):
        store = io.IOStore()
        with self.assertRaisesRegex(
                ValueError,
                r"Task 'my-task' status not found\."):
            store.get_task_status('my-task')

    def test_get_task_error_returns_none_when_no_error(self):
        from kfp.local import status
        store = io.IOStore()
        store.put_task_status('my-task', status.Status.SUCCESS)
        self.assertIsNone(store.get_task_error('my-task'))

    def test_get_task_error_returns_none_for_unknown_task(self):
        store = io.IOStore()
        self.assertIsNone(store.get_task_error('unknown-task'))


if __name__ == '__main__':
    unittest.main()
