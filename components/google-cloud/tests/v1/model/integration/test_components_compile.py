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
"""Test google-cloud-pipeline-Components to ensure they compile without error."""

import json
import os
import unittest
import kfp
from kfp.v2 import compiler
from google.cloud import aiplatform
from google_cloud_pipeline_components.v1.model import (
    ModelExportOp,
    ModelUploadOp,
)


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._display_name = "test_display_name"
    self._serving_container_image_uri = "gcr.io/test_project/test_image:test_tag"
    self._artifact_uri = "project/test_artifact_uri"
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json")

  def tearDown(self):
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_model_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      # TODO(b/219835305): Reenable the importer node when fix.
      # # Using importer and UnmanagedContainerModel artifact for model upload
      # # component. This is the recommended approach going forward.
      # unmanaged_model_importer = importer_node.importer(
      #     artifact_uri=self._artifact_uri,
      #     artifact_class=artifact_types.UnmanagedContainerModel,
      #     metadata={
      #         "containerSpec": {
      #             "imageUri": self._serving_container_image_uri
      #         }
      #     })

      model_upload_op = ModelUploadOp(
          project=self._project,
          display_name=self._display_name,
          # unmanaged_container_model=unmanaged_model_importer.outputs['artifact']
      )

      model_export_op = ModelExportOp(
          model=model_upload_op.outputs["model"],
          export_format_id="export_format",
          artifact_destination="artifact_destination",
          image_destination="image_destination")

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/model_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
