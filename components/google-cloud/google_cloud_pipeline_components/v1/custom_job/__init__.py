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
# fmt: off
"""Run KFP components as [Vertex AI Custom Training Jobs](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) with customized worker and cloud configurations."""
# fmt: on

from google_cloud_pipeline_components.v1.custom_job.component import custom_training_job as CustomTrainingJobOp
from google_cloud_pipeline_components.v1.custom_job.utils import create_custom_training_job_from_component
from google_cloud_pipeline_components.v1.custom_job.utils import create_custom_training_job_op_from_component

__all__ = [
    'CustomTrainingJobOp',
    'create_custom_training_job_op_from_component',
    'create_custom_training_job_from_component',
]
