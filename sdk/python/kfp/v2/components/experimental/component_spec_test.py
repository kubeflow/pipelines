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

import unittest
from unittest.mock import patch, mock_open
from kfp.v2.components.experimental.component_spec import ComponentSpec
from kfp.v2.components.experimental import component_spec
import textwrap


class ComponentSpecTest(unittest.TestCase):

    @unittest.skip("Placeholder check is not completed. ")
    def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
            self):
        with self.assertRaisesRegex(
                ValueError,
                'Argument "InputValuePlaceholder\(name=\'input000\'\)" '
                'references non-existing input.'):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(name='input000'),
                        component_spec.OutputPathPlaceholder(name='output1'),
                    ],
                ),
                input_specs=[
                    component_spec.InputSpec(name='input1', type='String'),
                ],
                output_specs=[
                    component_spec.OutputSpec(name='output1', type='String'),
                ],
            )

        with self.assertRaisesRegex(
                ValueError,
                'Argument "OutputPathPlaceholder\(name=\'output000\'\)" '
                'references non-existing output.'):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(name='input1'),
                        component_spec.OutputPathPlaceholder(name='output000'),
                    ],
                ),
                input_specs=[
                    component_spec.InputSpec(name='input1', type='String'),
                ],
                output_specs=[
                    component_spec.OutputSpec(name='output1', type='String'),
                ],
            )

    def test_component_spec_save_to_component_yaml(self):
        open_mock = mock_open()
        expected_yaml = textwrap.dedent("""\
        implementation:
          commands:
          - sh
          - -c
          - 'set -ex

            echo "$0" > "$1"'
          - name: input1
          - name: output1
          image: alpine
        inputs:
          input1:
            type: String
        name: component_1
        outputs:
          output1:
            type: String
        schema_version: v2
        """)

        with patch("builtins.open", open_mock, create=True):
            component_spec.ComponentSpec(
                name='component_1',
                implementation=component_spec.ContainerSpec(
                    image='alpine',
                    commands=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        component_spec.InputValuePlaceholder(name='input1'),
                        component_spec.OutputPathPlaceholder(name='output1'),
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

    def test_component_spec_load_from_v2_component_yaml(self):
        component_yaml_v2 = textwrap.dedent("""\
        implementation:
          commands:
          - sh
          - -c
          - 'set -ex

            echo "$0" > "$1"'
          - name: input1
          - name: output1
          image: alpine
        inputs:
          input1:
            type: String
        name: component_1
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
                    component_spec.InputValuePlaceholder(name='input1'),
                    component_spec.OutputPathPlaceholder(name='output1'),
                ],
            ),
            inputs={'input1': component_spec.InputSpec(type='String')},
            outputs={'output1': component_spec.OutputSpec(type='String')},
            schema_version=component_spec.SchemaVersion.V2)
        self.assertEqual(generated_spec, expected_spec)

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
                        name='Input parameter'),
                    component_spec.InputPathPlaceholder(name='Input artifact'),
                    component_spec.OutputPathPlaceholder(name='Output 1'),
                    component_spec.OutputPathPlaceholder(name='Output 2'),
                ],
                env={},
            ),
            inputs={
                'Input parameter': component_spec.InputSpec(type='Artifact'),
                'Input artifact': component_spec.InputSpec(type='Artifact')
            },
            outputs={
                'Output 1': component_spec.OutputSpec(type='String'),
                'Output 2': component_spec.OutputSpec(type='String'),
            },
            schema_version=component_spec.SchemaVersion.V1)

        self.assertEqual(generated_spec, expected_spec)


if __name__ == '__main__':
    unittest.main()
