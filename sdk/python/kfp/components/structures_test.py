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
"""Tests for kfp.components.structures."""

import os
import tempfile
import textwrap
import unittest

from absl.testing import parameterized
from kfp.components import structures

V1_YAML_IF_PLACEHOLDER = textwrap.dedent("""\
    implementation:
      container:
        args:
        - if:
            cond:
              isPresent: optional_input_1
            else:
              - --arg2
              - default
            then:
              - --arg1
              - {inputUri: optional_input_1}
        image: alpine
    inputs:
    - {name: optional_input_1, optional: true, type: String}
    name: component_if
    """)

V2_YAML_IF_PLACEHOLDER = textwrap.dedent("""\
    implementation:
      container:
        args:
        - ifPresent:
            else: [--arg2, default]
            inputName: optional_input_1
            then:
            - --arg1
            - {inputUri: optional_input_1}
        image: alpine
    inputs:
      optional_input_1: {default: null, type: String}
    name: component_if
    """)

V2_COMPONENT_SPEC_IF_PLACEHOLDER = structures.ComponentSpec(
    name='component_if',
    implementation=structures.Implementation(
        container=structures.ContainerSpec(
            image='alpine',
            args=[
                structures.IfPresentPlaceholder(
                    if_structure=structures.IfPresentPlaceholderStructure(
                        input_name='optional_input_1',
                        then=[
                            '--arg1',
                            structures.InputUriPlaceholder(
                                input_name='optional_input_1'),
                        ],
                        otherwise=[
                            '--arg2',
                            'default',
                        ]))
            ])),
    inputs={
        'optional_input_1': structures.InputSpec(type='String', default=None)
    },
)

V1_YAML_CONCAT_PLACEHOLDER = textwrap.dedent("""\
    name: component_concat
    implementation:
      container:
        args:
        - concat: ['--arg1', {inputValue: input_prefix}]
        image: alpine
    inputs:
    - {name: input_prefix, type: String}
    """)

V2_YAML_CONCAT_PLACEHOLDER = textwrap.dedent("""\
    implementation:
      container:
        args:
        - concat:
          - --arg1
          - {inputValue: input_prefix}
        image: alpine
    inputs:
      input_prefix: {type: String}
    name: component_concat
    """)

V2_COMPONENT_SPEC_CONCAT_PLACEHOLDER = structures.ComponentSpec(
    name='component_concat',
    implementation=structures.Implementation(
        container=structures.ContainerSpec(
            image='alpine',
            args=[
                structures.ConcatPlaceholder(items=[
                    '--arg1',
                    structures.InputValuePlaceholder(input_name='input_prefix'),
                ])
            ])),
    inputs={'input_prefix': structures.InputSpec(type='String')},
)

V2_YAML_NESTED_PLACEHOLDER = textwrap.dedent("""\
    implementation:
      container:
        args:
        - concat:
          - --arg1
          - ifPresent:
              else:
              - --arg2
              - default
              - concat:
                - --arg1
                - {inputValue: input_prefix}
              inputName: input_prefix
              then:
              - --arg1
              - {inputValue: input_prefix}
        image: alpine
    inputs:
      input_prefix: {type: String}
    name: component_nested
    """)

V2_COMPONENT_SPEC_NESTED_PLACEHOLDER = structures.ComponentSpec(
    name='component_nested',
    implementation=structures.Implementation(
        container=structures.ContainerSpec(
            image='alpine',
            args=[
                structures.ConcatPlaceholder(items=[
                    '--arg1',
                    structures.IfPresentPlaceholder(
                        if_structure=structures.IfPresentPlaceholderStructure(
                            input_name='input_prefix',
                            then=[
                                '--arg1',
                                structures.InputValuePlaceholder(
                                    input_name='input_prefix'),
                            ],
                            otherwise=[
                                '--arg2',
                                'default',
                                structures.ConcatPlaceholder(items=[
                                    '--arg1',
                                    structures.InputValuePlaceholder(
                                        input_name='input_prefix'),
                                ]),
                            ])),
                ])
            ])),
    inputs={'input_prefix': structures.InputSpec(type='String')},
)


class StructuresTest(parameterized.TestCase):

    def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
            self):
        with self.assertRaisesRegex(
                ValueError,
                r'^Argument \"InputValuePlaceholder[\s\S]*\'input000\'[\s\S]*references non-existing input.'
        ):
            structures.ComponentSpec(
                name='component_1',
                implementation=structures.Implementation(
                    container=structures.ContainerSpec(
                        image='alpine',
                        command=[
                            'sh',
                            '-c',
                            'set -ex\necho "$0" > "$1"',
                            structures.InputValuePlaceholder(
                                input_name='input000'),
                            structures.OutputPathPlaceholder(
                                output_name='output1'),
                        ],
                    )),
                inputs={'input1': structures.InputSpec(type='String')},
                outputs={'output1': structures.OutputSpec(type='String')},
            )

        with self.assertRaisesRegex(
                ValueError,
                r'^Argument \"OutputPathPlaceholder[\s\S]*\'output000\'[\s\S]*references non-existing output.'
        ):
            structures.ComponentSpec(
                name='component_1',
                implementation=structures.Implementation(
                    container=structures.ContainerSpec(
                        image='alpine',
                        command=[
                            'sh',
                            '-c',
                            'set -ex\necho "$0" > "$1"',
                            structures.InputValuePlaceholder(
                                input_name='input1'),
                            structures.OutputPathPlaceholder(
                                output_name='output000'),
                        ],
                    )),
                inputs={'input1': structures.InputSpec(type='String')},
                outputs={'output1': structures.OutputSpec(type='String')},
            )

    def test_simple_component_spec_save_to_component_yaml(self):
        # tests writing old style (less verbose) and reading in new style (more verbose)
        with tempfile.TemporaryDirectory() as tempdir:
            output_path = os.path.join(tempdir, 'component.yaml')
            original_component_spec = structures.ComponentSpec(
                name='component_1',
                implementation=structures.Implementation(
                    container=structures.ContainerSpec(
                        image='alpine',
                        command=[
                            'sh',
                            '-c',
                            'set -ex\necho "$0" > "$1"',
                            structures.InputValuePlaceholder(
                                input_name='input1'),
                            structures.OutputPathPlaceholder(
                                output_name='output1'),
                        ],
                    )),
                inputs={'input1': structures.InputSpec(type='String')},
                outputs={'output1': structures.OutputSpec(type='String')},
            )
            original_component_spec.save_to_component_yaml(output_path)

            # test that it can be read back correctly
            with open(output_path, 'r') as f:
                new_component_spec = structures.ComponentSpec.load_from_component_yaml(
                    f.read())

        self.assertEqual(original_component_spec, new_component_spec)

    @parameterized.parameters(
        {
            'expected_yaml': V2_YAML_IF_PLACEHOLDER,
            'component': V2_COMPONENT_SPEC_IF_PLACEHOLDER
        },
        {
            'expected_yaml': V2_YAML_CONCAT_PLACEHOLDER,
            'component': V2_COMPONENT_SPEC_CONCAT_PLACEHOLDER
        },
        {
            'expected_yaml': V2_YAML_NESTED_PLACEHOLDER,
            'component': V2_COMPONENT_SPEC_NESTED_PLACEHOLDER
        },
    )
    def test_component_spec_placeholder_save_to_component_yaml(
            self, expected_yaml, component):
        with tempfile.TemporaryDirectory() as tempdir:
            output_path = os.path.join(tempdir, 'component.yaml')
            component.save_to_component_yaml(output_path)
            with open(output_path, 'r') as f:
                contents = f.read()

        # test that what was written can be reloaded correctly
        new_component_spec = structures.ComponentSpec.load_from_component_yaml(
            contents)
        self.assertEqual(new_component_spec, component)

    def test_simple_component_spec_load_from_v2_component_yaml(self):
        component_yaml_v2 = textwrap.dedent("""\
        name: component_1
        inputs:
          input1:
            type: String
        outputs:
          output1:
            type: String
        implementation:
          container:
            image: alpine
            command:
            - sh
            - -c
            - 'set -ex

                echo "$0" > "$1"'
            - inputValue: input1
            - outputPath: output1
        """)

        generated_spec = structures.ComponentSpec.load_from_component_yaml(
            component_yaml_v2)

        expected_spec = structures.ComponentSpec(
            name='component_1',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        structures.InputValuePlaceholder(input_name='input1'),
                        structures.OutputPathPlaceholder(output_name='output1'),
                    ],
                )),
            inputs={'input1': structures.InputSpec(type='String')},
            outputs={'output1': structures.OutputSpec(type='String')})
        self.assertEqual(generated_spec, expected_spec)

    @parameterized.parameters(
        {
            'yaml': V1_YAML_IF_PLACEHOLDER,
            'expected_component': V2_COMPONENT_SPEC_IF_PLACEHOLDER
        },
        {
            'yaml': V2_YAML_IF_PLACEHOLDER,
            'expected_component': V2_COMPONENT_SPEC_IF_PLACEHOLDER
        },
        {
            'yaml': V1_YAML_CONCAT_PLACEHOLDER,
            'expected_component': V2_COMPONENT_SPEC_CONCAT_PLACEHOLDER
        },
        {
            'yaml': V2_YAML_CONCAT_PLACEHOLDER,
            'expected_component': V2_COMPONENT_SPEC_CONCAT_PLACEHOLDER
        },
        {
            'yaml': V2_YAML_NESTED_PLACEHOLDER,
            'expected_component': V2_COMPONENT_SPEC_NESTED_PLACEHOLDER
        },
    )
    def test_component_spec_placeholder_load_from_v2_component_yaml(
            self, yaml, expected_component):
        generated_spec = structures.ComponentSpec.load_from_component_yaml(yaml)
        self.assertEqual(generated_spec, expected_component)

    def test_component_spec_load_from_v1_component_yaml(self):
        component_yaml_v1 = textwrap.dedent("""\
        name: Component with 2 inputs and 2 outputs
        inputs:
        - {name: Input parameter, type: String}
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

        generated_spec = structures.ComponentSpec.load_from_component_yaml(
            component_yaml_v1)

        expected_spec = structures.ComponentSpec(
            name='Component with 2 inputs and 2 outputs',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='busybox',
                    command=[
                        'sh',
                        '-c',
                        (' mkdir -p $(dirname "$2") mkdir -p $(dirname "$3") '
                         'echo "$0" > "$2" cp "$1" "$3" '),
                    ],
                    args=[
                        structures.InputValuePlaceholder(
                            input_name='input_parameter'),
                        structures.InputPathPlaceholder(
                            input_name='input_artifact'),
                        structures.OutputPathPlaceholder(
                            output_name='output_1'),
                        structures.OutputPathPlaceholder(
                            output_name='output_2'),
                    ],
                    env={},
                )),
            inputs={
                'input_parameter': structures.InputSpec(type='String'),
                'input_artifact': structures.InputSpec(type='Artifact')
            },
            outputs={
                'output_1': structures.OutputSpec(type='Artifact'),
                'output_2': structures.OutputSpec(type='Artifact'),
            })
        self.assertEqual(generated_spec, expected_spec)


class TestValidators(unittest.TestCase):

    def test_IfPresentPlaceholderStructure_otherwise(self):
        obj = structures.IfPresentPlaceholderStructure(
            then='then', input_name='input_name', otherwise=['something'])
        self.assertEqual(obj.otherwise, ['something'])

        obj = structures.IfPresentPlaceholderStructure(
            then='then', input_name='input_name', otherwise=[])
        self.assertEqual(obj.otherwise, None)

    def test_ContainerSpec_command_and_args(self):
        obj = structures.ContainerSpec(
            image='image', command=['command'], args=['args'])
        self.assertEqual(obj.command, ['command'])
        self.assertEqual(obj.args, ['args'])

        obj = structures.ContainerSpec(image='image', command=[], args=[])
        self.assertEqual(obj.command, None)
        self.assertEqual(obj.args, None)

    def test_ContainerSpec_env(self):
        obj = structures.ContainerSpec(
            image='image',
            command=['command'],
            args=['args'],
            env={'env': 'env'})
        self.assertEqual(obj.env, {'env': 'env'})

        obj = structures.ContainerSpec(
            image='image', command=[], args=[], env={})
        self.assertEqual(obj.env, None)

    def test_ComponentSpec_inputs(self):
        obj = structures.ComponentSpec(
            name='name',
            implementation=structures.Implementation(container=None),
            inputs={})
        self.assertEqual(obj.inputs, None)

    def test_ComponentSpec_outputs(self):
        obj = structures.ComponentSpec(
            name='name',
            implementation=structures.Implementation(container=None),
            outputs={})
        self.assertEqual(obj.outputs, None)


if __name__ == '__main__':
    unittest.main()
