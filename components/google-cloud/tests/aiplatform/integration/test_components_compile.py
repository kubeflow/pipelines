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
from kfp import compiler
from google.cloud import aiplatform
from google_cloud_pipeline_components.aiplatform import (
    ImageDatasetCreateOp,
    AutoMLImageTrainingJobRunOp,
    AutoMLTabularTrainingJobRunOp,
    TabularDatasetCreateOp,
    ImageDatasetExportDataOp,
    ImageDatasetImportDataOp,
    TabularDatasetExportDataOp,
    AutoMLTextTrainingJobRunOp,
    TextDatasetCreateOp,
    TextDatasetExportDataOp,
    TextDatasetImportDataOp,
    AutoMLVideoTrainingJobRunOp,
    VideoDatasetCreateOp,
    VideoDatasetExportDataOp,
    VideoDatasetImportDataOp,
    ModelBatchPredictOp,
    EndpointCreateOp,
    EndpointDeleteOp,
    ModelDeployOp,
    ModelExportOp,
    ModelUploadOp,
    ModelUndeployOp,
    ModelDeleteOp,
    TimeSeriesDatasetCreateOp,
    TimeSeriesDatasetExportDataOp,
    AutoMLForecastingTrainingJobRunOp,
)


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

  def test_automl_image_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = ImageDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.image
          .single_label_classification,
      )

      training_job_run_op = AutoMLImageTrainingJobRunOp(
          project=self._project,
          display_name=self._display_name,
          prediction_type="classification",
          model_type="CLOUD",
          dataset=dataset_create_op.outputs["dataset"],
          model_display_name=self._model_display_name,
          training_fraction_split=0.6,
          validation_fraction_split=0.2,
          test_fraction_split=0.2,
          budget_milli_node_hours=8000,
      )

      dataset_export_op = ImageDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

      dataset_import_op = ImageDatasetImportDataOp(
          project=self._project,
          gcs_source=self._gcs_source,
          dataset=dataset_create_op.outputs["dataset"],
          import_schema_uri=aiplatform.schema.dataset.ioformat.image
          .single_label_classification)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/automl_image_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_automl_tabular_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TabularDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
      )

      training_job_run_op = AutoMLTabularTrainingJobRunOp(
          project=self._project,
          display_name=self._display_name,
          optimization_prediction_type="regression",
          optimization_objective="minimize-rmse",
          column_transformations=[
              {
                  "numeric": {
                      "column_name": "longitude"
                  }
              },
          ],
          target_column="longitude",
          dataset=dataset_create_op.outputs["dataset"],
          model_version_description="description",
      )

      dataset_export_op = TabularDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

    self.assertFalse(os.path.exists(self._package_path))
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    self.assertTrue(os.path.exists(self._package_path))

  def test_automl_text_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TextDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.text
          .multi_label_classification,
      )

      training_job_run_op = AutoMLTextTrainingJobRunOp(
          project=self._project,
          display_name=self._display_name,
          dataset=dataset_create_op.outputs["dataset"],
          prediction_type="classification",
          multi_label=True,
          training_fraction_split=0.6,
          validation_fraction_split=0.2,
          test_fraction_split=0.2,
          model_display_name=self._model_display_name,
      )

      dataset_export_op = TextDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

      dataset_import_op = TextDatasetImportDataOp(
          project=self._project,
          gcs_source=self._gcs_source,
          dataset=dataset_create_op.outputs["dataset"],
          import_schema_uri=aiplatform.schema.dataset.ioformat.text
          .multi_label_classification)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/automl_text_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_automl_video_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = VideoDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.video
          .classification,
      )

      training_job_run_op = AutoMLVideoTrainingJobRunOp(
          project=self._project,
          display_name=self._display_name,
          model_type="CLOUD",
          dataset=dataset_create_op.outputs["dataset"],
          prediction_type="classification",
          training_fraction_split=0.6,
          test_fraction_split=0.2,
          model_display_name=self._model_display_name,
      )

      dataset_export_op = VideoDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

      dataset_import_op = VideoDatasetImportDataOp(
          project=self._project,
          gcs_source=self._gcs_source,
          dataset=dataset_create_op.outputs["dataset"],
          import_schema_uri=aiplatform.schema.dataset.ioformat.video
          .classification)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/automl_video_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_automl_forecasting_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TimeSeriesDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
      )

      training_job_run_op = AutoMLForecastingTrainingJobRunOp(
          project=self._project,
          display_name=self._display_name,
          dataset=dataset_create_op.outputs["dataset"],
          optimization_objective="optimization_objective",
          training_fraction_split=0.6,
          test_fraction_split=0.2,
          model_display_name=self._model_display_name,
          target_column="target_column",
          time_column="time_column",
          time_series_identifier_column="time_series_identifier_column",
          unavailable_at_forecast_columns=[],
          available_at_forecast_columns=[],
          forecast_horizon=12,
          data_granularity_unit="day",
          data_granularity_count=1,
          holiday_regions=["GLOBAL"],
          hierarchy_group_total_weight=1.0,
          window_stride_length=1,
          model_version_description="description",
      )

      dataset_export_op = TimeSeriesDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

    self.assertFalse(os.path.exists(self._package_path))
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    self.assertTrue(os.path.exists(self._package_path))

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

  def test_model_upload_and_model_delete_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      model_upload_op = ModelUploadOp(
          project=self._project,
          location=self._location,
          display_name=self._display_name,
          description="some description",
          serving_container_image_uri=self._serving_container_image_uri,
          serving_container_command=["command1", "command2"],
          serving_container_args=["arg1", "arg2"],
          serving_container_environment_variables=["env1", "env2"],
          serving_container_ports=["123", "456"],
          serving_container_predict_route="some serving_container_predict_route",
          serving_container_health_route="some serving_container_health_route",
          instance_schema_uri="some instance_schema_uri",
          parameters_schema_uri="some parameters_schema_uri",
          prediction_schema_uri="some prediction_schema_uri",
          artifact_uri="some artifact_uri",
          explanation_metadata={"xai_m": "bar"},
          explanation_parameters={"xai_p": "foo"},
          encryption_spec_key_name="some encryption_spec_key_name",
          labels={"foo": "bar"})

      _ = ModelDeleteOp(
          model=model_upload_op.outputs["model"],
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/model_upload_and_delete_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_create_endpoint_op_and_delete_endpoint_op_compile(self):

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

      delete_endpoint_op = EndpointDeleteOp(
          endpoint=create_endpoint_op.outputs["endpoint"])

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/create_and_delete_endpoint_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_model_export_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      model_upload_op = ModelUploadOp(
          project=self._project,
          display_name=self._display_name,
          serving_container_image_uri=self._serving_container_image_uri,
          artifact_uri=self._artifact_uri)

      model_export_op = ModelExportOp(
          model=model_upload_op.outputs["model"],
          export_format_id="export_format",
          artifact_destination="artifact_destination",
          image_destination="image_destination")

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/model_export_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_model_deploy_op_and_model_undeploy_op_compile(self):

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

      _ = ModelUndeployOp(
          model=model_upload_op.outputs["model"],
          endpoint=create_endpoint_op.outputs["endpoint"],
      ).after(model_deploy_op)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/model_deploy_and_undeploy_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
