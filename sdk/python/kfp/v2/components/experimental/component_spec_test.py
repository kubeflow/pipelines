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
"""Tests for kfp.v2.components.experimental.component_spec."""

from absl.testing import parameterized

import textwrap
import unittest
from unittest import mock
import pydantic

from kfp.v2.components.experimental import component_spec

V1_YAML_IF_PLACEHOLDER = textwrap.dedent("""\
    name: component_1
    inputs:
    - {name: optional_input_1, type: String, optional: true}
    implementation:
      container:
        image: alpine
        args:
        - if:
            cond:
              isPresent: optional_input_1
            then:
              - --arg1
              - {inputValue: optional_input_1}
            else:
              - --arg2
              - default
    """)

V2_YAML_IF_PLACEHOLDER = textwrap.dedent("""\
    name: component_1
    implementation:
      image: alpine
      arguments:
      - if_present:
          name: optional_input_1
          then:
          - --arg1
          - input_value: optional_input_1
          otherwise:
          - --arg2
          - default
    inputs:
      optional_input_1:
        type: String
    schema_version: v2
    """)


def v2_component_spec_if_placeholder(
        schema_version: component_spec.SchemaVersion):
    return component_spec.ComponentSpec(
        name='component_1',
        implementation=component_spec.ContainerSpec(
            image='alpine',
            arguments=[
                component_spec.IfPresentPlaceholder(
                    if_present=component_spec.IfPresentPlaceholderStructure(
                        name='optional_input_1',
                        then=[
                            '--arg1',
                            component_spec.InputValuePlaceholder(
                                input_value='optional_input_1'),
                        ],
                        otherwise=[
                            '--arg2',
                            'default',
                        ]))
            ]),
        inputs={'optional_input_1': component_spec.InputSpec(type='String')},
        schema_version=schema_version,
    )


V1_YAML_CONCAT_PLACEHOLDER = textwrap.dedent("""\
    name: component_concat
    inputs:
    - {name: input_prefix, type: String}
    implementation:
      container:
        image: alpine
        args:
        - concat: ['--arg1', {inputValue: input_prefix}]
    """)

V2_YAML_CONCAT_PLACEHOLDER = textwrap.dedent("""\
    name: component_concat
    implementation:
      image: alpine
      arguments:
      - concat:
        - --arg1
        - input_value: input_prefix
    inputs:
      input_prefix:
        type: String
    schema_version: v2
    """)


def v2_component_spec_concat_placeholder(
        schema_version: component_spec.SchemaVersion):
    return component_spec.ComponentSpec(
        name='component_concat',
        implementation=component_spec.ContainerSpec(
            image='alpine',
            arguments=[
                component_spec.ConcatPlaceholder(concat=[
                    '--arg1',
                    component_spec.InputValuePlaceholder(
                        input_value='input_prefix'),
                ])
            ]),
        inputs={'input_prefix': component_spec.InputSpec(type='String')},
        schema_version=schema_version,
    )


V2_YAML_NESTED_PLACEHOLDER = textwrap.dedent("""\
    name: component_concat
    implementation:
      image: alpine
      arguments:
      - concat:
        - --arg1
        - if_present:
            name: input_prefix
            then:
            - --arg1
            - input_value: input_prefix
            otherwise:
            - --arg2
            - default
            - concat:
              - --arg1
              - input_value: input_prefix
    inputs:
      input_prefix:
        type: String
    schema_version: v2
    """)


def v2_component_spec_nested_placeholder(
        schema_version: component_spec.SchemaVersion):
    component_spec.ConcatPlaceholder.update_forward_refs()
    component_spec.IfPresentPlaceholderStructure.update_forward_refs()

    return component_spec.ComponentSpec(
        name='component_concat',
        implementation=component_spec.ContainerSpec(
            image='alpine',
            arguments=[
                component_spec.ConcatPlaceholder(concat=[
                    '--arg1',
                    component_spec.IfPresentPlaceholder(
                        if_present=component_spec.IfPresentPlaceholderStructure(
                            name='input_prefix',
                            then=[
                                '--arg1',
                                component_spec.InputValuePlaceholder(
                                    input_value='input_prefix'),
                            ],
                            otherwise=[
                                '--arg2',
                                'default',
                                component_spec.ConcatPlaceholder(concat=[
                                    '--arg1',
                                    component_spec.InputValuePlaceholder(
                                        input_value='input_prefix'),
                                ]),
                            ])),
                ])
            ]),
        inputs={'input_prefix': component_spec.InputSpec(type='String')},
        schema_version=schema_version,
    )


class ComponentSpecTest(parameterized.TestCase):

    def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
            self):
        with self.assertRaisesRegex(
                pydantic.ValidationError, 'Argument "input_value=\'input000\'" '
                'references non-existing input.'):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(
                            input_value='input000'),
                        component_spec.OutputPathPlaceholder(
                            output_path='output1'),
                    ],
                ),
                inputs={'input1': component_spec.InputSpec(type='String')},
                outputs={'output1': component_spec.OutputSpec(type='String')},
            )

        with self.assertRaisesRegex(
                pydantic.ValidationError,
                'Argument "output_path=\'output000\'" '
                'references non-existing output.'):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(
                            input_value='input1'),
                        component_spec.OutputPathPlaceholder(
                            output_path='output000'),
                    ],
                ),
                inputs={'input1': component_spec.InputSpec(type='String')},
                outputs={'output1': component_spec.OutputSpec(type='String')},
            )

    def test_simple_component_spec_save_to_component_yaml(self):
        open_mock = mock.mock_open()
        expected_yaml = textwrap.dedent("""\
        name: component_1
        implementation:
          image: alpine
          commands:
          - sh
          - -c
          - 'set -ex

            echo "$0" > "$1"'
          - input_value: input1
          - output_path: output1
        inputs:
          input1:
            type: String
        outputs:
          output1:
            type: String
        schema_version: v2
        """)

        with mock.patch("builtins.open", open_mock, create=True):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(
                            input_value='input1'),
                        component_spec.OutputPathPlaceholder(
                            output_path='output1'),
                    ],
                ),
                inputs={
                    'input1': component_spec.InputSpec(type='String')
                },
                outputs={
                    'output1': component_spec.OutputSpec(type='String')
                },
                schema_version=component_spec.SchemaVersion.V2,
            ).save_to_component_yaml('test_save_file.txt')

        open_mock.assert_called_with('test_save_file.txt', 'a')
        open_mock.return_value.write.assert_called_once_with(expected_yaml)

    @parameterized.parameters(
        {
            'expected_yaml':
                V2_YAML_IF_PLACEHOLDER,
            'component':
                v2_component_spec_if_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
        {
            'expected_yaml':
                V2_YAML_CONCAT_PLACEHOLDER,
            'component':
                v2_component_spec_concat_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
        {
            'expected_yaml':
                V2_YAML_NESTED_PLACEHOLDER,
            'component':
                v2_component_spec_nested_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
    )
    def test_component_spec_placeholder_save_to_component_yaml(
            self, expected_yaml, component):
        open_mock = mock.mock_open()

        with mock.patch("builtins.open", open_mock, create=True):
            component.save_to_component_yaml('test_save_file.txt')

        open_mock.assert_called_with('test_save_file.txt', 'a')
        open_mock.return_value.write.assert_called_once_with(expected_yaml)

    def test_simple_component_spec_load_from_v2_component_yaml(self):
        component_yaml_v2 = textwrap.dedent("""\
        name: component_1
        implementation:
          image: alpine
          commands:
          - sh
          - -c
          - 'set -ex

            echo "$0" > "$1"'
          - input_value: input1
          - output_path: output1
        inputs:
          input1:
            type: String
        outputs:
          output1:
            type: String
        schema_version: v2
        """)

        generated_spec = component_spec.ComponentSpec.load_from_component_yaml(
            component_yaml_v2)

        expected_spec = component_spec.ComponentSpec(
            name='component_1',
            implementation=component_spec.ContainerSpec(
                image='alpine',
                commands=[
                    'sh',
                    '-c',
                    'set -ex\necho "$0" > "$1"',
                    component_spec.InputValuePlaceholder(input_value='input1'),
                    component_spec.OutputPathPlaceholder(output_path='output1'),
                ],
            ),
            inputs={'input1': component_spec.InputSpec(type='String')},
            outputs={'output1': component_spec.OutputSpec(type='String')},
            schema_version=component_spec.SchemaVersion.V2)
        self.assertEqual(generated_spec, expected_spec)

    @parameterized.parameters(
        {
            'yaml':
                V1_YAML_IF_PLACEHOLDER,
            'expected_component':
                v2_component_spec_if_placeholder(
                    schema_version=component_spec.SchemaVersion.V1)
        },
        {
            'yaml':
                V2_YAML_IF_PLACEHOLDER,
            'expected_component':
                v2_component_spec_if_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
        {
            'yaml':
                V1_YAML_CONCAT_PLACEHOLDER,
            'expected_component':
                v2_component_spec_concat_placeholder(
                    schema_version=component_spec.SchemaVersion.V1)
        },
        {
            'yaml':
                V2_YAML_CONCAT_PLACEHOLDER,
            'expected_component':
                v2_component_spec_concat_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
        {
            'yaml':
                V2_YAML_NESTED_PLACEHOLDER,
            'expected_component':
                v2_component_spec_nested_placeholder(
                    schema_version=component_spec.SchemaVersion.V2)
        },
    )
    def test_component_spec_placeholder_load_from_v2_component_yaml(
            self, yaml, expected_component):
        generated_spec = component_spec.ComponentSpec.load_from_component_yaml(
            yaml)
        self.assertEqual(generated_spec, expected_component)

    def test_component_spec_load_from_v1_component_yaml(self):
        component_yaml_v1 = textwrap.dedent("""\
        name: Component with 2 inputs and 2 outputs
        inputs:
        - {name: Input parameter}
        - {name: Input artifact}
        outputs:
        - {name: Output 1}
        - {name: Output 2}
        implementation:
          container:
            image: busybox
            command: [sh, -c, '
                mkdir -p $(dirname "$2")
                mkdir -p $(dirname "$3")
                echo "$0" > "$2"
                cp "$1" "$3"
                '
            ]
            args:
            - {inputValue: Input parameter}
            - {inputPath: Input artifact}
            - {outputPath: Output 1}
            - {outputPath: Output 2}
        """)

        generated_spec = component_spec.ComponentSpec.load_from_component_yaml(
            component_yaml_v1)

        expected_spec = component_spec.ComponentSpec(
            name='Component with 2 inputs and 2 outputs',
            implementation=component_spec.ContainerSpec(
                image='busybox',
                commands=[
                    'sh',
                    '-c',
                    (' mkdir -p $(dirname "$2") mkdir -p $(dirname "$3") '
                     'echo "$0" > "$2" cp "$1" "$3" '),
                ],
                arguments=[
                    component_spec.InputValuePlaceholder(
                        input_value='Input parameter'),
                    component_spec.InputPathPlaceholder(
                        input_path='Input artifact'),
                    component_spec.OutputPathPlaceholder(
                        output_path='Output 1'),
                    component_spec.OutputPathPlaceholder(
                        output_path='Output 2'),
                ],
                env={},
            ),
            inputs={
                'Input parameter': component_spec.InputSpec(type='Artifact'),
                'Input artifact': component_spec.InputSpec(type='Artifact')
            },
            outputs={
                'Output 1': component_spec.OutputSpec(type='Artifact'),
                'Output 2': component_spec.OutputSpec(type='Artifact'),
            },
            schema_version=component_spec.SchemaVersion.V1)

        self.assertEqual(generated_spec, expected_spec)


if __name__ == '__main__':
    unittest.main()
