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
"""Google Cloud Pipeline Dataset components."""

import os
from kfp.components import load_component_from_file

from .create_image_dataset import component as create_image_dataset_component
from .create_tabular_dataset import component as create_tabular_dataset_component
from .create_text_dataset import component as create_text_dataset_component
from .create_time_series_dataset import component as create_time_series_dataset_component
from .create_video_dataset import component as create_video_dataset_component
from .export_image_dataset import component as export_image_dataset_component
from .export_tabular_dataset import component as export_tabular_dataset_component
from .export_text_dataset import component as export_text_dataset_component
from .export_time_series_dataset import component as export_time_series_dataset_component
from .export_video_dataset import component as export_video_dataset_component

__all__ = [
    'ImageDatasetCreateOp',
    'TabularDatasetCreateOp',
    'TextDatasetCreateOp',
    'TimeSeriesDatasetCreateOp',
    'VideoDatasetCreateOp',
    'ImageDatasetExportDataOp',
    'TabularDatasetExportDataOp',
    'TextDatasetExportDataOp',
    'TimeSeriesDatasetExportDataOp',
    'VideoDatasetExportDataOp',
    'ImageDatasetImportDataOp',
    'TextDatasetImportDataOp',
    'VideoDatasetImportDataOp',
]

ImageDatasetCreateOp = create_image_dataset_component.image_dataset_create
TabularDatasetCreateOp = create_tabular_dataset_component.tabular_dataset_create
TextDatasetCreateOp = create_text_dataset_component.text_dataset_create
TimeSeriesDatasetCreateOp = (
    create_time_series_dataset_component.time_series_dataset_create
)
VideoDatasetCreateOp = create_video_dataset_component.video_dataset_create
ImageDatasetExportDataOp = export_image_dataset_component.image_dataset_export
TabularDatasetExportDataOp = (
    export_tabular_dataset_component.tabular_dataset_export
)
TextDatasetExportDataOp = export_text_dataset_component.text_dataset_export
TimeSeriesDatasetExportDataOp = (
    export_time_series_dataset_component.time_series_dataset_export
)
VideoDatasetExportDataOp = export_video_dataset_component.video_dataset_export

ImageDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_image_dataset/component.yaml'
    )
)

TextDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_text_dataset/component.yaml'
    )
)

VideoDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_video_dataset/component.yaml'
    )
)
