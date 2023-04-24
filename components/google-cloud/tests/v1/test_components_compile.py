# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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

import kfp
from google_cloud_pipeline_components.v1.automl.training_job import (
    AutoMLForecastingTrainingJobRunOp,
    AutoMLImageTrainingJobRunOp,
    AutoMLTabularTrainingJobRunOp,
    AutoMLTextTrainingJobRunOp,
    AutoMLVideoTrainingJobRunOp,
)
from google_cloud_pipeline_components.v1.dataset import (
    ImageDatasetCreateOp,
    ImageDatasetExportDataOp,
    ImageDatasetImportDataOp,
    TabularDatasetCreateOp,
    TabularDatasetExportDataOp,
    TextDatasetCreateOp,
    TextDatasetExportDataOp,
    TextDatasetImportDataOp,
    TimeSeriesDatasetCreateOp,
    TimeSeriesDatasetExportDataOp,
    VideoDatasetCreateOp,
    VideoDatasetExportDataOp,
    VideoDatasetImportDataOp,
)
from kfp import compiler

import unittest
from google.cloud import aiplatform


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
    self._serving_container_image_uri = (
        "gcr.io/test_project/test_image:test_tag"
    )
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json"
    )

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
          import_schema_uri=aiplatform.schema.dataset.ioformat.image.single_label_classification,
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
          import_schema_uri=aiplatform.schema.dataset.ioformat.image.single_label_classification,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path
    )
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
              {"numeric": {"column_name": "longitude"}},
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
        pipeline_func=pipeline, package_path=self._package_path
    )
    self.assertTrue(os.path.exists(self._package_path))

  def test_automl_text_component_compile(self):
    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TextDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.text.multi_label_classification,
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
          import_schema_uri=aiplatform.schema.dataset.ioformat.text.multi_label_classification,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path
    )
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
          import_schema_uri=aiplatform.schema.dataset.ioformat.video.classification,
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
          import_schema_uri=aiplatform.schema.dataset.ioformat.video.classification,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path
    )
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
        pipeline_func=pipeline, package_path=self._package_path
    )
    self.assertTrue(os.path.exists(self._package_path))
