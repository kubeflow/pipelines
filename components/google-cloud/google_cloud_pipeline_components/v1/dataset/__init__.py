# pytype: skip-file
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
    'TimeSeriesDatasetCreateOp',
    'TimeSeriesDatasetExportDataOp',
]

TimeSeriesDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_time_series_dataset/component.yaml'))

ImageDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_image_dataset/component.yaml'))

TabularDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_tabular_dataset/component.yaml'))

TextDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_text_dataset/component.yaml'))

VideoDatasetCreateOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'create_video_dataset/component.yaml'))

ImageDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_image_dataset/component.yaml'))

TabularDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_tabular_dataset/component.yaml'))

TimeSeriesDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_time_series_dataset/component.yaml'))

TextDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_text_dataset/component.yaml'))

VideoDatasetExportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'export_video_dataset/component.yaml'))

ImageDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_image_dataset/component.yaml'))

TextDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_text_dataset/component.yaml'))

VideoDatasetImportDataOp = load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_video_dataset/component.yaml'))
