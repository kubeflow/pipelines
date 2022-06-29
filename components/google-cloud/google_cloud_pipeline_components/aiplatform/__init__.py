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
"""Core modules for AI Platform Pipeline Components."""

import os
from google.cloud import aiplatform as aiplatform_sdk
from google_cloud_pipeline_components.aiplatform import utils
try:
  from kfp.v2.components import load_component_from_file
except ImportError:
  from kfp.components import load_component_from_file

__all__ = [
    'ImageDatasetCreateOp',
    'TabularDatasetCreateOp',
    'TextDatasetCreateOp',
    'VideoDatasetCreateOp',
    'ImageDatasetExportDataOp',
    'TabularDatasetExportDataOp',
    'TextDatasetExportDataOp',
    'VideoDatasetExportDataOp',
    'ImageDatasetImportDataOp',
    'TextDatasetImportDataOp',
    'VideoDatasetImportDataOp',
    'CustomContainerTrainingJobRunOp',
    'CustomPythonPackageTrainingJobRunOp',
    'AutoMLImageTrainingJobRunOp',
    'AutoMLTextTrainingJobRunOp',
    'AutoMLTabularTrainingJobRunOp',
    'AutoMLVideoTrainingJobRunOp',
    'ModelDeployOp',
    'ModelUndeployOp',
    'ModelBatchPredictOp',
    'ModelDeleteOp',
    'ModelExportOp',
    'ModelUploadOp',
    'EndpointCreateOp',
    'EndpointDeleteOp',
    'TimeSeriesDatasetCreateOp',
    'TimeSeriesDatasetExportDataOp',
    'AutoMLForecastingTrainingJobRunOp',
]

TimeSeriesDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/create_time_series_dataset/component.yaml'))

ImageDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/create_image_dataset/component.yaml'))

TabularDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/create_tabular_dataset/component.yaml'))

TextDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/create_text_dataset/component.yaml'))

VideoDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/create_video_dataset/component.yaml'))

ImageDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/export_image_dataset/component.yaml'))

TabularDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/export_tabular_dataset/component.yaml'))

TimeSeriesDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/export_time_series_dataset/component.yaml'))

TextDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/export_text_dataset/component.yaml'))

VideoDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/export_video_dataset/component.yaml'))

ImageDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/import_image_dataset/component.yaml'))

TextDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/import_text_dataset/component.yaml'))

VideoDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'dataset/import_video_dataset/component.yaml'))

CustomContainerTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomContainerTrainingJob,
    aiplatform_sdk.CustomContainerTrainingJob.run,
)

CustomPythonPackageTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomPythonPackageTrainingJob,
    aiplatform_sdk.CustomPythonPackageTrainingJob.run,
)

AutoMLImageTrainingJobRunOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_training_job/automl_image_training_job/component.yaml'))

AutoMLTextTrainingJobRunOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_training_job/automl_text_training_job/component.yaml'))

AutoMLTabularTrainingJobRunOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_training_job/automl_tabular_training_job/component.yaml'))

AutoMLForecastingTrainingJobRunOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_training_job/automl_forecasting_training_job/component.yaml'))

AutoMLVideoTrainingJobRunOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__),
        'automl_training_job/automl_video_training_job/component.yaml'))

ModelDeleteOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'model/delete_model/component.yaml'))

ModelExportOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'model/export_model/component.yaml'))

ModelDeployOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'endpoint/deploy_model/component.yaml'))

ModelUndeployOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'endpoint/undeploy_model/component.yaml'))

ModelBatchPredictOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'batch_predict_job/component.yaml'))

ModelUploadOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'model/upload_model/component.yaml'))

EndpointCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'endpoint/create_endpoint/component.yaml'))

EndpointDeleteOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'endpoint/delete_endpoint/component.yaml'))
