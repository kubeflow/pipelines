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

from google.cloud import aiplatform as aiplatform_sdk
from google_cloud_pipeline_components.aiplatform import utils

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
    'ModelBatchPredictOp',
    'ModelUploadOp',
    'EndpointCreateOp',
    'TimeSeriesDatasetCreateOp',
    'TimeSeriesDatasetExportDataOp',
    'AutoMLForecastingTrainingJobRunOp',
]

TimeSeriesDatasetCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.TimeSeriesDataset, aiplatform_sdk.TimeSeriesDataset.create
)

ImageDatasetCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.ImageDataset, aiplatform_sdk.ImageDataset.create
)

TabularDatasetCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.TabularDataset, aiplatform_sdk.TabularDataset.create
)

TextDatasetCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.TextDataset, aiplatform_sdk.TextDataset.create
)

VideoDatasetCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.VideoDataset, aiplatform_sdk.VideoDataset.create
)

ImageDatasetExportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.ImageDataset,
    aiplatform_sdk.ImageDataset.export_data,
)

TabularDatasetExportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.TabularDataset,
    aiplatform_sdk.TabularDataset.export_data,
)

TimeSeriesDatasetExportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.TimeSeriesDataset,
    aiplatform_sdk.TimeSeriesDataset.export_data,
)

TextDatasetExportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.TextDataset,
    aiplatform_sdk.TextDataset.export_data,
)

VideoDatasetExportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.VideoDataset,
    aiplatform_sdk.VideoDataset.export_data,
)

ImageDatasetImportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.ImageDataset,
    aiplatform_sdk.ImageDataset.import_data,
)

TextDatasetImportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.TextDataset,
    aiplatform_sdk.TextDataset.import_data,
)

VideoDatasetImportDataOp = utils.convert_method_to_component(
    aiplatform_sdk.VideoDataset,
    aiplatform_sdk.VideoDataset.import_data,
)

CustomContainerTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomContainerTrainingJob,
    aiplatform_sdk.CustomContainerTrainingJob.run,
)

CustomPythonPackageTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomPythonPackageTrainingJob,
    aiplatform_sdk.CustomPythonPackageTrainingJob.run,
)

AutoMLImageTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.AutoMLImageTrainingJob,
    aiplatform_sdk.AutoMLImageTrainingJob.run,
)

AutoMLTextTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.AutoMLTextTrainingJob,
    aiplatform_sdk.AutoMLTextTrainingJob.run,
)

AutoMLTabularTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.AutoMLTabularTrainingJob,
    aiplatform_sdk.AutoMLTabularTrainingJob.run,
)

AutoMLForecastingTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.AutoMLForecastingTrainingJob,
    aiplatform_sdk.AutoMLForecastingTrainingJob.run,
)

AutoMLVideoTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.AutoMLVideoTrainingJob,
    aiplatform_sdk.AutoMLVideoTrainingJob.run,
)

ModelDeployOp = utils.convert_method_to_component(
    aiplatform_sdk.Model,
    aiplatform_sdk.Model.deploy,
)

ModelBatchPredictOp = utils.convert_method_to_component(
    aiplatform_sdk.Model,
    aiplatform_sdk.Model.batch_predict,
)

ModelUploadOp = utils.convert_method_to_component(
    aiplatform_sdk.Model, aiplatform_sdk.Model.upload
)

ModelExportOp = utils.convert_method_to_component(
    aiplatform_sdk.Model, aiplatform_sdk.Model.export_model
)

EndpointCreateOp = utils.convert_method_to_component(
    aiplatform_sdk.Endpoint, aiplatform_sdk.Endpoint.create
)
