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
from typing import List
import unittest

from absl.testing import parameterized
from kfp.components import placeholders


class TestExecutorInputPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ('{{$}}', placeholders.ExecutorInputPlaceholder()),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.ExecutorInputPlaceholder):
        self.assertEqual(
            placeholders.ExecutorInputPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestInputValuePlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.inputs.parameters['input1']}}",
         placeholders.InputValuePlaceholder('input1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.InputValuePlaceholder):
        self.assertEqual(
            placeholders.InputValuePlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)

    @parameterized.parameters([
        ("{{$.inputs.parameters[''input1'']}}",
         placeholders.InputValuePlaceholder('input1')),
        ('{{$.inputs.parameters["input1"]}}',
         placeholders.InputValuePlaceholder('input1')),
    ])
    def test_from_placeholder_special_quote_case(
            self, placeholder_string: str,
            placeholder_obj: placeholders.InputValuePlaceholder):
        self.assertEqual(
            placeholders.InputValuePlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)


class TestInputPathPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.inputs.artifacts['input1'].path}}",
         placeholders.InputPathPlaceholder('input1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.InputPathPlaceholder):
        self.assertEqual(
            placeholders.InputPathPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestInputUriPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.inputs.artifacts['input1'].uri}}",
         placeholders.InputUriPlaceholder('input1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.InputUriPlaceholder):
        self.assertEqual(
            placeholders.InputUriPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestInputMetadataPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.inputs.artifacts['input1'].metadata}}",
         placeholders.InputMetadataPlaceholder('input1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.InputMetadataPlaceholder):
        self.assertEqual(
            placeholders.InputMetadataPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestOutputPathPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.outputs.artifacts['output1'].path}}",
         placeholders.OutputPathPlaceholder('output1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.OutputPathPlaceholder):
        self.assertEqual(
            placeholders.OutputPathPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestOutputParameterPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.outputs.parameters['output1'].output_file}}",
         placeholders.OutputParameterPlaceholder('output1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.OutputParameterPlaceholder):
        self.assertEqual(
            placeholders.OutputParameterPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestOutputUriPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.outputs.artifacts['output1'].uri}}",
         placeholders.OutputUriPlaceholder('output1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.OutputUriPlaceholder):
        self.assertEqual(
            placeholders.OutputUriPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestOutputMetadataPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ("{{$.outputs.artifacts['output1'].metadata}}",
         placeholders.OutputMetadataPlaceholder('output1')),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.OutputMetadataPlaceholder):
        self.assertEqual(
            placeholders.OutputMetadataPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestIfPresentPlaceholderStructure(parameterized.TestCase):

    def test_else_transform(self):
        obj = placeholders.IfPresentPlaceholder(
            then='then', input_name='input_name', else_=['something'])
        self.assertEqual(obj.else_, ['something'])

        obj = placeholders.IfPresentPlaceholder(
            then='then', input_name='input_name', else_=[])
        self.assertEqual(obj.else_, None)

    @parameterized.parameters([
        ('{"IfPresent": {"InputName": "output1", "Then": "then", "Else": "something"}}',
         placeholders.IfPresentPlaceholder(
             input_name='output1', then='then', else_='something')),
        ('{"IfPresent": {"InputName": "output1", "Then": "then"}}',
         placeholders.IfPresentPlaceholder(input_name='output1', then='then')),
        ('{"IfPresent": {"InputName": "output1", "Then": "then"}}',
         placeholders.IfPresentPlaceholder('output1', 'then')),
        ('{"IfPresent": {"InputName": "output1", "Then": ["then"], "Else": ["something"]}}',
         placeholders.IfPresentPlaceholder(
             input_name='output1', then=['then'], else_=['something'])),
        ('{"IfPresent": {"InputName": "output1", "Then": ["then", {"IfPresent": {"InputName": "output1", "Then": ["then"], "Else": ["something"]}}], "Else": ["something"]}}',
         placeholders.IfPresentPlaceholder(
             input_name='output1',
             then=[
                 'then',
                 placeholders.IfPresentPlaceholder(
                     input_name='output1', then=['then'], else_=['something'])
             ],
             else_=['something'])),
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.ConcatPlaceholder):
        self.assertEqual(
            placeholders.IfPresentPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)


class TestConcatPlaceholder(parameterized.TestCase):

    @parameterized.parameters([
        ('{"Concat": ["a", "b"]}', placeholders.ConcatPlaceholder(['a', 'b'])),
        ('{"Concat": ["a", {"Concat": ["b", "c"]}]}',
         placeholders.ConcatPlaceholder(
             ['a', placeholders.ConcatPlaceholder(['b', 'c'])])),
        ('{"Concat": ["a", {"Concat": ["b", {"IfPresent": {"InputName": "output1", "Then": ["then"], "Else": ["something"]}}]}]}',
         placeholders.ConcatPlaceholder([
             'a',
             placeholders.ConcatPlaceholder([
                 'b',
                 placeholders.IfPresentPlaceholder(
                     input_name='output1', then=['then'], else_=['something'])
             ])
         ]))
    ])
    def test_to_from_placeholder(
            self, placeholder_string: str,
            placeholder_obj: placeholders.ConcatPlaceholder):
        self.assertEqual(
            placeholders.ConcatPlaceholder._from_placeholder_string(
                placeholder_string), placeholder_obj)
        self.assertEqual(placeholder_obj._to_placeholder_string(),
                         placeholder_string)

    @parameterized.parameters([
        "{{$.inputs.parameters[''input1'']}}+{{$.inputs.parameters[''input2'']}}",
        '{"Concat": ["a", "b"]}', '{"Concat": ["a", {"Concat": ["b", "c"]}]}',
        "{{$.inputs.parameters[''input1'']}}something{{$.inputs.parameters[''input2'']}}",
        "{{$.inputs.parameters[''input_prefix'']}}some value",
        "some value{{$.inputs.parameters[''input_suffix'']}}",
        "some value{{$.inputs.parameters[''input_infix'']}}some value"
    ])
    def test_is_match(self, placeholder: str):
        self.assertTrue(placeholders.ConcatPlaceholder._is_match(placeholder))

    @parameterized.parameters([
        ("{{$.inputs.parameters[''input1'']}}something{{$.inputs.parameters[''input2'']}}",
         [
             "{{$.inputs.parameters[''input1'']}}", 'something',
             "{{$.inputs.parameters[''input2'']}}"
         ]),
        ("{{$.inputs.parameters[''input_prefix'']}}some value",
         ["{{$.inputs.parameters[''input_prefix'']}}", 'some value']),
        ("some value{{$.inputs.parameters[''input_suffix'']}}",
         ['some value', "{{$.inputs.parameters[''input_suffix'']}}"]),
        ("some value{{$.inputs.parameters[''input_infix'']}}some value", [
            'some value', "{{$.inputs.parameters[''input_infix'']}}",
            'some value'
        ]),
        ("{{$.inputs.parameters[''input1'']}}+{{$.inputs.parameters[''input2'']}}",
         [
             "{{$.inputs.parameters[''input1'']}}",
             "{{$.inputs.parameters[''input2'']}}"
         ])
    ])
    def test_split_cel_concat_string(self, placeholder: str,
                                     expected: List[str]):
        self.assertEqual(
            placeholders.ConcatPlaceholder._split_cel_concat_string(
                placeholder), expected)


class TestProcessCommandArg(unittest.TestCase):

    def test_string(self):
        arg = 'test'
        struct = placeholders.maybe_convert_placeholder_string_to_placeholder(
            arg)
        self.assertEqual(struct, arg)

    def test_input_value_placeholder(self):
        arg = "{{$.inputs.parameters['input1']}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            arg)
        expected = placeholders.InputValuePlaceholder(input_name='input1')
        self.assertEqual(actual, expected)

    def test_input_path_placeholder(self):
        arg = "{{$.inputs.artifacts['input1'].path}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            arg)
        expected = placeholders.InputPathPlaceholder('input1')
        self.assertEqual(actual, expected)

    def test_input_uri_placeholder(self):
        arg = "{{$.inputs.artifacts['input1'].uri}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            arg)
        expected = placeholders.InputUriPlaceholder('input1')
        self.assertEqual(actual, expected)

    def test_output_path_placeholder(self):
        arg = "{{$.outputs.artifacts['output1'].path}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            arg)
        expected = placeholders.OutputPathPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_output_uri_placeholder(self):
        placeholder = "{{$.outputs.artifacts['output1'].uri}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            placeholder)
        expected = placeholders.OutputUriPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_output_parameter_placeholder(self):
        placeholder = "{{$.outputs.parameters['output1'].output_file}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            placeholder)
        expected = placeholders.OutputParameterPlaceholder('output1')
        self.assertEqual(actual, expected)

    def test_concat_placeholder(self):
        placeholder = "{{$.inputs.parameters[''input1'']}}+{{$.inputs.parameters[''input2'']}}"
        actual = placeholders.maybe_convert_placeholder_string_to_placeholder(
            placeholder)
        expected = placeholders.ConcatPlaceholder(items=[
            placeholders.InputValuePlaceholder(input_name='input1'),
            placeholders.InputValuePlaceholder(input_name='input2')
        ])
        self.assertEqual(actual, expected)
