# Copyright 2022 The Kubeflow Authors
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
"""Contains tests for kfp.components.placeholders."""
import os
import tempfile
from typing import Any

from absl.testing import parameterized
from kfp import compiler
from kfp import dsl
from kfp.components import placeholders
from kfp.dsl import Artifact
from kfp.dsl import Output


class TestExecutorInputPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(placeholders.ExecutorInputPlaceholder()._to_string(),
                         '{{$}}')


class TestInputValuePlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputValuePlaceholder('input1')._to_string(),
            "{{$.inputs.parameters['input1']}}")


class TestInputPathPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputPathPlaceholder('input1')._to_string(),
            "{{$.inputs.artifacts['input1'].path}}")


class TestInputUriPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputUriPlaceholder('input1')._to_string(),
            "{{$.inputs.artifacts['input1'].uri}}")


class TestInputMetadataPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputMetadataPlaceholder('input1')._to_string(),
            "{{$.inputs.artifacts['input1'].metadata}}")


class TestOutputPathPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputPathPlaceholder('output1')._to_string(),
            "{{$.outputs.artifacts['output1'].path}}")


class TestOutputParameterPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputParameterPlaceholder('output1')._to_string(),
            "{{$.outputs.parameters['output1'].output_file}}")


class TestOutputUriPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputUriPlaceholder('output1')._to_string(),
            "{{$.outputs.artifacts['output1'].uri}}")


class TestOutputMetadataPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputMetadataPlaceholder('output1')._to_string(),
            "{{$.outputs.artifacts['output1'].metadata}}")


class TestIfPresentPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        (placeholders.IfPresentPlaceholder(
            input_name='input1', then=['then'], else_=['something']),
         '{"IfPresent": {"InputName": "input1", "Then": ["then"], "Else": ["something"]}}'
        ),
        (placeholders.IfPresentPlaceholder(
            input_name='input1', then='then', else_='something'),
         '{"IfPresent": {"InputName": "input1", "Then": "then", "Else": "something"}}'
        ),
        (placeholders.IfPresentPlaceholder(
            input_name='input1', then='then', else_=['something']),
         '{"IfPresent": {"InputName": "input1", "Then": "then", "Else": ["something"]}}'
        ),
        (placeholders.IfPresentPlaceholder(
            input_name='input1', then=['then'], else_='something'),
         '{"IfPresent": {"InputName": "input1", "Then": ["then"], "Else": "something"}}'
        ),
    ])
    def test_strings_and_lists(
            self, placeholder_obj: placeholders.IfPresentPlaceholder,
            placeholder: str):
        self.assertEqual(placeholder_obj._to_string(), placeholder)

    @parameterized.parameters([
        (placeholders.IfPresentPlaceholder(
            input_name='input1',
            then=[
                '--flag',
                placeholders.OutputUriPlaceholder(output_name='output1')
            ],
            else_=[
                '--flag',
                placeholders.OutputMetadataPlaceholder(output_name='output1')
            ]),
         """{"IfPresent": {"InputName": "input1", "Then": ["--flag", "{{$.outputs.artifacts['output1'].uri}}"], "Else": ["--flag", "{{$.outputs.artifacts['output1'].metadata}}"]}}"""
        ),
        (placeholders.IfPresentPlaceholder(
            input_name='input1',
            then=placeholders.InputPathPlaceholder(input_name='input2'),
            else_=placeholders.InputValuePlaceholder(input_name='input2')),
         """{"IfPresent": {"InputName": "input1", "Then": "{{$.inputs.artifacts['input2'].path}}", "Else": "{{$.inputs.parameters['input2']}}"}}"""
        ),
    ])
    def test_with_primitive_placeholders(
            self, placeholder_obj: placeholders.IfPresentPlaceholder,
            placeholder: str):
        self.assertEqual(placeholder_obj._to_string(), placeholder)

    def test_if_present_with_single_element_simple_can_be_compiled(self):

        @dsl.container_component
        def container_component(a: str):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    placeholders.IfPresentPlaceholder(
                        input_name='a', then='b', else_='c')
                ])

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_component, package_path=output_yaml)

    def test_if_present_with_single_element_parameter_reference_can_be_compiled(
            self):

        @dsl.container_component
        def container_component(a: str):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    placeholders.IfPresentPlaceholder(
                        input_name='a', then=a, else_='c')
                ])

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_component, package_path=output_yaml)

    def test_if_present_with_single_element_artifact_reference_can_be_compiled(
            self):

        @dsl.container_component
        def container_component(a: dsl.Input[dsl.Artifact]):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    placeholders.IfPresentPlaceholder(
                        input_name='a', then=a.path, else_='c')
                ])

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_component, package_path=output_yaml)


class TestConcatPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        (placeholders.ConcatPlaceholder(['a']), '{"Concat": ["a"]}'),
        (placeholders.ConcatPlaceholder(['a', 'b']), '{"Concat": ["a", "b"]}'),
    ])
    def test_strings(self, placeholder_obj: placeholders.ConcatPlaceholder,
                     placeholder_string: str):
        self.assertEqual(placeholder_obj._to_string(), placeholder_string)

    @parameterized.parameters([
        (placeholders.ConcatPlaceholder([
            'a', placeholders.InputPathPlaceholder(input_name='input2')
        ]), """{"Concat": ["a", "{{$.inputs.artifacts['input2'].path}}"]}"""),
        (placeholders.ConcatPlaceholder([
            placeholders.InputValuePlaceholder(input_name='input2'), 'b'
        ]), """{"Concat": ["{{$.inputs.parameters['input2']}}", "b"]}"""),
    ])
    def test_primitive_placeholders(
            self, placeholder_obj: placeholders.ConcatPlaceholder,
            placeholder_string: str):
        self.assertEqual(placeholder_obj._to_string(), placeholder_string)


class TestContainerPlaceholdersTogether(parameterized.TestCase):

    @parameterized.parameters([
        (placeholders.ConcatPlaceholder([
            'a', placeholders.ConcatPlaceholder(['b', 'c'])
        ]), '{"Concat": ["a", {"Concat": ["b", "c"]}]}'),
        (placeholders.ConcatPlaceholder([
            'a',
            placeholders.ConcatPlaceholder([
                'b',
                placeholders.ConcatPlaceholder([
                    'c',
                    placeholders.InputValuePlaceholder(input_name='input2')
                ])
            ])
        ]),
         """{"Concat": ["a", {"Concat": ["b", {"Concat": ["c", "{{$.inputs.parameters['input2']}}"]}]}]}"""
        ),
        (placeholders.ConcatPlaceholder([
            'a',
            placeholders.ConcatPlaceholder([
                'b',
                placeholders.IfPresentPlaceholder(
                    input_name='output1', then='then', else_='something')
            ])
        ]),
         '{"Concat": ["a", {"Concat": ["b", {"IfPresent": {"InputName": "output1", "Then": "then", "Else": "something"}}]}]}'
        ),
        (placeholders.ConcatPlaceholder([
            'a',
            placeholders.IfPresentPlaceholder(
                input_name='output1',
                then=placeholders.ConcatPlaceholder([
                    '--',
                    'flag',
                    placeholders.InputPathPlaceholder(input_name='input2'),
                ]),
                else_='b'),
            'c',
        ]),
         """{"Concat": ["a", {"IfPresent": {"InputName": "output1", "Then": {"Concat": ["--", "flag", "{{$.inputs.artifacts['input2'].path}}"]}, "Else": "b"}}, "c"]}"""
        ),
        (placeholders.ConcatPlaceholder([
            'a',
            placeholders.IfPresentPlaceholder(
                input_name='output1',
                then=placeholders.ConcatPlaceholder(['--', 'flag']),
                else_=placeholders.InputPathPlaceholder(input_name='input2')),
            'c',
        ]),
         """{"Concat": ["a", {"IfPresent": {"InputName": "output1", "Then": {"Concat": ["--", "flag"]}, "Else": "{{$.inputs.artifacts['input2'].path}}"}}, "c"]}"""
        ),
    ])
    def test_valid(self, placeholder_obj: placeholders.IfPresentPlaceholder,
                   placeholder: str):
        self.assertEqual(placeholder_obj._to_string(), placeholder)

    def test_only_single_element_ifpresent_inside_concat_outer(self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            placeholders.ConcatPlaceholder([
                'b',
                placeholders.IfPresentPlaceholder(
                    input_name='output1', then=['then'], else_=['something'])
            ])

    def test_only_single_element_ifpresent_inside_concat_recursive(self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            placeholders.ConcatPlaceholder([
                'a',
                placeholders.ConcatPlaceholder([
                    'b',
                    placeholders.IfPresentPlaceholder(
                        input_name='output1',
                        then=['then'],
                        else_=['something'])
                ])
            ])

        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            placeholders.ConcatPlaceholder([
                'a',
                placeholders.ConcatPlaceholder([
                    'b',
                    placeholders.IfPresentPlaceholder(
                        input_name='output1',
                        then=placeholders.ConcatPlaceholder([
                            placeholders.IfPresentPlaceholder(
                                input_name='a', then=['b'])
                        ]),
                        else_='something')
                ])
            ])

    def test_only_single_element_in_nested_ifpresent_inside_concat(self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            dsl.ConcatPlaceholder([
                'my-prefix-',
                dsl.IfPresentPlaceholder(
                    input_name='input1',
                    then=[
                        dsl.IfPresentPlaceholder(
                            input_name='input1',
                            then=dsl.ConcatPlaceholder(['infix-', 'value']))
                    ])
            ])

    def test_recursive_nested_placeholder_validation_does_not_exit_when_first_valid_then_is_found(
            self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            dsl.ConcatPlaceholder([
                'my-prefix-',
                dsl.IfPresentPlaceholder(
                    input_name='input1',
                    then=dsl.IfPresentPlaceholder(
                        input_name='input1',
                        then=[dsl.ConcatPlaceholder(['infix-', 'value'])]))
            ])

    def test_only_single_element_in_nested_ifpresent_inside_concat_with_outer_ifpresent(
            self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            dsl.IfPresentPlaceholder(
                input_name='input_1',
                then=dsl.ConcatPlaceholder([
                    'my-prefix-',
                    dsl.IfPresentPlaceholder(
                        input_name='input1',
                        then=dsl.IfPresentPlaceholder(
                            input_name='input1',
                            then=[dsl.ConcatPlaceholder(['infix-', 'value'])]))
                ]))

    def test_valid_then_but_invalid_else(self):
        with self.assertRaisesRegex(
                ValueError,
                f'Please use a single element for `then` and `else_` only\.'):
            dsl.ConcatPlaceholder([
                'my-prefix-',
                dsl.IfPresentPlaceholder(
                    input_name='input1',
                    then=dsl.IfPresentPlaceholder(
                        input_name='input1',
                        then='single-element',
                        else_=['one', 'two']))
            ])


class TestConvertCommandLineElementToStringOrStruct(parameterized.TestCase):

    @parameterized.parameters(['a', 'word', 1])
    def test_pass_through(self, val: Any):
        self.assertEqual(
            placeholders.convert_command_line_element_to_string_or_struct(val),
            val)

    @parameterized.parameters([
        (placeholders.ExecutorInputPlaceholder(), '{{$}}'),
        (placeholders.InputValuePlaceholder('input1'),
         """{{$.inputs.parameters['input1']}}"""),
        (placeholders.OutputPathPlaceholder('output1'),
         """{{$.outputs.artifacts['output1'].path}}"""),
    ])
    def test_primitive_placeholder(self, placeholder: Any, expected: str):
        self.assertEqual(
            placeholders.convert_command_line_element_to_string_or_struct(
                placeholder), expected)


class OtherPlaceholderTests(parameterized.TestCase):

    def test_primitive_placeholder_can_be_used_in_fstring1(self):

        @dsl.container_component
        def echo_bool(boolean: bool = True):
            return dsl.ContainerSpec(
                image='alpine', command=['sh', '-c', f'echo {boolean}'])

        self.assertEqual(
            echo_bool.component_spec.implementation.container.command,
            ['sh', '-c', "echo {{$.inputs.parameters['boolean']}}"])

    def test_primitive_placeholder_can_be_used_in_fstring2(self):

        @dsl.container_component
        def container_with_placeholder_in_fstring(
            output_artifact: Output[Artifact],
            text1: str,
        ):
            return dsl.ContainerSpec(
                image='python:3.7',
                command=[
                    'my_program',
                    f'prefix-{text1}',
                    f'{output_artifact.uri}/0',
                ])

        self.assertEqual(
            container_with_placeholder_in_fstring.component_spec.implementation
            .container.command, [
                'my_program',
                "prefix-{{$.inputs.parameters['text1']}}",
                "{{$.outputs.artifacts['output_artifact'].uri}}/0",
            ])

    def test_cannot_use_concat_placeholder_in_f_string(self):

        with self.assertRaisesRegex(
                ValueError, 'Cannot use ConcatPlaceholder in an f-string.'):

            @dsl.container_component
            def container_with_placeholder_in_fstring(
                text1: str,
                text2: str,
            ):
                return dsl.ContainerSpec(
                    image='python:3.7',
                    command=[
                        'my_program',
                        f'another-prefix-{dsl.ConcatPlaceholder([text1, text2])}',
                    ])

    def test_cannot_use_ifpresent_placeholder_in_f_string(self):

        with self.assertRaisesRegex(
                ValueError, 'Cannot use IfPresentPlaceholder in an f-string.'):

            @dsl.container_component
            def container_with_placeholder_in_fstring(
                text1: str,
                text2: str,
            ):
                return dsl.ContainerSpec(
                    image='python:3.7',
                    command=[
                        'echo',
                        f"another-prefix-{dsl.IfPresentPlaceholder(input_name='text1', then=['val'])}",
                    ])
