# Copyright 2023 The Kubeflow Authors
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
"""Tests for local_task.py."""
import unittest

from kfp import dsl
from kfp.local import local_task


class TestLocalTask(unittest.TestCase):

    def test_output_property(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        self.assertEqual(task.output, 1)
        self.assertEqual(task.outputs['Output'], 1)

    def test_outputs_property(self):
        task = local_task.LocalTask(
            outputs={
                'int_output':
                    1,
                'str_output':
                    'foo',
                'dataset_output':
                    dsl.Dataset(
                        name='dataset_output',
                        uri='foo/bar/dataset_output',
                        metadata={'key': 'value'})
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        self.assertEqual(task.outputs['int_output'], 1)
        self.assertEqual(task.outputs['str_output'], 'foo')
        assert_artifacts_equal(
            self,
            task.outputs['dataset_output'],
            dsl.Dataset(
                name='dataset_output',
                uri='foo/bar/dataset_output',
                metadata={'key': 'value'}),
        )

    def test_platform_spec_property(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        with self.assertRaisesRegex(
                NotImplementedError,
                r'Platform-specific features are not supported for local execution\.'
        ):
            task.platform_spec

    def test_name_property(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        self.assertEqual(task.name, 'my-task-name')

    def test_inputs_property(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        self.assertEqual(task.inputs, {'my_int': 1})

    def test_dependent_tasks_property(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        with self.assertRaisesRegex(
                NotImplementedError,
                r'Task has no dependent tasks since it is executed independently\.'
        ):
            task.dependent_tasks

    def test_sampling_of_task_configuration_methods(self):
        task = local_task.LocalTask(
            outputs={
                'Output': 1,
            },
            arguments={'my_int': 1},
            task_name='my-task-name',
        )
        with self.assertRaisesRegex(
                NotImplementedError,
                r"Task configuration methods are not supported for local execution\. Got call to '\.set_caching_options\(\)'\."
        ):
            task.set_caching_options(enable_caching=True)
        with self.assertRaisesRegex(
                NotImplementedError,
                r"Task configuration methods are not supported for local execution\. Got call to '\.set_env_variable\(\)'\."
        ):
            task.set_env_variable(name='foo', value='BAR')
        with self.assertRaisesRegex(
                NotImplementedError,
                r"Task configuration methods are not supported for local execution\. Got call to '\.ignore_upstream_failure\(\)'\."
        ):
            task.ignore_upstream_failure()


def assert_artifacts_equal(
    test_class: unittest.TestCase,
    a1: dsl.Artifact,
    a2: dsl.Artifact,
) -> None:
    test_class.assertEqual(a1.name, a2.name)
    test_class.assertEqual(a1.uri, a2.uri)
    test_class.assertEqual(a1.metadata, a2.metadata)
    test_class.assertEqual(a1.schema_title, a2.schema_title)
    test_class.assertEqual(a1.schema_version, a2.schema_version)
    test_class.assertIsInstance(a1, type(a2))


if __name__ == '__main__':
    unittest.main()
