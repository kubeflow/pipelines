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
"""Create `Vertex AI AutoML training jobs <https://cloud.google.com/vertex-ai/docs/beginner/beginners-guide>`_ for image, text, video, and forecasting."""

from google_cloud_pipeline_components.v1.automl.training_job.automl_forecasting_training_job.component import automl_forecasting_training_job as AutoMLForecastingTrainingJobRunOp
from google_cloud_pipeline_components.v1.automl.training_job.automl_image_training_job.component import automl_image_training_job as AutoMLImageTrainingJobRunOp
from google_cloud_pipeline_components.v1.automl.training_job.automl_tabular_training_job.component import automl_tabular_training_job as AutoMLTabularTrainingJobRunOp
from google_cloud_pipeline_components.v1.automl.training_job.automl_text_training_job.component import automl_text_training_job as AutoMLTextTrainingJobRunOp
from google_cloud_pipeline_components.v1.automl.training_job.automl_video_training_job.component import automl_video_training_job as AutoMLVideoTrainingJobRunOp

__all__ = [
    'AutoMLImageTrainingJobRunOp',
    'AutoMLTextTrainingJobRunOp',
    'AutoMLTabularTrainingJobRunOp',
    'AutoMLForecastingTrainingJobRunOp',
    'AutoMLVideoTrainingJobRunOp',
]
