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

from google_cloud_pipeline_components.aiplatform import utils
from kfp.components import load_component_from_file

from google.cloud import aiplatform as aiplatform_sdk

__all__ = [
    'CustomContainerTrainingJobRunOp',
    'CustomPythonPackageTrainingJobRunOp',
]


CustomContainerTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomContainerTrainingJob,
    aiplatform_sdk.CustomContainerTrainingJob.run,
)

CustomPythonPackageTrainingJobRunOp = utils.convert_method_to_component(
    aiplatform_sdk.CustomPythonPackageTrainingJob,
    aiplatform_sdk.CustomPythonPackageTrainingJob.run,
)
