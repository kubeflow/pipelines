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

import datetime
import json
import os
import tempfile
from typing import List, Optional
import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from kfp.dsl import constants as dsl_constants
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

    def test_replace_placeholders_with_utc_timestamps(self):
        """Test that replace_placeholders() correctly resolves UTC timestamp
        placeholders.

        This test verifies the full path where utc_timestamp is
        generated internally within replace_placeholders() and
        propagated through the resolution hierarchy. Verifies that the
        timestamps match the expected ISO8601 format and are identical
        across different timestamp placeholders in the same command.
        """
        import re

        iso8601_pattern = r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$'

        full_command = [
            'task',
            '--create-time={{$.pipeline_job_create_time_utc}}',
            '--schedule-time={{$.pipeline_job_schedule_time_utc}}',
            '--resource-name={{$.pipeline_job_resource_name}}',
        ]

        actual = placeholder_utils.replace_placeholders(
            full_command=full_command,
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            unique_pipeline_id='job-123',
        )

        # Verify the command was resolved correctly
        self.assertEqual(len(actual), 4)
        self.assertEqual(actual[0], 'task')

        # Extract the timestamp values from the resolved command
        create_time_match = re.search(r'--create-time=(.+)$', actual[1])
        schedule_time_match = re.search(r'--schedule-time=(.+)$', actual[2])

        self.assertIsNotNone(
            create_time_match,
            f'Could not extract create-time from {actual[1]!r}')
        self.assertIsNotNone(
            schedule_time_match,
            f'Could not extract schedule-time from {actual[2]!r}')

        create_timestamp = create_time_match.group(1)
        schedule_timestamp = schedule_time_match.group(1)

        # Verify both timestamps are valid ISO8601 format
        self.assertRegex(
            create_timestamp, iso8601_pattern,
            f'Create timestamp {create_timestamp!r} is not valid ISO8601')
        self.assertRegex(
            schedule_timestamp, iso8601_pattern,
            f'Schedule timestamp {schedule_timestamp!r} is not valid ISO8601')

        # Verify both timestamps are identical (stability within replace_placeholders call)
        self.assertEqual(
            create_timestamp, schedule_timestamp,
            'Create and schedule timestamps must be identical within same replace_placeholders() call'
        )

        # Verify resource name is resolved
        self.assertEqual(
            actual[3], '--resource-name=my-pipeline-2023-10-10-13-32-59-420710')


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
            '{{$.pipeline_job_resource_name}}',
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

    def test_pipeline_job_create_time_utc_placeholder(self):
        """Test that create and schedule timestamps are stable and match for
        the same run context.

        Verifies that multiple calls to resolve create/schedule time
        placeholders return the same timestamp value, ensuring the
        utc_timestamp is not regenerated per call. This catches
        regressions where datetime.now() might be called per placeholder
        resolution instead of using the stable utc_timestamp parameter.
        """
        utc_timestamp = datetime.datetime.now(
            datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%S.%fZ')
        iso8601_pattern = r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$'
        kwargs = dict(
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
            utc_timestamp=utc_timestamp,
        )
        # Resolve create time multiple times
        actual_create_1 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_create_time_utc}}', **kwargs)
        actual_create_2 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_create_time_utc}}', **kwargs)
        # Resolve schedule time multiple times
        actual_schedule_1 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_schedule_time_utc}}', **kwargs)
        actual_schedule_2 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_schedule_time_utc}}', **kwargs)

        # Verify format is valid
        self.assertRegex(actual_create_1, iso8601_pattern)
        self.assertRegex(actual_schedule_1, iso8601_pattern)

        # Verify stability: multiple calls to the same placeholder return the same value
        self.assertEqual(
            actual_create_1, actual_create_2,
            'Multiple calls to create_time_utc should return identical values')
        self.assertEqual(
            actual_schedule_1, actual_schedule_2,
            'Multiple calls to schedule_time_utc should return identical values'
        )

        # Verify semantic equivalence: create and schedule times are the same
        self.assertEqual(
            actual_create_1, actual_schedule_1,
            'Create and schedule timestamps must be identical for the same job context'
        )

    def test_pipeline_job_schedule_time_utc_placeholder(self):
        """Test that schedule time placeholder returns stable timestamps across
        multiple calls.

        Verifies that multiple resolutions of the schedule_time_utc
        placeholder return the same timestamp value within a single job
        context, ensuring the timestamp does not change between
        placeholder resolutions.
        """
        utc_timestamp = datetime.datetime.now(
            datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%S.%fZ')
        iso8601_pattern = r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$'
        kwargs = dict(
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
            utc_timestamp=utc_timestamp,
        )
        # Resolve schedule time multiple times to verify stability
        actual_1 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_schedule_time_utc}}', **kwargs)
        actual_2 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_schedule_time_utc}}', **kwargs)
        actual_3 = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_schedule_time_utc}}', **kwargs)

        # Verify format is valid ISO8601
        self.assertRegex(actual_1, iso8601_pattern)
        self.assertRegex(actual_2, iso8601_pattern)
        self.assertRegex(actual_3, iso8601_pattern)

        # Verify stability: all calls return the same timestamp
        self.assertEqual(
            actual_1, actual_2,
            'Multiple calls to schedule_time_utc should return identical values'
        )
        self.assertEqual(
            actual_2, actual_3,
            'Multiple calls to schedule_time_utc should return identical values'
        )

    def test_utc_timestamp_placeholder_without_explicit_timestamp(self):
        """Test behavior when UTC timestamp placeholders are resolved without
        explicit utc_timestamp.

        Documents the current behavior: when utc_timestamp parameter is not provided
        (defaults to empty string), the placeholder is replaced with an empty string.
        This is generally safe when called from replace_placeholders() which always
        provides a utc_timestamp, but direct calls to resolve_individual_placeholder()
        without utc_timestamp could produce unexpected empty values.

        This test serves as a regression check and documents the current contract.
        """
        # When utc_timestamp is not provided (defaults to ''), the placeholder
        # is replaced with an empty string
        actual = placeholder_utils.resolve_individual_placeholder(
            element='{{$.pipeline_job_create_time_utc}}',
            executor_input_dict=EXECUTOR_INPUT_DICT,
            pipeline_resource_name='my-pipeline-2023-10-10-13-32-59-420710',
            task_resource_name='comp',
            pipeline_root='/foo/bar/my-pipeline-2023-10-10-13-32-59-420710',
            pipeline_job_id='123456789',
            pipeline_task_id='987654321',
            # utc_timestamp is not provided, defaults to ''
        )
        # The placeholder is replaced with an empty string
        self.assertEqual(actual, '')


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


class TestWorkspacePlaceholderResolution(unittest.TestCase):
    """Tests for workspace placeholder resolution."""

    def setUp(self):
        """Set up test environment."""
        # Initialize local config for testing
        from kfp import local
        local.init(runner=local.SubprocessRunner())
        # Workspace prefix respects TMPDIR (via tempfile.mkdtemp).
        self._workspace_prefix = os.path.join(tempfile.gettempdir(),
                                              'kfp-workspace-')

    def test_workspace_path_placeholder_resolution(self):
        """Test that workspace path placeholder is correctly resolved."""
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'workspace_path': '{{$.workspace_path}}'
                }
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        result = placeholder_utils.resolve_individual_placeholder(
            element='{{$.workspace_path}}',
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        self.assertIsInstance(result, str)
        self.assertTrue(result.startswith(self._workspace_prefix))

    def test_embedded_workspace_placeholder(self):
        """Test embedded workspace placeholder resolution."""
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'file_path':
                        os.path.join('{{$.workspace_path}}', 'data', 'file.txt')
                }
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        result = placeholder_utils.resolve_individual_placeholder(
            element="os.path.join('{{$.workspace_path}}', 'data', 'file.txt')",
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        self.assertIsInstance(result, str)
        self.assertIn('os.path.join', result)
        self.assertIn(self._workspace_prefix, result)
        self.assertIn('data', result)
        self.assertIn('file.txt', result)

    def test_workspace_configured_resolves(self):
        """Test that workspace placeholder resolves when workspace is
        configured."""
        executor_input_dict = {
            'inputs': {
                'parameterValues': {
                    'workspace_path': '{{$.workspace_path}}'
                }
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        result = placeholder_utils.resolve_individual_placeholder(
            element='{{$.workspace_path}}',
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        # Should resolve to actual workspace path
        self.assertIsInstance(result, str)
        self.assertTrue(result.startswith(self._workspace_prefix))

    def test_literal_mount_path_replaced_for_subprocess(self):
        """'/kfp-workspace' should be replaced with host workspace in
        subprocess mode."""
        executor_input_dict = {
            'inputs': {
                'parameterValues': {}
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        element = f'{dsl_constants.WORKSPACE_MOUNT_PATH}/data/file.txt'
        result = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        self.assertIsInstance(result, str)
        self.assertTrue(result.startswith(self._workspace_prefix))
        self.assertTrue(result.endswith('/data/file.txt'))


class TestWorkspacePlaceholderResolutionDocker(unittest.TestCase):
    """DockerRunner-specific workspace placeholder resolution."""

    def setUp(self):
        from kfp import local
        local.init(runner=local.DockerRunner())

    def test_workspace_placeholder_maps_to_mount_path(self):
        executor_input_dict = {
            'inputs': {
                'parameterValues': {}
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        result = placeholder_utils.resolve_individual_placeholder(
            element='{{$.workspace_path}}/data/file.txt',
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        self.assertEqual(result,
                         f'{dsl_constants.WORKSPACE_MOUNT_PATH}/data/file.txt')

    def test_literal_mount_path_kept_for_docker(self):
        executor_input_dict = {
            'inputs': {
                'parameterValues': {}
            },
            'outputs': {
                'outputFile': '/tmp/outputs/output.txt'
            }
        }

        element = f'{dsl_constants.WORKSPACE_MOUNT_PATH}/data/file.txt'
        result = placeholder_utils.resolve_individual_placeholder(
            element=element,
            executor_input_dict=executor_input_dict,
            pipeline_resource_name='test-pipeline',
            task_resource_name='test-task',
            pipeline_root='/tmp/pipeline',
            pipeline_job_id='test-job-id',
            pipeline_task_id='test-task-id')

        self.assertEqual(result, element)


class TestWorkspacePlaceholderMissing(unittest.TestCase):

    def test_raises_when_workspace_not_configured(self):
        # When placeholder or mount path appears and no workspace configured, raise
        from unittest import mock
        with mock.patch('kfp.local.config.LocalExecutionConfig.instance', None):
            with self.assertRaises(RuntimeError):
                placeholder_utils.resolve_individual_placeholder(
                    element='{{$.workspace_path}}/foo',
                    executor_input_dict={'outputs': {
                        'outputFile': '/tmp/x'
                    }},
                    pipeline_resource_name='p',
                    task_resource_name='t',
                    pipeline_root='/tmp/root',
                    pipeline_job_id='j',
                    pipeline_task_id='k')


if __name__ == '__main__':
    unittest.main()
