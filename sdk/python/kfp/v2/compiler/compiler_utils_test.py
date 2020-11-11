# Copyright 2020 Google LLC
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

import unittest

from kfp.v2 import dsl
from kfp.v2.compiler import compiler_utils
from kfp.pipeline_spec import pipeline_spec_pb2
from google.protobuf import json_format


class CompilerUtilsTest(unittest.TestCase):

  def test_build_runtime_parameter_spec(self):
    pipeline_params = [
        dsl.PipelineParam(name='input1', param_type='Integer', value=99),
        dsl.PipelineParam(name='input2', param_type='String', value='hello'),
        dsl.PipelineParam(name='input3', param_type='Float', value=3.1415926),
        dsl.PipelineParam(name='input4', param_type=None, value=None),
    ]
    expected_dict = {
        'runtimeParameters': {
            'input1': {
                'type': 'INT',
                'defaultValue': {
                    'intValue': '99'
                }
            },
            'input2': {
                'type': 'STRING',
                'defaultValue': {
                    'stringValue': 'hello'
                }
            },
            'input3': {
                'type': 'DOUBLE',
                'defaultValue': {
                    'doubleValue': '3.1415926'
                }
            },
            'input4': {
                'type': 'STRING'
            }
        }
    }
    expected_spec = pipeline_spec_pb2.PipelineSpec()
    json_format.ParseDict(expected_dict, expected_spec)

    pipeline_spec = pipeline_spec_pb2.PipelineSpec(
        runtime_parameters=compiler_utils.build_runtime_parameter_spec(
            pipeline_params))
    self.maxDiff = None
    self.assertEqual(expected_spec, pipeline_spec)

  def test_build_runtime_parameter_spec_with_unsupported_type_should_fail(self):
    pipeline_params = [
        dsl.PipelineParam(name='input1', param_type='Dict'),
    ]

    with self.assertRaisesRegexp(
        TypeError, 'Unsupported type "Dict" for argument "input1"'):
      compiler_utils.build_runtime_parameter_spec(pipeline_params)

  def test_build_runtime_config_spec(self):
    expected_dict = {
        'gcsOutputDirectory': 'gs://path',
        'parameters': {
            'input1': {
                'stringValue': 'test'
            }
        }
    }
    expected_spec = pipeline_spec_pb2.PipelineJob.RuntimeConfig()
    json_format.ParseDict(expected_dict, expected_spec)

    runtime_config = compiler_utils.build_runtime_config_spec(
        'gs://path', {'input1': 'test'})
    self.assertEqual(expected_spec, runtime_config)


if __name__ == '__main__':
  unittest.main()
