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
"""Manage models via `Vertex AI Model Registry <https://cloud.google.com/vertex-ai/docs/model-registry/introduction>`_."""

from google_cloud_pipeline_components.v1.model.delete_model.component import model_delete as ModelDeleteOp
from google_cloud_pipeline_components.v1.model.export_model.component import model_export as ModelExportOp
from google_cloud_pipeline_components.v1.model.upload_model.component import model_upload as ModelUploadOp

__all__ = [
    'ModelExportOp',
    'ModelUploadOp',
    'ModelDeleteOp',
]
