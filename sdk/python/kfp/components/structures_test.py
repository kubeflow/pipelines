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
from kfp import compiler
from kfp import dsl
from kfp.components import component_factory
from kfp.components import placeholders
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
        container=structures.ContainerSpecImplementation(
            image='alpine',
            args=[
                placeholders.IfPresentPlaceholder(
                    input_name='optional_input_1',
                    then=[
                        '--arg1',
                        placeholders.InputUriPlaceholder(
                            input_name='optional_input_1'),
                    ],
                    else_=[
                        '--arg2',
                        'default',
                    ])
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
        container=structures.ContainerSpecImplementation(
            image='alpine',
            args=[
                placeholders.ConcatPlaceholder(items=[
                    '--arg1',
                    placeholders.InputValuePlaceholder(
                        input_name='input_prefix'),
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
                then: {inputValue: input_prefix}
                else: default
        image: alpine
    inputs:
    - {name: input_prefix, optional: false, type: String}
    """)

COMPONENT_SPEC_NESTED_PLACEHOLDER = structures.ComponentSpec(
    name='component_nested',
    implementation=structures.Implementation(
        container=structures.ContainerSpecImplementation(
            image='alpine',
            args=[
                placeholders.ConcatPlaceholder(items=[
                    '--arg1',
                    placeholders.IfPresentPlaceholder(
                        input_name='input_prefix',
                        then=placeholders.InputValuePlaceholder(
                            input_name='input_prefix'),
                        else_='default'),
                ]),
            ])),
    inputs={'input_prefix': structures.InputSpec(type='String')},
)

V1_YAML_EXECUTOR_INPUT_PLACEHOLDER = textwrap.dedent("""\
    name: component_executor_input
    inputs:
    - {name: input, type: String}
    implementation:
      container:
        image: alpine
        command:
        - python
        - -m
        - kfp.containers.entrypoint
        args:
        - --executor_input
        - {executorInput: null}
        - --function_name
        - test_function
    """)

COMPONENT_SPEC_EXECUTOR_INPUT_PLACEHOLDER = structures.ComponentSpec(
    name='component_executor_input',
    implementation=structures.Implementation(
        container=structures.ContainerSpecImplementation(
            image='alpine',
            command=[
                'python',
                '-m',
                'kfp.containers.entrypoint',
            ],
            args=[
                '--executor_input',
                placeholders.ExecutorInputPlaceholder(),
                '--function_name',
                'test_function',
            ])),
    inputs={'input': structures.InputSpec(type='String')},
)


class StructuresTest(parameterized.TestCase):

    def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
            self):
        with self.assertRaisesRegex(
                ValueError,
                r'^Argument "InputValuePlaceholder" references nonexistant input: "input000".'
        ):
            structures.ComponentSpec(
                name='component_1',
                implementation=structures.Implementation(
                    container=structures.ContainerSpecImplementation(
                        image='alpine',
                        command=[
                            'sh',
                            '-c',
                            'set -ex\necho "$0" > "$1"',
                            placeholders.InputValuePlaceholder(
                                input_name='input000'),
                            placeholders.OutputPathPlaceholder(
                                output_name='output1'),
                        ],
                    )),
                inputs={'input1': structures.InputSpec(type='String')},
                outputs={'output1': structures.OutputSpec(type='String')},
            )

        with self.assertRaisesRegex(
                ValueError,
                r'^Argument "OutputPathPlaceholder" references nonexistant output: "output000".'
        ):
            structures.ComponentSpec(
                name='component_1',
                implementation=structures.Implementation(
                    container=structures.ContainerSpecImplementation(
                        image='alpine',
                        command=[
                            'sh',
                            '-c',
                            'set -ex\necho "$0" > "$1"',
                            placeholders.InputValuePlaceholder(
                                input_name='input1'),
                            placeholders.OutputPathPlaceholder(
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
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=[
                        'sh', '-c', 'set -ex\necho "$0" > "$1"',
                        "{{$.inputs.parameters['input1']}}",
                        "{{$.outputs.parameters['output1'].output_file}}"
                    ],
                )),
            inputs={'input1': structures.InputSpec(type='String')},
            outputs={'output1': structures.OutputSpec(type='String')},
        )
        from kfp.components import base_component

        class TestComponent(base_component.BaseComponent):

            def execute(self, **kwargs):
                pass

        test_component = TestComponent(component_spec=original_component_spec)
        with tempfile.TemporaryDirectory() as tempdir:
            output_path = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(test_component, output_path)

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
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=[
                        'sh',
                        '-c',
                        'set -ex\necho "$0" > "$1"',
                        "{{$.inputs.parameters['input1']}}",
                        "{{$.outputs.parameters['output1'].output_file}}",
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
        {
            'yaml': V1_YAML_EXECUTOR_INPUT_PLACEHOLDER,
            'expected_component': COMPONENT_SPEC_EXECUTOR_INPUT_PLACEHOLDER
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
                container=structures.ContainerSpecImplementation(
                    image='busybox',
                    command=[
                        'sh',
                        '-c',
                        (' mkdir -p $(dirname "$2") mkdir -p $(dirname "$3") '
                         'echo "$0" > "$2" cp "$1" "$3" '),
                    ],
                    args=[
                        placeholders.InputValuePlaceholder(
                            input_name='input_parameter'),
                        placeholders.InputPathPlaceholder(
                            input_name='input_artifact'),
                        placeholders.OutputPathPlaceholder(
                            output_name='output_1'),
                        placeholders.OutputPathPlaceholder(
                            output_name='output_2'),
                    ],
                    env={},
                )),
            inputs={
                'input_parameter':
                    structures.InputSpec(type='String'),
                'input_artifact':
                    structures.InputSpec(type='system.Artifact@0.0.1')
            },
            outputs={
                'output_1': structures.OutputSpec(type='system.Artifact@0.0.1'),
                'output_2': structures.OutputSpec(type='system.Artifact@0.0.1'),
            })
        self.assertEqual(generated_spec, expected_spec)


class TestContainerSpecImplementation(unittest.TestCase):

    def test_command_and_args(self):
        obj = structures.ContainerSpecImplementation(
            image='image', command=['command'], args=['args'])
        self.assertEqual(obj.command, ['command'])
        self.assertEqual(obj.args, ['args'])

        obj = structures.ContainerSpecImplementation(
            image='image', command=[], args=[])
        self.assertEqual(obj.command, None)
        self.assertEqual(obj.args, None)

    def test_env(self):
        obj = structures.ContainerSpecImplementation(
            image='image',
            command=['command'],
            args=['args'],
            env={'env': 'env'})
        self.assertEqual(obj.env, {'env': 'env'})

        obj = structures.ContainerSpecImplementation(
            image='image', command=[], args=[], env={})
        self.assertEqual(obj.env, None)

    def test_from_container_dict_no_placeholders(self):
        component_spec = structures.ComponentSpec(
            name='test',
            implementation=structures.Implementation(
                container=structures.ContainerSpecImplementation(
                    image='python:3.7',
                    command=[
                        'sh', '-c',
                        '\nif ! [ -x "$(command -v pip)" ]; then\n    python3 -m ensurepip || python3 -m ensurepip --user || apt-get install python3-pip\nfi\n\nPIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet     --no-warn-script-location \'kfp==2.0.0-alpha.2\' && "$0" "$@"\n',
                        'sh', '-ec',
                        'program_path=$(mktemp -d)\nprintf "%s" "$0" > "$program_path/ephemeral_component.py"\npython3 -m kfp.components.executor_main                         --component_module_path                         "$program_path/ephemeral_component.py"                         "$@"\n',
                        '\nimport kfp\nfrom kfp import dsl\nfrom kfp.dsl import *\nfrom typing import *\n\ndef concat_message(first: str, second: str) -> str:\n    return first + second\n\n'
                    ],
                    args=[
                        '--executor_input',
                        placeholders.ExecutorInputPlaceholder(),
                        '--function_to_execute', 'concat_message'
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

        loaded_container_spec = structures.ContainerSpecImplementation.from_container_dict(
            container_dict)

    def test_raise_error_if_access_artifact_by_itself(self):

        def comp_with_artifact_input(dataset: dsl.Input[dsl.Dataset]):
            return dsl.ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'run.sh'],
                args=[dataset])

        def comp_with_artifact_output(dataset_old: dsl.Output[dsl.Dataset],
                                      dataset_new: dsl.Output[dsl.Dataset],
                                      optional_input: str = 'default'):
            return dsl.ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'run.sh'],
                args=[
                    dsl.IfPresentPlaceholder(
                        input_name='optional_input',
                        then=[dataset_old],
                        else_=[dataset_new])
                ])

        self.assertRaisesRegex(
            ValueError,
            r'Cannot access artifact by itself in the container definition.',
            component_factory.create_container_component_from_func,
            comp_with_artifact_input)
        self.assertRaisesRegex(
            ValueError,
            r'Cannot access artifact by itself in the container definition.',
            component_factory.create_container_component_from_func,
            comp_with_artifact_output)


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
            structures.InputSpec(type='String', default=None),
            structures.InputSpec(type='String', default=None))
        self.assertNotEqual(
            structures.InputSpec(type='String', default=None),
            structures.InputSpec(type='String', default='test', optional=True))
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
        input_spec = structures.InputSpec(
            type='String', default=None, optional=True)
        self.assertEqual(input_spec.default, None)
        self.assertEqual(input_spec.optional, True)

        input_spec = structures.InputSpec(type='String')
        self.assertEqual(input_spec.default, None)
        self.assertEqual(input_spec.optional, False)

    def test_from_ir_component_inputs_dict(self):
        parameter_dict = {'parameterType': 'STRING'}
        input_spec = structures.InputSpec.from_ir_component_inputs_dict(
            parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, None)

        parameter_dict = {'parameterType': 'NUMBER_INTEGER'}
        input_spec = structures.InputSpec.from_ir_component_inputs_dict(
            parameter_dict)
        self.assertEqual(input_spec.type, 'Integer')
        self.assertEqual(input_spec.default, None)

        parameter_dict = {
            'defaultValue': 'default value',
            'parameterType': 'STRING'
        }
        input_spec = structures.InputSpec.from_ir_component_inputs_dict(
            parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, 'default value')

        input_spec = structures.InputSpec.from_ir_component_inputs_dict(
            parameter_dict)
        self.assertEqual(input_spec.type, 'String')
        self.assertEqual(input_spec.default, 'default value')

        artifact_dict = {
            'artifactType': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            }
        }
        input_spec = structures.InputSpec.from_ir_component_inputs_dict(
            artifact_dict)
        self.assertEqual(input_spec.type, 'system.Artifact@0.0.1')

    def test_assert_optional_must_be_true_when_default_is_not_none(self):
        with self.assertRaisesRegex(ValueError,
                                    r'must be True if `default` is not None'):
            input_spec = structures.InputSpec(type='String', default='text')


class TestOutputSpec(parameterized.TestCase):

    def test_from_ir_component_outputs_dict(self):
        parameter_dict = {'parameterType': 'STRING'}
        output_spec = structures.OutputSpec.from_ir_component_outputs_dict(
            parameter_dict)
        self.assertEqual(output_spec.type, 'String')

        artifact_dict = {
            'artifactType': {
                'schemaTitle': 'system.Artifact',
                'schemaVersion': '0.0.1'
            }
        }
        output_spec = structures.OutputSpec.from_ir_component_outputs_dict(
            artifact_dict)
        self.assertEqual(output_spec.type, 'system.Artifact@0.0.1')


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


class TestReadInComponent(parameterized.TestCase):

    def test_read_v1(self):
        component_spec = structures.ComponentSpec.load_from_component_yaml(
            V1_YAML_IF_PLACEHOLDER)
        self.assertEqual(component_spec.name, 'component-if')
        self.assertEqual(component_spec.implementation.container.image,
                         'alpine')

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
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=['sh', '-c', 'echo "$0" >> "$1"'],
                    args=[
                        "{{$.inputs.parameters['input1']}}",
                        "{{$.outputs.artifacts['output1'].path}}",
                    ],
                    env=None,
                    resources=None),
                graph=None,
                importer=None),
            description=None,
            inputs={
                'input1': structures.InputSpec(type='String', default=None)
            },
            outputs={
                'output1': structures.OutputSpec(type='system.Artifact@0.0.1')
            })
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
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=['sh', '-c', 'echo "$0" "$1"'],
                    args=[
                        'input: ',
                        "{{$.inputs.parameters['optional_input_1']}}",
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
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=[
                        'sh',
                        '-c',
                        'echo "$0"',
                        "{{$.inputs.parameters['input1']}}+{{$.inputs.parameters['input2']}}",
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


class TestNormalizeTimeString(parameterized.TestCase):

    @parameterized.parameters([
        ('1 hour', '1h'),
        ('2 hours', '2h'),
        ('2hours', '2h'),
        ('2 w', '2w'),
        ('2d', '2d'),
    ])
    def test(self, unnorm: str, norm: str):
        self.assertEqual(structures.normalize_time_string(unnorm), norm)

    def test_multipart_duration_raises(self):
        with self.assertRaisesRegex(ValueError, 'Invalid duration string:'):
            structures.convert_duration_to_seconds('1 day 1 hour')

    def test_non_int_value_raises(self):
        with self.assertRaisesRegex(ValueError, 'Invalid duration string:'):
            structures.convert_duration_to_seconds('one hour')


class TestConvertDurationToSeconds(parameterized.TestCase):

    @parameterized.parameters([
        ('1 hour', 3600),
        ('2 hours', 7200),
        ('2hours', 7200),
        ('2 w', 1209600),
        ('2d', 172800),
    ])
    def test(self, duration: str, seconds: int):
        self.assertEqual(
            structures.convert_duration_to_seconds(duration), seconds)

    def test_unsupported_duration_unit(self):
        with self.assertRaisesRegex(ValueError, 'Unsupported duration unit:'):
            structures.convert_duration_to_seconds('1 year')


class TestRetryPolicy(unittest.TestCase):

    def test_to_proto(self):
        retry_policy_struct = structures.RetryPolicy(
            max_retry_count=10,
            backoff_duration='1h',
            backoff_factor=1.5,
            backoff_max_duration='2 weeks')

        retry_policy_proto = retry_policy_struct.to_proto()
        self.assertEqual(retry_policy_proto.max_retry_count, 10)
        self.assertEqual(retry_policy_proto.backoff_duration.seconds, 3600)
        self.assertEqual(retry_policy_proto.backoff_factor, 1.5)
        # tests cap
        self.assertEqual(retry_policy_proto.backoff_max_duration.seconds, 3600)


if __name__ == '__main__':
    unittest.main()
