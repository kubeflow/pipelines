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
"""Tests for kfp.dsl.base_component."""

import unittest

from kfp import dsl
from kfp.dsl import placeholders
from kfp.dsl import python_component
from kfp.dsl import structures

component_op = python_component.PythonComponent(
    # dummy python_func not used in behavior that is being tested
    python_func=lambda: None,
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


class BlockPipelineTaskRegistration(unittest.TestCase):

    def test_mutating_decorator(self):

        def call_pipeline_spec(component):
            component.pipeline_spec
            return component

        @dsl.component
        def identity(text: str) -> str:
            return text

        @dsl.pipeline
        def pipeline_custom_job():
            modified_identity = call_pipeline_spec(identity)
            modified_identity(text='text')

        self.assertEqual(len(pipeline_custom_job.pipeline_spec.components), 1)
        self.assertEqual(
            len(pipeline_custom_job.pipeline_spec.deployment_spec), 1)

    def test_call_directly_in_pipeline(self):

        @dsl.component
        def identity(text: str) -> str:
            return text

        @dsl.pipeline
        def pipeline_custom_job():
            identity.pipeline_spec
            identity(text='text')

        self.assertEqual(len(pipeline_custom_job.pipeline_spec.components), 1)
        self.assertEqual(
            len(pipeline_custom_job.pipeline_spec.deployment_spec), 1)


if __name__ == '__main__':
    unittest.main()
