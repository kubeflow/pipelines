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
"""Google Cloud Pipeline Model components."""

import os

from .export_model import component as model_export_component
from .upload_model import component as model_upload_component

__all__ = [
    'ModelExportOp',
    'ModelUploadOp',
]

ModelExportOp = model_export_component.model_export
ModelUploadOp = model_upload_component.model_upload
