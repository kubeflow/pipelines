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
from google_cloud_pipeline_components.v1.dataset import (
    ImageDatasetCreateOp,
    TabularDatasetCreateOp,
    ImageDatasetExportDataOp,
    ImageDatasetImportDataOp,
    TabularDatasetExportDataOp,
    TextDatasetCreateOp,
    TextDatasetExportDataOp,
    TextDatasetImportDataOp,
    VideoDatasetCreateOp,
    VideoDatasetExportDataOp,
    VideoDatasetImportDataOp,
    TimeSeriesDatasetCreateOp,
    TimeSeriesDatasetExportDataOp,
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

  def test_image_dataset_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = ImageDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.image
          .single_label_classification,
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
    with open("testdata/image_dataset_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_tabular_dataset_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TabularDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
      )

      dataset_export_op = TabularDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/tabular_dataset_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_text_dataset_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TextDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.text
          .multi_label_classification,
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
    with open("testdata/text_dataset_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_video_dataset_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = VideoDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
          import_schema_uri=aiplatform.schema.dataset.ioformat.video
          .classification,
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
    with open("testdata/video_dataset_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)

  def test_time_series_dataset_component_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      dataset_create_op = TimeSeriesDatasetCreateOp(
          project=self._project,
          display_name=self._display_name,
          gcs_source=self._gcs_source,
      )

      dataset_export_op = TimeSeriesDatasetExportDataOp(
          project=self._project,
          dataset=dataset_create_op.outputs["dataset"],
          output_dir=self._gcs_output_dir,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
    with open("testdata/time_series_dataset_pipeline.json") as ef:
      expected_executor_output_json = json.load(ef, strict=False)
    # Ignore the kfp SDK version during comparison
    del executor_output_json["sdkVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
