# Copyright 2020-2022 The Kubeflow Authors
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
"""Tests for kfp.v2.compiler.compiler_utils."""

import unittest

from kfp.dsl import _pipeline_param
from kfp.v2.compiler import compiler_utils
from kfp.pipeline_spec import pipeline_spec_pb2
from google.protobuf import json_format
from google.protobuf import message
from absl.testing import parameterized


class CompilerUtilsTest(unittest.TestCase):

    def assertProtoEquals(self, proto1: message.Message,
                          proto2: message.Message):
        """Asserts the equality between two messages."""
        self.assertDictEqual(
            json_format.MessageToDict(proto1),
            json_format.MessageToDict(proto2))

    def test_build_runtime_config_spec(self):
        expected_dict = {
            'gcsOutputDirectory': 'gs://path',
            'parameters': {
                'input1': {
                    'stringValue': 'test'
                },
                'input2': {
                    'intValue': 2
                },
                'input3': {
                    'stringValue': '[1, 2, 3]'
                }
            }
        }
        expected_spec = pipeline_spec_pb2.PipelineJob.RuntimeConfig()
        json_format.ParseDict(expected_dict, expected_spec)

        runtime_config = compiler_utils.build_runtime_config_spec(
            'gs://path', {
                'input1':
                    _pipeline_param.PipelineParam(
                        name='input1', param_type='String', value='test'),
                'input2':
                    _pipeline_param.PipelineParam(
                        name='input2', param_type='Integer', value=2),
                'input3':
                    _pipeline_param.PipelineParam(
                        name='input3', param_type='List', value=[1, 2, 3]),
                'input4':
                    _pipeline_param.PipelineParam(
                        name='input4', param_type='Double', value=None)
            })
        self.assertEqual(expected_spec, runtime_config)

    def test_validate_pipeline_name(self):
        compiler_utils.validate_pipeline_name('my-pipeline')

        compiler_utils.validate_pipeline_name('p' * 128)

        with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
            compiler_utils.validate_pipeline_name('my_pipeline')

        with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
            compiler_utils.validate_pipeline_name('My pipeline')

        with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
            compiler_utils.validate_pipeline_name('-my-pipeline')

        with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
            compiler_utils.validate_pipeline_name('p' * 129)

    def test_refactor_v2_component_success(self):
        test_v2_container_spec = compiler_utils.PipelineContainerSpec(
            image='my/dummy-image',
            command=['python', '-m', 'my_package.my_entrypoint'],
            args=['arg1', 'arg2', '--function_name', 'test_func'])
        expected_container_spec = compiler_utils.PipelineContainerSpec(
            image='my/dummy-image',
            command=['python', '-m', 'kfp.container.entrypoint'],
            args=[
                '--executor_input_str', '{{$}}', '--function_name', 'test_func'
            ])
        compiler_utils.refactor_v2_container_spec(test_v2_container_spec)
        self.assertProtoEquals(expected_container_spec, test_v2_container_spec)


class TestNormalizeTimeString(parameterized.TestCase):

    @parameterized.parameters([
        ('1 hour', '1h'),
        ('2 hours', '2h'),
        ('2hours', '2h'),
        ('2 w', '2w'),
        ('2d', '2d'),
    ])
    def test(self, unnorm: str, norm: str):
        self.assertEqual(compiler_utils.normalize_time_string(unnorm), norm)

    def test_multipart_duration_raises(self):
        with self.assertRaisesRegex(ValueError, 'Invalid duration string:'):
            compiler_utils.convert_duration_to_seconds('1 day 1 hour')

    def test_non_int_value_raises(self):
        with self.assertRaisesRegex(ValueError, 'Invalid duration string:'):
            compiler_utils.convert_duration_to_seconds('one hour')


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
            compiler_utils.convert_duration_to_seconds(duration), seconds)

    def test_unsupported_duration_unit(self):
        with self.assertRaisesRegex(ValueError, 'Unsupported duration unit:'):
            compiler_utils.convert_duration_to_seconds('1 year')


class TestMakeRetryPolicyProto(unittest.TestCase):

    def test_defaults(self):
        retry_policy_proto = compiler_utils.make_retry_policy_proto()
        self.assertEqual(retry_policy_proto.max_retry_count, 0)
        self.assertEqual(retry_policy_proto.backoff_duration.seconds, 0)
        self.assertEqual(retry_policy_proto.backoff_factor, 2.0)
        self.assertEqual(retry_policy_proto.backoff_max_duration.seconds, 3600)

    def test_with_args(self):
        retry_policy_proto = compiler_utils.make_retry_policy_proto(
            max_retry_count=10,
            backoff_duration='1h',
            backoff_factor=1.5,
            backoff_max_duration='2 weeks')
        self.assertEqual(retry_policy_proto.max_retry_count, 10)
        self.assertEqual(retry_policy_proto.backoff_duration.seconds, 3600)
        self.assertEqual(retry_policy_proto.backoff_factor, 1.5)
        # tests cap
        self.assertEqual(retry_policy_proto.backoff_max_duration.seconds, 3600)


if __name__ == '__main__':
    unittest.main()
