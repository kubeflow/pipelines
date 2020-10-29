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

import json
import unittest
from kfp.v2.compiler import importer_node
from kfp.v2.proto import pipeline_spec_pb2 as pb
from google.protobuf import json_format


class ImporterNodeTest(unittest.TestCase):

  def test_build_importer_spec(self):

    dependent_task = {
        'taskInfo': {
            'name': 'task1'
        },
        'inputs': {
            'artifacts': {
                'input1': {
                    'producerTask': '',
                    'outputArtifactKey': 'output1'
                }
            }
        },
        'executorLabel': 'task1_input1_importer'
    }
    dependent_task_spec = pb.PipelineTaskSpec()
    json_format.Parse(json.dumps(dependent_task), dependent_task_spec)

    expected_task = {
        'taskInfo': {
            'name': 'task1_input1_importer'
        },
        'outputs': {
            'artifacts': {
                'result': {
                    'artifactType': {
                        'instanceSchema': 'title: Artifact'
                    }
                }
            }
        },
        'executorLabel': 'task1_input1_importer'
    }
    expected_task_spec = pb.PipelineTaskSpec()
    json_format.Parse(json.dumps(expected_task), expected_task_spec)

    expected_importer = {
        'artifactUri': {
            'runtimeParameter': 'output1'
        },
        'typeSchema': {
            'instanceSchema': 'title: Artifact'
        }
    }
    expected_importer_spec = pb.PipelineDeploymentConfig.ImporterSpec()
    json_format.Parse(json.dumps(expected_importer), expected_importer_spec)

    task_spec, importer_spec = importer_node.build_importer_spec(
        dependent_task_spec, 'input1', 'title: Artifact')

    self.maxDiff = None
    self.assertEqual(expected_task_spec, task_spec)
    self.assertEqual(expected_importer_spec, importer_spec)


if __name__ == '__main__':
  unittest.main()
