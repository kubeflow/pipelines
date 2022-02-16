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
"""Test google-cloud-pipeline-Components to ensure the compile without error."""

import json
import os
import unittest
import kfp
from kfp.v2 import compiler
from google.cloud import aiplatform
from google_cloud_pipeline_components.v1.endpoint import (
    EndpointCreateOp,
    ModelDeployOp,
)
from google_cloud_pipeline_components.v1.model import ModelUploadOp


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._display_name = "test_display_name"
    self._model_display_name = "test_model_display_name"
    self._gcs_source = "gs://test_gcs_source"
    self._gcs_output_dir = "gs://test_gcs_output_dir"
    self._pipeline_root = "gs://test_pipeline_root"
    self._gcs_destination_prefix = "gs://test_gcs_output_dir/batch_prediction"
    self._serving_container_image_uri = "gcr.io/test_project/test_image:test_tag"
    self._artifact_uri = "project/test_artifact_uri"
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json")

  def tearDown(self):
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_create_endpoint_op_compile(self):

    @kfp.dsl.pipeline(name="delete-endpoint-test")
    def pipeline():
      create_endpoint_op = EndpointCreateOp(
          project=self._project,
          location=self._location,
          display_name=self._display_name,
          description="some description",
          labels={"foo": "bar"},
          network="abc",
          encryption_spec_key_name="some encryption_spec_key_name")

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/create_endpoint_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_model_deploy_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      model_upload_op = ModelUploadOp(
          project=self._project,
          display_name=self._display_name,
          serving_container_image_uri=self._serving_container_image_uri,
          artifact_uri=self._artifact_uri)

      create_endpoint_op = EndpointCreateOp(
          project=self._project,
          location=self._location,
          display_name=self._display_name)

      model_deploy_op = ModelDeployOp(
          model=model_upload_op.outputs["model"],
          endpoint=create_endpoint_op.outputs["endpoint"],
          deployed_model_display_name="deployed_model_display_name",
          traffic_split={},
          dedicated_resources_machine_type="n1-standard-4",
          dedicated_resources_min_replica_count=1,
          dedicated_resources_max_replica_count=2,
          dedicated_resources_accelerator_type="fake-accelerator",
          dedicated_resources_accelerator_count=1,
          automatic_resources_min_replica_count=1,
          automatic_resources_max_replica_count=2,
          service_account="fake-sa",
          explanation_metadata={"xai_m": "bar"},
          explanation_parameters={"xai_p": "foo"},
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/model_deploy_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
