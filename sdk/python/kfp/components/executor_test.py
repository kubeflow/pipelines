# Copyright 2021 Google LLC
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
"""Tests for kfp.components.executor"""

import os
import tempfile
from typing import Callable
import unittest
import json

from kfp.components import executor, InputPath, OutputPath
from kfp.dsl import io_types
from kfp.dsl.io_types import Dataset, InputArtifact, Metrics, Model, OutputArtifact

_EXECUTOR_INPUT = """\
{
  "inputs": {
    "parameters": {
      "input_parameter": {
        "stringValue": "Hello, KFP"
      }
    },
    "artifacts": {
      "input_artifact_one": {
        "artifacts": [
          {
            "metadata": {},
            "name": "input_artifact_one",
            "type": {
              "schemaTitle": "kfp.Dataset"
            },
            "uri": "gs://some-bucket/input_artifact_one"
          }
        ]
      }
    }
  },
  "outputs": {
    "artifacts": {
      "output_artifact_one": {
        "artifacts": [
          {
            "metadata": {},
            "name": "output_artifact_one",
            "type": {
              "schemaTitle": "kfp.Model"
            },
            "uri": "gs://some-bucket/output_artifact_one"
          }
        ]
      },
      "output_artifact_two": {
        "artifacts": [
          {
            "metadata": {},
            "name": "output_artifact_two",
            "type": {
              "schemaTitle": "kfp.Metrics"
            },
            "uri": "gs://some-bucket/output_artifact_two"
          }
        ]
      }
    },
    "parameters": {
      "output_parameter": {
        "outputFile": "gs://some-bucket/output_parameter"
      }
    },
    "outputFile": "%s/output_metadata.json"
  }
}
"""


class ExecutorTest(unittest.TestCase):

  def setUp(self):
    self.maxDiff = None
    self._test_dir = tempfile.mkdtemp()
    io_types._GCS_LOCAL_MOUNT_PREFIX = self._test_dir + '/'
    self._executor_input = json.loads(_EXECUTOR_INPUT % self._test_dir)
    return super().setUp()

  def _get_executor(self, func: Callable) -> executor.Executor:
    return executor.Executor(executor_input=self._executor_input,
                             function_to_execute=func)

  def test_input_parameter(self):

    def test_func(input_parameter: str):
      self.assertEqual(input_parameter, "Hello, KFP")

    self._get_executor(test_func).execute()

  def test_input_artifact(self):

    def test_func(input_artifact_one: InputArtifact(Dataset)):
      self.assertEqual(input_artifact_one.uri,
                       'gs://some-bucket/input_artifact_one')
      self.assertEqual(
          input_artifact_one.path,
          os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))
      self.assertEqual(input_artifact_one.get().name, 'input_artifact_one')

    self._get_executor(test_func).execute()

  def test_output_artifact(self):

    def test_func(output_artifact_one: OutputArtifact(Model)):
      # Test that output artifacts always have filename 'data' added.
      self.assertEqual(output_artifact_one.uri,
                       'gs://some-bucket/output_artifact_one/data')

      self.assertEqual(
          output_artifact_one.path,
          os.path.join(self._test_dir, 'some-bucket/output_artifact_one',
                       'data'))
      self.assertEqual(output_artifact_one.get().name, 'output_artifact_one')

    self._get_executor(test_func).execute()

  def test_output_parameter(self):

    def test_func(output_parameter_path: OutputPath(str)):
      # Test that output parameters just use the passed in filename.
      self.assertEqual(output_parameter_path,
                       'gs://some-bucket/output_parameter')

    self._get_executor(test_func).execute()

  def test_input_path_artifact(self):

    def test_func(input_artifact_one_path: InputPath('Dataset')):
      self.assertEqual(
          input_artifact_one_path,
          os.path.join(self._test_dir, 'some-bucket/input_artifact_one'))

    self._get_executor(test_func).execute()

  def test_output_path_artifact(self):

    def test_func(output_artifact_one_path: OutputPath('Model')):
      # Test that output path also get 'data' appended.
      self.assertEqual(
          output_artifact_one_path,
          os.path.join(self._test_dir, 'some-bucket/output_artifact_one/data'))

    self._get_executor(test_func).execute()

  def test_output_metadata(self):

    def test_func(output_artifact_two: OutputArtifact(Metrics)):
      output_artifact_two.get().metadata['key_1'] = 'value_1'
      output_artifact_two.get().metadata['key_2'] = 2
      output_artifact_two.uri = 'new-uri'

      # log_metric works here since the schema is specified as Metrics.
      output_artifact_two.get().log_metric('metric', 0.9)

    self._get_executor(test_func).execute()
    with open(os.path.join(self._test_dir, 'output_metadata.json'), 'r') as f:
      output_metadata = json.loads(f.read())
    print(output_metadata)
    self.assertDictEqual(
        output_metadata, {
            'artifacts': {
                'output_artifact_one': {
                    'artifacts': [{
                        'name': 'output_artifact_one',
                        'uri': 'gs://some-bucket/output_artifact_one/data',
                        'metadata': {}
                    }]
                },
                'output_artifact_two': {
                    'artifacts': [{
                        'name': 'output_artifact_two',
                        'uri': 'new-uri',
                        'metadata': {
                            'key_1': 'value_1',
                            'key_2': 2,
                            'metric': 0.9
                        }
                    }]
                }
            }
        })
