# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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

import os

from google_cloud_pipeline_components.types import artifact_types
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from google_cloud_pipeline_components.v1.model import ModelUploadOp
from google_cloud_pipeline_components.tests.v1 import utils
import kfp
from kfp.dsl import importer

import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._display_name = "test_display_name"
    self._gcs_source = "gs://test_gcs_source"
    self._gcs_destination_prefix = "gs://test_gcs_output_dir/batch_prediction"
    self._serving_container_image_uri = (
        "gcr.io/test_project/test_image:test_tag"
    )
    self._artifact_uri = "project/test_artifact_uri"
    self._package_path = os.path.join(
        os.path.dirname(__file__), "pipeline.yaml"
    )

  def tearDown(self):
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_batch_prediction_op_compile(self):
    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      # Using importer and UnmanagedContainerModel artifact for model upload
      # component. This is the recommended approach going forward.
      unmanaged_model_importer = importer(
          artifact_uri=self._artifact_uri,
          artifact_class=artifact_types.UnmanagedContainerModel,
          metadata={
              "containerSpec": {"imageUri": self._serving_container_image_uri}
          },
      )

      model_upload_op = ModelUploadOp(
          project=self._project,
          display_name=self._display_name,
          unmanaged_container_model=unmanaged_model_importer.outputs[
              "artifact"
          ],
      )

      batch_predict_op = ModelBatchPredictOp(
          project=self._project,
          location=self._location,
          job_display_name=self._display_name,
          model=model_upload_op.outputs["model"],
          instances_format="instance_format",
          gcs_source_uris=[self._gcs_source],
          bigquery_source_input_uri="bigquery_source_input_uri",
          instance_type="instance_type",
          key_field="key_field",
          included_fields=["field1", "field2"],
          excluded_fields=["field3", "field4"],
          model_parameters={"foo": "bar"},
          predictions_format="predictions_format",
          gcs_destination_output_uri_prefix=self._gcs_destination_prefix,
          bigquery_destination_output_uri="bigquery_destination_output_uri",
          machine_type="machine_type",
          unmanaged_container_model=unmanaged_model_importer.outputs[
              "artifact"
          ],
          accelerator_type="accelerator_type",
          accelerator_count=1,
          starting_replica_count=2,
          max_replica_count=3,
          manual_batch_tuning_parameters_batch_size=4,
          generate_explanation=True,
          explanation_metadata={"xai_m": "bar"},
          explanation_parameters={"xai_p": "foo"},
          encryption_spec_key_name="some encryption_spec_key_name",
          labels={"foo": "bar"},
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "batch_prediction_pipeline.yaml",
        ),
    )
