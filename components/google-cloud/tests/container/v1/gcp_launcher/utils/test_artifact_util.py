# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
import time
import unittest
from unittest import mock
from google_cloud_pipeline_components.container.v1.gcp_launcher import custom_job_remote_runner
from google.cloud import aiplatform
from google.cloud.aiplatform.compat.types import job_state as gca_job_state
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.types.artifact_types import VertexModel


class ArtifactUtilTests(unittest.TestCase):

  def setUp(self):
    super(ArtifactUtilTests, self).setUp()
    self._output_file_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'localpath/foo')

  def tearDown(self):
    if os.path.exists(self._output_file_path):
      os.remove(self._output_file_path)

  def test_update_output_artifacts(self):
    executor_input = '{"outputs":{"artifacts":{"model":{"artifacts":[{"metadata":{},"name":"foobar","type":{"schemaTitle":"google.VertexModel"},"uri":"gs://abc"}]}},"outputFile":"' + self._output_file_path + '"}}'
    vertex_model = VertexModel('model', 'https://new/uri',
                               'fake_model_resource_name')
    artifact_util.update_output_artifacts(executor_input, [vertex_model])

    with open(self._output_file_path) as f:
      executor_output = json.load(f, strict=False)
      self.assertEqual(
          executor_output,
          json.loads(
              '{"artifacts": {"model": {"artifacts": [{"metadata": {"resourceName": "fake_model_resource_name"}, "name": "foobar", "type": {"schemaTitle": "google.VertexModel"}, "uri": "https://new/uri"}]}}}'
          ))
