# Copyright 2021-2022 The Kubeflow Authors
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
"""Tests for kfp.components.base_component."""

import unittest
from unittest.mock import patch

from kfp.components import base_component
from kfp.components import pipeline_task
from kfp.components import placeholders
from kfp.components import structures


class TestComponent(base_component.BaseComponent):

    def execute(self, *args, **kwargs):
        pass


component_op = TestComponent(
    component_spec=structures.ComponentSpec(
        name='component_1',
        implementation=structures.Implementation(
            container=structures.ContainerSpecImplementation(
                image='alpine',
                command=[
                    'sh',
                    '-c',
                    'set -ex\necho "$0" "$1" "$2" > "$3"',
                    placeholders.InputValuePlaceholder(input_name='input1'),
                    placeholders.InputValuePlaceholder(input_name='input2'),
                    placeholders.InputValuePlaceholder(input_name='input3'),
                    placeholders.OutputPathPlaceholder(output_name='output1'),
                ],
            )),
        inputs={
            'input1':
                structures.InputSpec(type='String'),
            'input2':
                structures.InputSpec(type='Integer'),
            'input3':
                structures.InputSpec(type='Float', default=3.14, optional=True),
            'input4':
                structures.InputSpec(
                    type='Optional[Float]', default=None, optional=True),
        },
        outputs={
            'output1': structures.OutputSpec(type='String'),
        },
    ))


class BaseComponentTest(unittest.TestCase):

    @patch.object(pipeline_task, 'create_pipeline_task', autospec=True)
    def test_instantiate_component_with_keyword_arguments(
            self, mock_create_pipeline_task):

        component_op(input1='hello', input2=100, input3=1.23, input4=3.21)

        mock_create_pipeline_task.assert_called_once_with(
            component_spec=component_op.component_spec,
            args={
                'input1': 'hello',
                'input2': 100,
                'input3': 1.23,
                'input4': 3.21,
            })

    @patch.object(pipeline_task, 'create_pipeline_task', autospec=True)
    def test_instantiate_component_omitting_arguments_with_default(
            self, mock_create_pipeline_task):

        component_op(input1='hello', input2=100)

        mock_create_pipeline_task.assert_called_once_with(
            component_spec=component_op.component_spec,
            args={
                'input1': 'hello',
                'input2': 100,
            })

    def test_instantiate_component_with_positional_arugment(self):
        with self.assertRaisesRegex(
                TypeError,
                'Components must be instantiated using keyword arguments.'
                r' Positional parameters are not allowed \(found 3 such'
                r' parameters for component "component-1"\).'):
            component_op('abc', 1, 2.3)

    def test_instantiate_component_with_unexpected_keyword_arugment(self):
        with self.assertRaisesRegex(
                TypeError,
                r'component-1\(\) got an unexpected keyword argument "input0".'
        ):
            component_op(input1='abc', input2=1, input3=2.3, input0='extra')

    def test_instantiate_component_with_missing_arugments(self):
        with self.assertRaisesRegex(
                TypeError,
                r'component-1\(\) missing 1 required argument: input1.'):
            component_op(input2=1)

        with self.assertRaisesRegex(
                TypeError,
                r'component-1\(\) missing 2 required arguments: input1, input2.'
        ):
            component_op()


if __name__ == '__main__':
    unittest.main()
