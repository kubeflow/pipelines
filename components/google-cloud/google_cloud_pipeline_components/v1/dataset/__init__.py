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

from kfp import components

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
    'GetVertexDatasetOp',
]

ImageDatasetCreateOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_image_dataset/component.yaml'
    )
)
ImageDatasetExportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_image_dataset/component.yaml'
    )
)
TabularDatasetCreateOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_tabular_dataset/component.yaml'
    )
)
TextDatasetCreateOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_text_dataset/component.yaml'
    )
)
TimeSeriesDatasetCreateOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_time_series_dataset/component.yaml'
    )
)
VideoDatasetCreateOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_video_dataset/component.yaml'
    )
)
TabularDatasetExportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_tabular_dataset/component.yaml'
    )
)
TextDatasetExportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_text_dataset/component.yaml'
    )
)
TimeSeriesDatasetExportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_time_series_dataset/component.yaml'
    )
)
VideoDatasetExportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_video_dataset/component.yaml'
    )
)
ImageDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_image_dataset/component.yaml'
    )
)

TextDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_text_dataset/component.yaml'
    )
)

VideoDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_video_dataset/component.yaml'
    )
)

GetVertexDatasetOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'get_vertex_dataset/component.yaml')
)
