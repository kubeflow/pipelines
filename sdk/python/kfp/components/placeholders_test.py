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
from typing import Any

from absl.testing import parameterized
from kfp.components import placeholders


class TestExecutorInputPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(placeholders.ExecutorInputPlaceholder().to_string(),
                         '{{$}}')


class TestInputValuePlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputValuePlaceholder('input1').to_string(),
            "{{$.inputs.parameters['input1']}}")


class TestInputPathPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputPathPlaceholder('input1').to_string(),
            "{{$.inputs.artifacts['input1'].path}}")


class TestInputUriPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputUriPlaceholder('input1').to_string(),
            "{{$.inputs.artifacts['input1'].uri}}")


class TestInputMetadataPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.InputMetadataPlaceholder('input1').to_string(),
            "{{$.inputs.artifacts['input1'].metadata}}")


class TestOutputPathPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputPathPlaceholder('output1').to_string(),
            "{{$.outputs.artifacts['output1'].path}}")


class TestOutputParameterPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputParameterPlaceholder('output1').to_string(),
            "{{$.outputs.parameters['output1'].output_file}}")


class TestOutputUriPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputUriPlaceholder('output1').to_string(),
            "{{$.outputs.artifacts['output1'].uri}}")


class TestOutputMetadataPlaceholder(parameterized.TestCase):

    def test(self):
        self.assertEqual(
            placeholders.OutputMetadataPlaceholder('output1').to_string(),
            "{{$.outputs.artifacts['output1'].metadata}}")


class TestIfPresentPlaceholderStructure(parameterized.TestCase):

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
        self.assertEqual(placeholder_obj.to_string(), placeholder)

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
        self.assertEqual(placeholder_obj.to_string(), placeholder)


class TestConcatPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        (placeholders.ConcatPlaceholder(['a']), '{"Concat": ["a"]}'),
        (placeholders.ConcatPlaceholder(['a', 'b']), '{"Concat": ["a", "b"]}'),
    ])
    def test_strings(self, placeholder_obj: placeholders.ConcatPlaceholder,
                     placeholder_string: str):
        self.assertEqual(placeholder_obj.to_string(), placeholder_string)

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
        self.assertEqual(placeholder_obj.to_string(), placeholder_string)


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
        # TODO(cjmccarthy): what should be allowed?
        (placeholders.ConcatPlaceholder([
            'a',
            placeholders.ConcatPlaceholder([
                'b',
                placeholders.IfPresentPlaceholder(
                    input_name='output1', then=['then'], else_=['something'])
            ])
        ]),
         '{"Concat": ["a", {"Concat": ["b", {"IfPresent": {"InputName": "output1", "Then": ["then"], "Else": ["something"]}}]}]}'
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
    def test(self, placeholder_obj: placeholders.IfPresentPlaceholder,
             placeholder: str):
        self.assertEqual(placeholder_obj.to_string(), placeholder)


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
