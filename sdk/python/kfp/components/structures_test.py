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
import sys
import tempfile
import textwrap
import unittest

from absl.testing import parameterized
from kfp import compiler
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

COMPONENT_SPEC_IF_PLACEHOLDER = structures.ComponentSpec(
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

COMPONENT_SPEC_CONCAT_PLACEHOLDER = structures.ComponentSpec(
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

V1_YAML_NESTED_PLACEHOLDER = textwrap.dedent("""\
    name: component_nested
    implementation:
      container:
        args:
        - concat:
            - --arg1
            - if:
                cond:
                    isPresent: input_prefix
                else:
                - --arg2
                - default
                - concat:
                    - --arg1
                    - {inputValue: input_prefix}
                then:
                - --arg1
                - {inputValue: input_prefix}
        image: alpine
    inputs:
    - {name: input_prefix, optional: false, type: String}
    """)

COMPONENT_SPEC_NESTED_PLACEHOLDER = structures.ComponentSpec(
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
        original_component_spec = structures.ComponentSpec(
            name='component_1',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        structures.InputValuePlaceholder(input_name='input1'),
                        structures.OutputParameterPlaceholder(
                            output_name='output1'),
                    ],
                )),
            inputs={'input1': structures.InputSpec(type='String')},
            outputs={'output1': structures.OutputSpec(type='String')},
        )
        from kfp.components import yaml_component
        yaml_component = yaml_component.YamlComponent(
            component_spec=original_component_spec)
        with tempfile.TemporaryDirectory() as tempdir:
            output_path = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(yaml_component, output_path)

            # test that it can be read back correctly
            with open(output_path, 'r') as f:
                contents = f.read()
            new_component_spec = structures.ComponentSpec.load_from_component_yaml(
                contents)

        self.assertEqual(original_component_spec, new_component_spec)

    def test_simple_component_spec_load_from_v2_component_yaml(self):
        component_yaml_v2 = textwrap.dedent("""\
components:
  comp-component-1:
    executorLabel: exec-component-1
    inputDefinitions:
      parameters:
        input1:
          parameterType: STRING
    outputDefinitions:
      parameters:
        output1:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-component-1:
      container:
        command:
        - sh
        - -c
        - 'set -ex

          echo "$0" > "$1"'
        - '{{$.inputs.parameters[''input1'']}}'
        - '{{$.outputs.parameters[''output1''].output_file}}'
        image: alpine
pipelineInfo:
  name: component-1
root:
  dag:
    tasks:
      component-1:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-component-1
        inputs:
          parameters:
            input1:
              componentInputParameter: input1
        taskInfo:
          name: component-1
  inputDefinitions:
    parameters:
      input1:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.2
        """)

        generated_spec = structures.ComponentSpec.load_from_component_yaml(
            component_yaml_v2)

        expected_spec = structures.ComponentSpec(
            name='component-1',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        structures.InputValuePlaceholder(input_name='input1'),
                        structures.OutputParameterPlaceholder(
                            output_name='output1'),
                    ],
                )),
            inputs={'input1': structures.InputSpec(type='String')},
            outputs={'output1': structures.OutputSpec(type='String')})

        self.assertEqual(generated_spec, expected_spec)

    @parameterized.parameters(
        {
            'yaml': V1_YAML_IF_PLACEHOLDER,
            'expected_component': COMPONENT_SPEC_IF_PLACEHOLDER
        },
        {
            'yaml': V1_YAML_CONCAT_PLACEHOLDER,
            'expected_component': COMPONENT_SPEC_CONCAT_PLACEHOLDER
        },
        {
            'yaml': V1_YAML_NESTED_PLACEHOLDER,
            'expected_component': COMPONENT_SPEC_NESTED_PLACEHOLDER
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


class TestInputValuePlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.InputValuePlaceholder('input1')
        actual = structure.to_placeholder()
        expected = "{{$.inputs.parameters['input1']}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder_single_quote(self):
        placeholder = "{{$.inputs.parameters['input1']}}"
        expected = structures.InputValuePlaceholder('input1')
        actual = structures.InputValuePlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder_double_single_quote(self):
        placeholder = "{{$.inputs.parameters[''input1'']}}"
        expected = structures.InputValuePlaceholder('input1')
        actual = structures.InputValuePlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder_double_quote(self):
        placeholder = '{{$.inputs.parameters["input1"]}}'
        expected = structures.InputValuePlaceholder('input1')
        actual = structures.InputValuePlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestInputPathPlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.InputPathPlaceholder('input1')
        actual = structure.to_placeholder()
        expected = "{{$.inputs.artifacts['input1'].path}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder(self):
        placeholder = "{{$.inputs.artifacts['input1'].path}}"
        expected = structures.InputPathPlaceholder('input1')
        actual = structures.InputPathPlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestInputUriPlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.InputUriPlaceholder('input1')
        actual = structure.to_placeholder()
        expected = "{{$.inputs.artifacts['input1'].uri}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder(self):
        placeholder = "{{$.inputs.artifacts['input1'].uri}}"
        expected = structures.InputUriPlaceholder('input1')
        actual = structures.InputUriPlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestOutputPathPlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.OutputPathPlaceholder('output1')
        actual = structure.to_placeholder()
        expected = "{{$.outputs.artifacts['output1'].path}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder(self):
        placeholder = "{{$.outputs.artifacts['output1'].path}}"
        expected = structures.OutputPathPlaceholder('output1')
        actual = structures.OutputPathPlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestOutputParameterPlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.OutputParameterPlaceholder('output1')
        actual = structure.to_placeholder()
        expected = "{{$.outputs.parameters['output1'].output_file}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder(self):
        placeholder = "{{$.outputs.parameters['output1'].output_file}}"
        expected = structures.OutputParameterPlaceholder('output1')
        actual = structures.OutputParameterPlaceholder.from_placeholder(
            placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestOutputUriPlaceholder(unittest.TestCase):

    def test_to_placeholder(self):
        structure = structures.OutputUriPlaceholder('output1')
        actual = structure.to_placeholder()
        expected = "{{$.outputs.artifacts['output1'].uri}}"
        self.assertEqual(
            actual,
            expected,
        )

    def test_from_placeholder(self):
        placeholder = "{{$.outputs.artifacts['output1'].uri}}"
        expected = structures.OutputUriPlaceholder('output1')
        actual = structures.OutputUriPlaceholder.from_placeholder(placeholder)
        self.assertEqual(
            actual,
            expected,
        )


class TestIfPresentPlaceholderStructure(unittest.TestCase):

    def test_otherwise(self):
        obj = structures.IfPresentPlaceholderStructure(
            then='then', input_name='input_name', otherwise=['something'])
        self.assertEqual(obj.otherwise, ['something'])

        obj = structures.IfPresentPlaceholderStructure(
            then='then', input_name='input_name', otherwise=[])
        self.assertEqual(obj.otherwise, None)


class TestContainerSpec(unittest.TestCase):

    def test_command_and_args(self):
        obj = structures.ContainerSpec(
            image='image', command=['command'], args=['args'])
        self.assertEqual(obj.command, ['command'])
        self.assertEqual(obj.args, ['args'])

        obj = structures.ContainerSpec(image='image', command=[], args=[])
        self.assertEqual(obj.command, None)
        self.assertEqual(obj.args, None)

    def test_env(self):
        obj = structures.ContainerSpec(
            image='image',
            command=['command'],
            args=['args'],
            env={'env': 'env'})
        self.assertEqual(obj.env, {'env': 'env'})

        obj = structures.ContainerSpec(
            image='image', command=[], args=[], env={})
        self.assertEqual(obj.env, None)

    def test_from_container_dict_no_placeholders(self):
        component_spec = structures.ComponentSpec(
            name='test',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='python:3.7',
                    command=[
                        'sh', '-c',
                        '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location \'kfp==2.0.0-alpha.2\' && "$0" "$@"\n',
                        'sh', '-ec',
                        'program_path=$(mktemp -d)\nprintf "%s" "$0" > "$program_path/ephemeral_component.py"\npython3 -m kfp.components.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"\n',
                        '\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef concat_message(first: str, second: str) -> str:\n    return first + second\n\n'
                    ],
                    args=[
                        '--executor_input', '{{$}}', '--function_to_execute',
                        'concat_message'
                    ],
                    env=None,
                    resources=None),
                graph=None,
                importer=None),
            description=None,
            inputs={
                'first': structures.InputSpec(type='String', default=None),
                'second': structures.InputSpec(type='String', default=None)
            },
            outputs={'Output': structures.OutputSpec(type='String')})
        container_dict = {
            'args': [
                '--executor_input', '{{$}}', '--function_to_execute', 'fail_op'
            ],
            'command': [
                'sh', '-c',
                '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location \'kfp==2.0.0-alpha.2\' && "$0" "$@"\n',
                'sh', '-ec',
                'program_path=$(mktemp -d)\nprintf "%s" "$0" > "$program_path/ephemeral_component.py"\npython3 -m kfp.components.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"\n',
                '\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef fail_op(message: str):\n    """Fails."""\n    import sys\n    print(message)\n    sys.exit(1)\n\n'
            ],
            'image': 'python:3.7'
        }

        loaded_container_spec = structures.ContainerSpec.from_container_dict(
            container_dict)


class TestComponentSpec(unittest.TestCase):

    def test_inputs(self):
        obj = structures.ComponentSpec(
            name='name',
            implementation=structures.Implementation(container=None),
            inputs={})
        self.assertEqual(obj.inputs, None)

    def test_outputs(self):
        obj = structures.ComponentSpec(
            name='name',
            implementation=structures.Implementation(container=None),
            outputs={})
        self.assertEqual(obj.outputs, None)


class TestInputSpec(unittest.TestCase):

    def test_equality(self):
        self.assertEqual(
            structures.InputSpec(type='str', default=None),
            structures.InputSpec(type='str', default=None))
        self.assertNotEqual(
            structures.InputSpec(type='str', default=None),
            structures.InputSpec(type='str', default='test'))
        self.assertEqual(
            structures.InputSpec(type='List', default=None),
            structures.InputSpec(type='typing.List', default=None))
        self.assertEqual(
            structures.InputSpec(type='List', default=None),
            structures.InputSpec(type='typing.List[int]', default=None))
        self.assertEqual(
            structures.InputSpec(type='List'),
            structures.InputSpec(type='typing.List[typing.Dict[str, str]]'))

    def test_optional(self):
        input_spec = structures.InputSpec(type='str', default='test')
        self.assertEqual(input_spec.default, 'test')
        self.assertEqual(input_spec._optional, True)

        input_spec = structures.InputSpec(type='str', default=None)
        self.assertEqual(input_spec.default, None)
        self.assertEqual(input_spec._optional, True)

        input_spec = structures.InputSpec(type='str')
        self.assertEqual(input_spec.default, None)
        self.assertEqual(input_spec._optional, False)

    def test_from_ir_parameter_dict(self):
        parameter_dict = {'parameterType': 'STRING'}
        input_spec = structures.InputSpec.from_ir_parameter_dict(parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, None)

        parameter_dict = {'parameterType': 'NUMBER_INTEGER'}
        input_spec = structures.InputSpec.from_ir_parameter_dict(parameter_dict)
        self.assertEqual(input_spec.type, 'Integer')
        self.assertEqual(input_spec.default, None)

        parameter_dict = {
            'defaultValue': 'default value',
            'parameterType': 'STRING'
        }
        input_spec = structures.InputSpec.from_ir_parameter_dict(parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, 'default value')

        input_spec = structures.InputSpec.from_ir_parameter_dict(parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, 'default value')


class TestOutputSpec(parameterized.TestCase):

    def test_from_ir_parameter_dict(self):
        parameter_dict = {'parameterType': 'STRING'}
        output_spec = structures.OutputSpec.from_ir_parameter_dict(
            parameter_dict)
        self.assertEqual(output_spec.type, 'String')

        artifact_dict = {
            'artifactType': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            }
        }
        output_spec = structures.OutputSpec.from_ir_parameter_dict(
            artifact_dict)
        self.assertEqual(output_spec.type, 'Artifact')


class TestProcessCommandArg(unittest.TestCase):

    def test_string(self):
        arg = 'test'
        struct = structures.maybe_convert_command_arg_to_placeholder(arg)
        self.assertEqual(struct, arg)

    def test_input_value_placeholder(self):
        arg = "{{$.inputs.parameters['input1']}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(arg)
        expected = structures.InputValuePlaceholder(input_name='input1')
        self.assertEqual(actual, expected)

    def test_input_path_placeholder(self):
        arg = "{{$.inputs.artifacts['input1'].path}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(arg)
        expected = structures.InputPathPlaceholder('input1')
        self.assertEqual(actual, expected)

    def test_input_uri_placeholder(self):
        arg = "{{$.inputs.artifacts['input1'].uri}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(arg)
        expected = structures.InputUriPlaceholder('input1')
        self.assertEqual(actual, expected)

    def test_output_path_placeholder(self):
        arg = "{{$.outputs.artifacts['output1'].path}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(arg)
        expected = structures.OutputPathPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_output_uri_placeholder(self):
        placeholder = "{{$.outputs.artifacts['output1'].uri}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(
            placeholder)
        expected = structures.OutputUriPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_output_parameter_placeholder(self):
        placeholder = "{{$.outputs.parameters['output1'].output_file}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(
            placeholder)
        expected = structures.OutputParameterPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_concat_placeholder(self):
        placeholder = "{{$.inputs.parameters[''input1'']}}+{{$.inputs.parameters[''input2'']}}"
        actual = structures.maybe_convert_command_arg_to_placeholder(
            placeholder)
        expected = structures.ConcatPlaceholder(items=[
            structures.InputValuePlaceholder(input_name='input1'), '+',
            structures.InputValuePlaceholder(input_name='input2')
        ])
        self.assertEqual(actual, expected)


V1_YAML = textwrap.dedent("""\
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

COMPILER_CLI_TEST_DATA_DIR = os.path.join(
    os.path.dirname(os.path.dirname(__file__)), 'compiler_cli_tests',
    'test_data')

SUPPORTED_COMPONENTS_COMPILE_TEST_CASES = [
    {
        'file': 'pipeline_with_importer',
        'component_name': 'pass_through_op'
    },
    {
        'file': 'pipeline_with_env',
        'component_name': 'print_env_op'
    },
    {
        'file': 'pipeline_with_loops',
        'component_name': 'args_generator_op'
    },
    {
        'file': 'pipeline_with_loops',
        'component_name': 'print_struct'
    },
    {
        'file': 'pipeline_with_loops',
        'component_name': 'print_text'
    },
    {
        'file': 'v2_component_with_pip_index_urls',
        'component_name': 'component_op'
    },
    {
        'file': 'pipeline_with_condition',
        'component_name': 'flip_coin_op'
    },
    {
        'file': 'pipeline_with_condition',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_task_final_status',
        'component_name': 'fail_op'
    },
    {
        'file': 'pipeline_with_task_final_status',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_loops_and_conditions',
        'component_name': 'args_generator_op'
    },
    {
        'file': 'pipeline_with_loops_and_conditions',
        'component_name': 'flip_coin_op'
    },
    {
        'file': 'pipeline_with_loops_and_conditions',
        'component_name': 'print_struct'
    },
    {
        'file': 'pipeline_with_loops_and_conditions',
        'component_name': 'print_text'
    },
    {
        'file': 'v2_component_with_optional_inputs',
        'component_name': 'component_op'
    },
    {
        'file': 'lightweight_python_functions_v2_with_outputs',
        'component_name': 'add_numbers'
    },
    {
        'file': 'lightweight_python_functions_v2_with_outputs',
        'component_name': 'concat_message'
    },
    {
        'file': 'lightweight_python_functions_v2_with_outputs',
        'component_name': 'output_artifact'
    },
    {
        'file': 'pipeline_with_nested_loops',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_nested_conditions',
        'component_name': 'flip_coin_op'
    },
    {
        'file': 'pipeline_with_nested_conditions',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_params_containing_format',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_params_containing_format',
        'component_name': 'print_op2'
    },
    {
        'file': 'pipeline_with_exit_handler',
        'component_name': 'fail_op'
    },
    {
        'file': 'pipeline_with_exit_handler',
        'component_name': 'print_op'
    },
    {
        'file': 'pipeline_with_placeholders',
        'component_name': 'print_op'
    },
]


class TestReadInComponent(parameterized.TestCase):

    def test_read_v1(self):
        component_spec = structures.ComponentSpec.load_from_component_yaml(
            V1_YAML_IF_PLACEHOLDER)
        self.assertEqual(component_spec.name, 'component-if')
        self.assertEqual(component_spec.implementation.container.image,
                         'alpine')

    @parameterized.parameters(SUPPORTED_COMPONENTS_COMPILE_TEST_CASES)
    def test_read_in_all_v2_components(self, file, component_name):
        try:
            sys.path.insert(0, COMPILER_CLI_TEST_DATA_DIR)
            mod = __import__(file, fromlist=[component_name])
            component = getattr(mod, component_name)
        finally:
            del sys.path[0]

        with tempfile.TemporaryDirectory() as tmpdir:
            component_file = os.path.join(tmpdir, 'component.yaml')
            compiler.Compiler().compile(component, component_file)
            with open(component_file, 'r') as f:
                yaml_str = f.read()
        loaded_component_spec = structures.ComponentSpec.load_from_component_yaml(
            yaml_str)
        self.assertEqual(component.component_spec, loaded_component_spec)

    def test_simple_placeholder(self):
        compiled_yaml = textwrap.dedent("""
components:
  comp-component1:
    executorLabel: exec-component1
    inputDefinitions:
      parameters:
        input1:
          parameterType: STRING
    outputDefinitions:
      artifacts:
        output1:
          artifactType:
            schemaTitle: system.Artifact
            schemaVersion: 0.0.1
deploymentSpec:
  executors:
    exec-component1:
      container:
        args:
        - '{{$.inputs.parameters[''input1'']}}'
        - '{{$.outputs.artifacts[''output1''].path}}'
        command:
        - sh
        - -c
        - echo "$0" >> "$1"
        image: alpine
pipelineInfo:
  name: component1
root:
  dag:
    tasks:
      component1:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-component1
        inputs:
          parameters:
            input1:
              componentInputParameter: input1
        taskInfo:
          name: component1
  inputDefinitions:
    parameters:
      input1:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.2""")
        loaded_component_spec = structures.ComponentSpec.load_from_component_yaml(
            compiled_yaml)
        component_spec = structures.ComponentSpec(
            name='component1',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=['sh', '-c', 'echo "$0" >> "$1"'],
                    args=[
                        structures.InputValuePlaceholder(input_name='input1'),
                        structures.OutputPathPlaceholder(output_name='output1')
                    ],
                    env=None,
                    resources=None),
                graph=None,
                importer=None),
            description=None,
            inputs={
                'input1': structures.InputSpec(type='String', default=None)
            },
            outputs={'output1': structures.OutputSpec(type='Artifact')})
        self.assertEqual(loaded_component_spec, component_spec)

    def test_if_placeholder(self):
        compiled_yaml = textwrap.dedent("""
components:
  comp-if:
    executorLabel: exec-if
    inputDefinitions:
      parameters:
        optional_input_1:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-if:
      container:
        args:
        - 'input: '
        - '{{$.inputs.parameters[''optional_input_1'']}}'
        command:
        - sh
        - -c
        - echo "$0" "$1"
        image: alpine
pipelineInfo:
  name: if
root:
  dag:
    tasks:
      if:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-if
        inputs:
          parameters:
            optional_input_1:
              componentInputParameter: optional_input_1
        taskInfo:
          name: if
  inputDefinitions:
    parameters:
      optional_input_1:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.2""")
        loaded_component_spec = structures.ComponentSpec.load_from_component_yaml(
            compiled_yaml)
        component_spec = structures.ComponentSpec(
            name='if',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=['sh', '-c', 'echo "$0" "$1"'],
                    args=[
                        'input: ',
                        structures.InputValuePlaceholder(
                            input_name='optional_input_1')
                    ],
                    env=None,
                    resources=None),
                graph=None,
                importer=None),
            description=None,
            inputs={
                'optional_input_1':
                    structures.InputSpec(type='String', default=None)
            },
            outputs=None)
        self.assertEqual(loaded_component_spec, component_spec)

    def test_concat_placeholder(self):
        compiled_yaml = textwrap.dedent("""
components:
  comp-concat:
    executorLabel: exec-concat
    inputDefinitions:
      parameters:
        input1:
          parameterType: STRING
        input2:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-concat:
      container:
        command:
        - sh
        - -c
        - echo "$0"
        - '{{$.inputs.parameters[''input1'']}}+{{$.inputs.parameters[''input2'']}}'
        image: alpine
pipelineInfo:
  name: concat
root:
  dag:
    tasks:
      concat:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-concat
        inputs:
          parameters:
            input1:
              componentInputParameter: input1
            input2:
              componentInputParameter: input2
        taskInfo:
          name: concat
  inputDefinitions:
    parameters:
      input1:
        parameterType: STRING
      input2:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.2""")
        loaded_component_spec = structures.ComponentSpec.load_from_component_yaml(
            compiled_yaml)
        component_spec = structures.ComponentSpec(
            name='concat',
            implementation=structures.Implementation(
                container=structures.ContainerSpec(
                    image='alpine',
                    command=[
                        'sh', '-c', 'echo "$0"',
                        structures.ConcatPlaceholder(items=[
                            structures.InputValuePlaceholder(
                                input_name='input1'), '+',
                            structures.InputValuePlaceholder(
                                input_name='input2')
                        ])
                    ],
                    args=None,
                    env=None,
                    resources=None),
                graph=None,
                importer=None),
            description=None,
            inputs={
                'input1': structures.InputSpec(type='String', default=None),
                'input2': structures.InputSpec(type='String', default=None)
            },
            outputs=None)
        self.assertEqual(loaded_component_spec, component_spec)


if __name__ == '__main__':
    unittest.main()
