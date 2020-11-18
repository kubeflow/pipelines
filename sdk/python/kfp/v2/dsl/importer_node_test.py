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
from kfp.v2.dsl import importer_node
from kfp.pipeline_spec import pipeline_spec_pb2 as pb
from google.protobuf import json_format


class ImporterNodeTest(unittest.TestCase):

  def test_build_importer_task(self):
    dependent_task = {
        'taskInfo': {
            'name': 'task1'
        },
        'inputs': {
            'artifacts': {
                'input1': {
                    'producerTask': '',
                }
            }
        },
        'executorLabel': 'task1_input1_importer'
    }
    dependent_task_spec = pb.PipelineTaskSpec()
    json_format.ParseDict(dependent_task, dependent_task_spec)

    expected_task = {
        'taskInfo': {
            'name': 'task1_input1_importer'
        },
        'outputs': {
            'artifacts': {
                'result': {
                    'artifactType': {
                        'instanceSchema': 'title: kfp.Artifact'
                    }
                }
            }
        },
        'executorLabel': 'task1_input1_importer'
    }
    expected_task_spec = pb.PipelineTaskSpec()
    json_format.ParseDict(expected_task, expected_task_spec)

    task_spec = importer_node.build_importer_task_spec(
        dependent_task=dependent_task_spec,
        input_name='input1',
        input_type_schema='title: kfp.Artifact')

    self.maxDiff = None
    self.assertEqual(expected_task_spec, task_spec)

  def test_build_importer_spec_from_pipeline_param(self):
    expected_importer = {
        'artifactUri': {
            'runtimeParameter': 'param1'
        },
        'typeSchema': {
            'instanceSchema': 'title: kfp.Artifact'
        }
    }
    expected_importer_spec = pb.PipelineDeploymentConfig.ImporterSpec()
    json_format.ParseDict(expected_importer, expected_importer_spec)
    importer_spec = importer_node.build_importer_spec(
        input_type_schema='title: kfp.Artifact', pipeline_param_name='param1')

    self.maxDiff = None
    self.assertEqual(expected_importer_spec, importer_spec)

  def test_build_importer_spec_from_constant_value(self):
    expected_importer = {
        'artifactUri': {
            'constantValue': {
                'stringValue': 'some_uri'
            }
        },
        'typeSchema': {
            'instanceSchema': 'title: kfp.Artifact'
        }
    }
    expected_importer_spec = pb.PipelineDeploymentConfig.ImporterSpec()
    json_format.ParseDict(expected_importer, expected_importer_spec)
    importer_spec = importer_node.build_importer_spec(
        input_type_schema='title: kfp.Artifact', constant_value='some_uri')

    self.maxDiff = None
    self.assertEqual(expected_importer_spec, importer_spec)

  def test_build_importer_spec_with_invalid_inputs_should_fail(self):
    with self.assertRaisesRegexp(
        AssertionError,
        'importer spec should be built using either pipeline_param_name or '
        'constant_value'):
      importer_node.build_importer_spec(
          input_type_schema='title: kfp.Artifact',
          pipeline_param_name='param1',
          constant_value='some_uri')

    with self.assertRaisesRegexp(
        AssertionError,
        'importer spec should be built using either pipeline_param_name or '
        'constant_value'):
      importer_node.build_importer_spec(input_type_schema='title: kfp.Artifact')


if __name__ == '__main__':
  unittest.main()
