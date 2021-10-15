# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.v2.components.experimental.pipeline_task."""

import textwrap
import unittest

from absl.testing import parameterized
from kfp.v2.components.experimental import pipeline_task
from kfp.v2.components.experimental import structures

V2_YAML = textwrap.dedent("""\
    name: component1
    inputs:
      input1: {type: String}
    outputs:
      output1: {type: Artifact}
    implementation:
      container:
        image: alpine
        commands:
        - sh
        - -c
        - echo "$0" >> "$1"
        arguments:
        - {inputValue: input1}
        - {outputPath: output1}
""")

V2_YAML_IF_PLACEHOLDER = textwrap.dedent("""\
    name: component_if
    inputs:
      optional_input_1: {type: String}
    implementation:
      container:
        image: alpine
        commands:
        - sh
        - -c
        - echo "$0" "$1"
        arguments:
        - ifPresent:
            inputName: optional_input_1
            then:
            - "input: "
            - {inputValue: optional_input_1}
            otherwise:
            - "default: "
            - "Hello world!"
""")

V2_YAML_CONCAT_PLACEHOLDER = textwrap.dedent("""\
    name: component_concat
    inputs:
      input1: {type: String}
      input2: {type: String}
    implementation:
      container:
        image: alpine
        commands:
        - sh
        - -c
        - echo "$0"
        - concat: [{inputValue: input1}, "+", {inputValue: input2}]
    """)


class PipelineTaskTest(parameterized.TestCase):

    def test_create_pipeline_task_valid(self):
        expected_component_spec = structures.ComponentSpec(
            name='component1',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    commands=['sh', '-c', 'echo "$0" >> "$1"'],
                    arguments=[
                        structures.InputValuePlaceholder(input_name='input1'),
                        structures.OutputPathPlaceholder(output_name='output1'),
                    ],
                )),
            inputs={
                'input1': structures.InputSpec(type='String'),
            },
            outputs={
                'output1': structures.OutputSpec(type='Artifact'),
            },
        )
        expected_task_spec = structures.TaskSpec(
            name='component1',
            inputs={'input1': 'value'},
            dependent_tasks=[],
            component_ref='component1',
        )
        expected_container_spec = structures.ContainerSpec(
            image='alpine',
            commands=['sh', '-c', 'echo "$0" >> "$1"'],
            arguments=[
                "{{$.inputs.parameters['input1']}}",
                "{{$.outputs.artifacts['output1'].path}}",
            ],
        )

        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            arguments={'input1': 'value'},
        )
        self.assertEqual(task.task_spec, expected_task_spec)
        self.assertEqual(task.component_spec, expected_component_spec)
        self.assertEqual(task.container_spec, expected_container_spec)

    def test_create_pipeline_task_invalid_missing_required_input(self):
        with self.assertRaisesRegex(ValueError,
                                    'No value provided for input: input1.'):
            task = pipeline_task.PipelineTask(
                component_spec=structures.ComponentSpec
                .load_from_component_yaml(V2_YAML),
                arguments={},
            )

    def test_create_pipeline_task_invalid_wrong_input(self):
        with self.assertRaisesRegex(
                ValueError,
                'Component "component1" got an unexpected input: input0.'):
            task = pipeline_task.PipelineTask(
                component_spec=structures.ComponentSpec
                .load_from_component_yaml(V2_YAML),
                arguments={
                    'input1': 'value',
                    'input0': 'abc',
                },
            )

    @parameterized.parameters(
        {
            'component_yaml':
                V2_YAML_IF_PLACEHOLDER,
            'arguments': {
                'optional_input_1': 'value'
            },
            'expected_container_spec':
                structures.ContainerSpec(
                    image='alpine',
                    commands=['sh', '-c', 'echo "$0" "$1"'],
                    arguments=[
                        'input: ',
                        "{{$.inputs.parameters['optional_input_1']}}",
                    ],
                )
        },
        {
            'component_yaml':
                V2_YAML_IF_PLACEHOLDER,
            'arguments': {},
            'expected_container_spec':
                structures.ContainerSpec(
                    image='alpine',
                    commands=['sh', '-c', 'echo "$0" "$1"'],
                    arguments=[
                        'default: ',
                        'Hello world!',
                    ],
                )
        },
    )
    def test_resolve_if_placeholder(
        self,
        component_yaml: str,
        arguments: dict,
        expected_container_spec: structures.ContainerSpec,
    ):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                component_yaml),
            arguments=arguments,
        )
        self.assertEqual(task.container_spec, expected_container_spec)

    def test_resolve_concat_placeholder(self):
        expected_container_spec = structures.ContainerSpec(
            image='alpine',
            commands=[
                'sh',
                '-c',
                'echo "$0"',
                "{{$.inputs.parameters['input1']}}+{{$.inputs.parameters['input2']}}",
            ],
        )

        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML_CONCAT_PLACEHOLDER),
            arguments={
                'input1': '1',
                'input2': '2',
            },
        )
        self.assertEqual(task.container_spec, expected_container_spec)


if __name__ == '__main__':
    unittest.main()
