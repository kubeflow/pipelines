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
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from google_cloud_pipeline_components.v1.model import ModelUploadOp


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._display_name = "test_display_name"
    self._gcs_source = "gs://test_gcs_source"
    self._gcs_destination_prefix = "gs://test_gcs_output_dir/batch_prediction"
    self._serving_container_image_uri = "gcr.io/test_project/test_image:test_tag"
    self._artifact_uri = "project/test_artifact_uri"
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json")

  def tearDown(self):
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_batch_prediction_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():

      model_upload_op = ModelUploadOp(
          project=self._project,
          display_name=self._display_name,
          serving_container_image_uri=self._serving_container_image_uri,
          artifact_uri=self._artifact_uri)

      batch_predict_op = ModelBatchPredictOp(
          project=self._project,
          location=self._location,
          job_display_name=self._display_name,
          model=model_upload_op.outputs["model"],
          instances_format="instance_format",
          gcs_source_uris=[self._gcs_source],
          bigquery_source_input_uri="bigquery_source_input_uri",
          model_parameters={"foo": "bar"},
          predictions_format="predictions_format",
          gcs_destination_output_uri_prefix=self._gcs_destination_prefix,
          bigquery_destination_output_uri="bigquery_destination_output_uri",
          machine_type="machine_type",
          accelerator_type="accelerator_type",
          accelerator_count=1,
          starting_replica_count=2,
          max_replica_count=3,
          manual_batch_tuning_parameters_batch_size=4,
          generate_explanation=True,
          explanation_metadata={"xai_m": "bar"},
          explanation_parameters={"xai_p": "foo"},
          encryption_spec_key_name="some encryption_spec_key_name",
          labels={"foo": "bar"})

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/batch_prediction_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
