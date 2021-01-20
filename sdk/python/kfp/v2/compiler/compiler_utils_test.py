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
"""Tests for kfp.v2.compiler.compiler_utils."""

import unittest

from kfp.v2 import dsl
from kfp.v2.compiler import compiler_utils
from kfp.pipeline_spec import pipeline_spec_pb2
from google.protobuf import json_format


class CompilerUtilsTest(unittest.TestCase):

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
        'gs://path', {'input1': 'test', 'input2': None})
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

if __name__ == '__main__':
  unittest.main()
