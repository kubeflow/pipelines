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
"""Test google-cloud-pipeline-componets to ensure the compile without error."""

import unittest
import kfp
from kfp.v2 import compiler
from google.cloud import aiplatform
from google_cloud_pipeline_components.aiplatform import (
    ImageDatasetCreateOp,
    AutoMLImageTrainingJobRunOp,
    ModelDeployOp,
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
    ModelUploadOp,
)


class ComponetsCompileTest(unittest.TestCase):

    def setUp(self):
        super(ComponetsCompileTest, self).setUp()
        self._project = "test_project"
        self._display_name = "test_display_name"
        self._model_display_name = "test_model_display_name"
        self._gcs_source = "gs://test_gcs_source"
        self._gcs_output_dir = "gs://test_gcs_output_dir"
        self._pipeline_root = "gs://test_pipeline_root"
        self._gcs_destination_prefix = "gs://test_gcs_output_dir/batch_prediction"
        self._serving_container_image_uri = "gcr.io/test_project/test_image:test_tag"
        self._artifact_uri = "project/test_artifact_uri"

    def test_image_data_pipeline_component_ops_compile(self):

        @kfp.dsl.pipeline(name="training-test")
        def pipeline():
            dataset_create_op = ImageDatasetCreateOp(
                project=self._project,
                display_name=self._display_name,
                gcs_source=self._gcs_source,
                import_schema_uri=aiplatform.schema.dataset.ioformat.image.
                single_label_classification,
            )

            training_job_run_op = AutoMLImageTrainingJobRunOp(
                project=self._project,
                display_name=self._display_name,
                prediction_type="classification",
                model_type="CLOUD",
                base_model=None,
                dataset=dataset_create_op.outputs["dataset"],
                model_display_name=self._model_display_name,
                training_fraction_split=0.6,
                validation_fraction_split=0.2,
                test_fraction_split=0.2,
                budget_milli_node_hours=8000,
            )

            model_deploy_op = ModelDeployOp(
                project=self._project,
                model=training_job_run_op.outputs["model"]
            )

            batch_predict_op = ModelBatchPredictOp(
                project=self._project,
                model=training_job_run_op.outputs["model"],
                job_display_name=self._display_name,
                gcs_source=self._gcs_source,
                gcs_destination_prefix=self._gcs_destination_prefix,
            )

            dataset_export_op = ImageDatasetExportDataOp(
                project=self._project,
                dataset=dataset_create_op.outputs["dataset"],
                output_dir=self._gcs_output_dir,
            )

            dataset_import_op = ImageDatasetImportDataOp(
                gcs_source=self._gcs_source,
                dataset=dataset_create_op.outputs["dataset"],
                import_schema_uri=aiplatform.schema.dataset.ioformat.image.
                single_label_classification
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline,
            pipeline_root=self._pipeline_root,
            output_path="pipeline.json"
        )

    def test_tabular_data_pipeline_component_ops_compile(self):

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
                optimization_prediction_type='regression',
                optimization_objective='minimize-rmse',
                column_transformations=[
                    {
                        "numeric": {
                            "column_name": "longitude"
                        }
                    },
                ],
                target_column="longitude",
                dataset=dataset_create_op.outputs["dataset"],
            )

            model_deploy_op = ModelDeployOp(
                project=self._project,
                model=training_job_run_op.outputs["model"]
            )

            batch_predict_op = ModelBatchPredictOp(
                project=self._project,
                model=training_job_run_op.outputs["model"],
                job_display_name=self._display_name,
                gcs_source=self._gcs_source,
                gcs_destination_prefix=self._gcs_destination_prefix,
            )

            dataset_export_op = TabularDatasetExportDataOp(
                project=self._project,
                dataset=dataset_create_op.outputs["dataset"],
                output_dir=self._gcs_output_dir,
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline,
            pipeline_root=self._pipeline_root,
            output_path="pipeline.json"
        )

    def test_text_data_pipeline_component_ops_compile(self):

        @kfp.dsl.pipeline(name="training-test")
        def pipeline():
            dataset_create_op = TextDatasetCreateOp(
                project=self._project,
                display_name=self._display_name,
                gcs_source=self._gcs_source,
                import_schema_uri=aiplatform.schema.dataset.ioformat.text.
                multi_label_classification,
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

            model_deploy_op = ModelDeployOp(
                project=self._project,
                model=training_job_run_op.outputs["model"]
            )

            batch_predict_op = ModelBatchPredictOp(
                project=self._project,
                model=training_job_run_op.outputs["model"],
                job_display_name=self._display_name,
                gcs_source=self._gcs_source,
                gcs_destination_prefix=self._gcs_destination_prefix,
            )

            dataset_export_op = TextDatasetExportDataOp(
                project=self._project,
                dataset=dataset_create_op.outputs["dataset"],
                output_dir=self._gcs_output_dir,
            )

            dataset_import_op = TextDatasetImportDataOp(
                gcs_source=self._gcs_source,
                dataset=dataset_create_op.outputs["dataset"],
                import_schema_uri=aiplatform.schema.dataset.ioformat.text.
                multi_label_classification
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline,
            pipeline_root=self._pipeline_root,
            output_path="pipeline.json"
        )

    def test_video_data_pipeline_component_ops_compile(self):

        @kfp.dsl.pipeline(name="training-test")
        def pipeline():
            dataset_create_op = VideoDatasetCreateOp(
                project=self._project,
                display_name=self._display_name,
                gcs_source=self._gcs_source,
                import_schema_uri=aiplatform.schema.dataset.ioformat.video.
                classification,
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

            model_deploy_op = ModelDeployOp(
                project=self._project,
                model=training_job_run_op.outputs["model"]
            )

            batch_predict_op = ModelBatchPredictOp(
                project=self._project,
                model=training_job_run_op.outputs["model"],
                job_display_name=self._display_name,
                gcs_source=self._gcs_source,
                gcs_destination_prefix=self._gcs_destination_prefix,
            )

            dataset_export_op = VideoDatasetExportDataOp(
                project=self._project,
                dataset=dataset_create_op.outputs["dataset"],
                output_dir=self._gcs_output_dir,
            )

            dataset_import_op = VideoDatasetImportDataOp(
                gcs_source=self._gcs_source,
                dataset=dataset_create_op.outputs["dataset"],
                import_schema_uri=aiplatform.schema.dataset.ioformat.video.
                classification
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline,
            pipeline_root=self._pipeline_root,
            output_path="pipeline.json"
        )

    def test_model_pipeline_component_ops_compile(self):

        @kfp.dsl.pipeline(name="training-test")
        def pipeline():

            model_upload_op = ModelUploadOp(
                project=self._project,
                display_name=self._display_name,
                serving_container_image_uri=self._serving_container_image_uri,
                artifact_uri=self._artifact_uri
            )

            endpoint_create_op = EndpointCreateOp(
                project=self._project,
                display_name=self._display_name
            )

            model_deploy_op = ModelDeployOp(
                project=self._project,
                model=model_upload_op.outputs["model"]
            )

            batch_predict_op = ModelBatchPredictOp(
                project=self._project,
                model=model_upload_op.outputs["model"],
                job_display_name=self._display_name,
                gcs_source=self._gcs_source,
                gcs_destination_prefix=self._gcs_destination_prefix
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline,
            pipeline_root=self._pipeline_root,
            output_path="pipeline.json"
        )
