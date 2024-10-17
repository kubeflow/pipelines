# Copyright 2023 The Kubeflow Authors
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
"""Tests for placeholder_utils.py."""

import json
from typing import List, Optional
import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from kfp.local import placeholder_utils
from kfp.pipeline_spec import pipeline_spec_pb2

executor_input = pipeline_spec_pb2.ExecutorInput()
json_format.ParseDict(
    {
        'inputs': {
            'parameterValues': {
                'boolean': False,
                'dictionary': {
                    'foo': 'bar'
                },
            },
            'artifacts': {
                'in_a': {
                    'artifacts': [{
                        'name':
                            'in_a',
                        'type': {
                            'schemaTitle': 'system.Dataset',
                            'schemaVersion': '0.0.1'
                        },
                        'uri':
                            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/upstream-comp/in_a',
                        'metadata': {
                            'foo': {
                                'bar': 'baz'
                            }
                        }
                    }]
                }
            },
        },
        'outputs': {
            'parameters': {
                'Output': {
                    'outputFile':
                        '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output'
                }
            },
            'artifacts': {
                'out_a': {
                    'artifacts': [{
                        'name':
                            'out_a',
                        'type': {
                            'schemaTitle': 'system.Dataset',
                            'schemaVersion': '0.0.1'
                        },
                        'uri':
                            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
                        # include metadata on outputs since it allows us to
                        # test the placeholder
                        # "{{$.outputs.artifacts[''out_a''].metadata[''foo'']}}"
                        # for comprehensive testing, but in practice metadata
                        # will never be set on output artifacts since they
                        # haven't been created yet
                        'metadata': {
                            'foo': {
                                'bar': 'baz'
                            }
                        }
                    }]
                }
            },
            'outputFile':
                '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json'
        }
    },
    executor_input)

EXECUTOR_INPUT_DICT = json_format.MessageToDict(executor_input)


class TestReplacePlaceholders(unittest.TestCase):
    # most of the logic is tested in TestReplacePlaceholderForElement, so this is just a basic test to invoke the code and make sure the placeholder resolution is applied correctly to every element in the list

    def test(self):
        full_command = [
            'echo',
            'something before the placeholder {{$}}',
            'something else',
            '{{$.outputs.output_file}}',
        ]
        actual = placeholder_utils.replace_placeholders(
            full_command=full_command,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            unique_pipeline_id=placeholder_utils.make_random_id(),
        )
        expected = [
            'echo',
            f'something before the placeholder {json.dumps(EXECUTOR_INPUT_DICT)}',
            'something else',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json',
        ]
        self.assertEqual(actual, expected)


class TestResolveIndividualPlaceholder(parameterized.TestCase):

    # TODO: consider supporting JSON escape
    # TODO: update when output lists of artifacts are supported
    @parameterized.parameters([
        (
            '{{$}}',
            json.dumps(EXECUTOR_INPUT_DICT),
        ),
        (
            '{{$.outputs.output_file}}',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json',
        ),
        (
            '{{$.outputMetadataUri}}',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json',
        ),
        (
            '{{$.pipeline_job_name}}',
            'my-pipeline-2023-10-10-13-32-59-420710',
        ),
        (
            '{{$.pipeline_job_uuid}}',
            '123456789',
        ),
        (
            '{{$.pipeline_task_name}}',
            'comp',
        ),
        (
            '{{$.pipeline_task_uuid}}',
            '987654321',
        ),
        (
            '{{$.pipeline_root}}',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
        ),
    ])
    def test_constant_placeholders(self, element: str, expected: str):
        actual = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
        )
        self.assertEqual(actual, expected)

    @parameterized.parameters([
        (
            '{{$}}invalidjson',
            json.dumps(EXECUTOR_INPUT_DICT) + 'invalidjson',
        ),
        (
            '{{$.pipeline_job_name}}/{{$.pipeline_task_name}}',
            'my-pipeline-2023-10-10-13-32-59-420710/comp',
        ),
        (
            '{{$.pipeline_root}}/foo/bar',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/foo/bar',
        ),
    ])
    def test_concatenated_placeholders_resolve(self, element: str,
                                               expected: str):
        actual = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
        )
        self.assertEqual(actual, expected)

    @parameterized.parameters([
        (
            "{{$.inputs.parameters[''boolean'']}}",
            json.dumps(False),
        ),
        (
            "{{$.inputs.parameters[''not_present'']}}",
            json.dumps(None),
        ),
        (
            "{{$.outputs.artifacts[''out_a''].metadata}}",
            json.dumps({'foo': {
                'bar': 'baz'
            }}),
        ),
        (
            "{{$.outputs.parameters[''Output''].output_file}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output',
        ),
        (
            "{{$.outputs.artifacts[''out_a''].uri}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
        ),
        (
            "{{$.outputs.artifacts[''out_a''].path}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
        ),
        (
            "{{$.outputs.artifacts[''out_a''].metadata[''foo'']}}",
            json.dumps({'bar': 'baz'}),
        ),
        (
            "{{$.inputs.artifacts[''in_a''].metadata}}",
            json.dumps({'foo': {
                'bar': 'baz'
            }}),
        ),
        (
            "{{$.inputs.artifacts[''in_a''].uri}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/upstream-comp/in_a',
        ),
        (
            "{{$.inputs.artifacts[''in_a''].path}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/upstream-comp/in_a',
        ),
        (
            "{{$.inputs.artifacts[''in_a''].metadata[''foo'']}}",
            json.dumps({'bar': 'baz'}),
        ),
    ])
    def test_io_placeholders(self, element: str, expected: str):
        actual = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
        )
        self.assertEqual(actual, expected)

    @parameterized.parameters([
        (
            "my-prefix-{{$.inputs.parameters[''boolean'']}}-suffix",
            'my-prefix-false-suffix',
        ),
        (
            "--param={{$.inputs.parameters[''not_present'']}}",
            '--param=null',
        ),
        (
            "prefix{{$.outputs.parameters[''Output''].output_file}}/suffix",
            'prefix/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output/suffix',
        ),
        (
            "prefix{{$.inputs.parameters[''dictionary'']}}suffix",
            'prefix{"foo": "bar"}suffix',
        ),
    ])
    def test_io_placeholder_with_string_concat(self, element: str,
                                               expected: str):
        actual = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
        )
        self.assertEqual(actual, expected)


class TestGetValueUsingPath(unittest.TestCase):

    def test_valid_path(self):
        actual = placeholder_utils.get_value_using_path(
            {'a': {
                'b': {
                    'c': 10
                }
            }},
            ['a', 'b', 'c'],
        )
        expected = 10
        self.assertEqual(actual, expected)

    def test_invalid_path(self):
        actual = placeholder_utils.get_value_using_path(
            {'a': {
                'b': {
                    'c': 10
                }
            }},
            ['a', 'x'],
        )
        self.assertIsNone(actual)

    def test_empty_path(self):
        with self.assertRaisesRegex(ValueError, r'path cannot be empty\.'):
            placeholder_utils.get_value_using_path({'a': 20}, [])


class TestResolveStructPlaceholders(parameterized.TestCase):

    @parameterized.parameters([
        (
            """{"Concat": ["a", "b", "c"]}""",
            [],
            'abc',
        ),
        (
            """{"Concat": ["prefix", "-", "{{$.outputs.artifacts[''x''].uri}}"]}""",
            [],
            "prefix-{{$.outputs.artifacts[''x''].uri}}",
        ),
        (
            """{"Concat": ["a", {"Concat": ["b", "c"]}]}""",
            [],
            'abc',
        ),
        (
            """{"IfPresent": {"InputName": "x", "Then": ["foo"], "Else": ["bar"]}}""",
            [],
            ['bar'],
        ),
        (
            """{"IfPresent": {"InputName": "x", "Then": ["foo"], "Else": ["bar"]}}""",
            ['x'],
            ['foo'],
        ),
        (
            """{"Concat": ["a", {"Concat": ["b", {"Concat": ["c", "{{$.inputs.parameters[''input2'']}}"]}]}]}""",
            [],
            "abc{{$.inputs.parameters[''input2'']}}",
        ),
        (
            """{"Concat": ["a", {"Concat": ["b", {"IfPresent": {"InputName": "foo", "Then": "c", "Else": "d"}}]}]}""",
            [],
            'abd',
        ),
        (
            """{"Concat": ["--flag", {"Concat": ["=", {"IfPresent": {"InputName": "x", "Then": "thing", "Else": "otherwise"}}]}]}""",
            ['x'],
            '--flag=thing',
        ),
        (
            """{"Concat": ["a", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["--", "flag", "{{$.inputs.artifacts['input2'].path}}"]}, "Else": "b"}}, "c"]}""",
            [],
            'abc',
        ),
        (
            """{"Concat": ["--flag", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["=", "{{$.inputs.artifacts['input2'].path}}"]}, "Else": "b"}}, "-suffix"]}""",
            ['foo'],
            "--flag={{$.inputs.artifacts['input2'].path}}-suffix",
        ),
        (
            """{"Concat": ["a-", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["--", "flag"]}, "Else": "{{$.inputs.artifacts['input2'].path}}"}}, "-c"]}""",
            [],
            "a-{{$.inputs.artifacts['input2'].path}}-c",
        ),
        (
            """{"Concat": ["--", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["flag"]}, "Else": "{{$.inputs.artifacts['input2'].path}}"}}, "=c"]}""",
            ['foo'],
            '--flag=c',
        ),
        (
            """{"Concat": ["--", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["flag"]}}}, "=c"]}""",
            ['foo'],
            '--flag=c',
        ),
        (
            """{"Concat": ["--flag", {"IfPresent": {"InputName": "foo", "Then": {"Concat": ["=", "other", "_val"]}}}, "=foo"]}""",
            [],
            '--flag=foo',
        ),
        (
            """{"IfPresent": {"InputName": "foo", "Then": {"Concat": ["--", "flag"]}}}""",
            ['foo'],
            '--flag',
        ),
        (
            """{"IfPresent": {"InputName": "foo", "Then": {"Concat": ["--", "flag"]}}}""",
            [],
            None,
        ),
    ])
    def test(
        self,
        placeholder: str,
        provided_inputs: List[str],
        expected: Optional[None],
    ):
        actual = placeholder_utils.resolve_struct_placeholders(
            placeholder,
            provided_inputs,
        )
        self.assertEqual(actual, expected)


class TestResolveSelfReferencesInExecutorInput(unittest.TestCase):

    def test_simple(self):
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'pipelinechannel--identity-Output':
                        'foo',
                    'string':
                        "{{$.inputs.parameters['pipelinechannel--identity-Output']}}-bar"
                }
            },
            'outputs': {
                'outputFile':
                    '/foo/bar/my-pipeline-2024-01-26-12-26-24-162075/echo/executor_output.json'
            }
        }
        expected = {
            'inputs': {
                'parameterValues': {
                    'pipelinechannel--identity-Output': 'foo',
                    'string': 'foo-bar'
                }
            },
            'outputs': {
                'outputFile':
                    '/foo/bar/my-pipeline-2024-01-26-12-26-24-162075/echo/executor_output.json'
            }
        }
        actual = placeholder_utils.resolve_self_references_in_executor_input(
            executor_input_dict,
            pipeline_resource_name='my-pipeline-2024-01-26-12-26-24-162075',
            task_resource_name='echo',
            pipeline_root='/foo/bar/my-pipeline-2024-01-26-12-26-24-162075',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
        )
        self.assertEqual(actual, expected)


if __name__ == '__main__':
    unittest.main()
