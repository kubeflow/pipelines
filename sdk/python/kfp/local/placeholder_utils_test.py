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
            }
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
    }, executor_input)

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
        )
        expected = [
            'echo',
            f'something before the placeholder {json.dumps(EXECUTOR_INPUT_DICT)}',
            'something else',
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/executor_output.json',
        ]
        self.assertEqual(actual, expected)


class TestReplacePlaceholderForElement(parameterized.TestCase):

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
        actual = placeholder_utils.replace_placeholder_for_element(
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
        actual = placeholder_utils.replace_placeholder_for_element(
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
            "{{$.outputs.parameters[''Output''].output_file}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/Output',
        ),
        # (
        #     "{{$.inputs.parameters['$0'].json_escape[0]}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.parameters['$0'].json_escape[1]}}",
        #     '',
        # ),
        # TODO: add input lists of artifacts when supported
        # (
        #     "{{$.inputs.artifacts['foo']}}",
        #     '',
        # ),
        # TODO: add input artifact constants when supported
        # (
        #     "{{$.inputs.artifacts['foo'].uri}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].path}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].value}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].metadata}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].metadata.json_escape[1]}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].metadata['$1'].json_escape[0]}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts['foo'].metadata['$1'].json_escape[1]}}",
        #     '',
        # ),
        # (
        #     "{{$.inputs.artifacts[foo].metadata['$1']}}",
        #     '',
        # ),
        # TODO: add output lists of artifacts when supported
        # (
        #     "{{$.outputs.artifacts[''out_a'']}}",
        #     '',
        # ),
        (
            "{{$.outputs.artifacts[''out_a''].uri}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
        ),
        (
            "{{$.outputs.artifacts[''out_a''].path}}",
            '/foo/bar/my-pipeline-2023-10-10-13-32-59-420710/comp/out_a',
        ),
        (
            "{{$.outputs.artifacts[''out_a''].metadata}}",
            json.dumps({'foo': {
                'bar': 'baz'
            }}),
        ),
        # TODO: consider supporting JSON escape
        # (
        #     "{{$.outputs.artifacts[''out_a''].metadata.json_escape[1]}}",
        #     '',
        # ),
        # (
        #     "{{$.outputs.artifacts[''out_a''].metadata[''foo''].json_escape[0]}}",
        #     '',
        # ),
        # (
        #     "{{$.outputs.artifacts[''out_a''].metadata[''foo''].json_escape[1]}}",
        #     '',
        # ),
        (
            "{{$.outputs.artifacts[''out_a''].metadata[''foo'']}}",
            json.dumps({'bar': 'baz'}),
        ),
    ])
    def test_io_placeholders(self, element: str, expected: str):
        actual = placeholder_utils.replace_placeholder_for_element(
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
        actual = placeholder_utils.replace_placeholder_for_element(
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


if __name__ == '__main__':
    unittest.main()
