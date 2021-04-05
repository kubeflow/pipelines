# Copyright 2021 Google LLC. All Rights Reserved.
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
"""Core modules for AI Platform Components."""

from google.cloud import aiplatform as aip
from google_cloud_components.aiplatform import utils

DatasetCreateOp = utils.convert_method_to_component(aip.Dataset.create)

DatasetExportDataOp = utils.convert_method_to_component(
    aip.Dataset.export_data, should_serialize_init=True)

CustomContainerTrainingJobRunOp = utils.convert_method_to_component(
    aip.CustomContainerTrainingJob.run, should_serialize_init=True)

ModelDeployOp = utils.convert_method_to_component(
    aip.Model.deploy, should_serialize_init=True)

ModelUploadOp = utils.convert_method_to_component(aip.Model.upload)
