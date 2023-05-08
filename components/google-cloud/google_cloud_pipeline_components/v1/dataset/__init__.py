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

from google_cloud_pipeline_components.v1.dataset.create_image_dataset.component import image_dataset_create as ImageDatasetCreateOp
from google_cloud_pipeline_components.v1.dataset.create_tabular_dataset.component import tabular_dataset_create as TabularDatasetCreateOp
from google_cloud_pipeline_components.v1.dataset.create_text_dataset.component import text_dataset_create as TextDatasetCreateOp
from google_cloud_pipeline_components.v1.dataset.create_time_series_dataset.component import time_series_dataset_create as TimeSeriesDatasetCreateOp
from google_cloud_pipeline_components.v1.dataset.create_video_dataset.component import video_dataset_create as VideoDatasetCreateOp
from google_cloud_pipeline_components.v1.dataset.export_image_dataset.component import image_dataset_export as ImageDatasetExportDataOp
from google_cloud_pipeline_components.v1.dataset.export_tabular_dataset.component import tabular_dataset_export as TabularDatasetExportDataOp
from google_cloud_pipeline_components.v1.dataset.export_text_dataset.component import text_dataset_export as TextDatasetExportDataOp
from google_cloud_pipeline_components.v1.dataset.export_time_series_dataset.component import time_series_dataset_export as TimeSeriesDatasetExportDataOp
from google_cloud_pipeline_components.v1.dataset.export_video_dataset.component import video_dataset_export as VideoDatasetExportDataOp
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

# TODO(b/279604238): convert components below to Python container components
ImageDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_image_dataset/component.yaml'
    )
)
"""
Upload data to existing managed dataset.
Args:
    project (String):
        Required. project to retrieve dataset from.
    location (String):
        Optional location to retrieve dataset from.
    dataset (Dataset):
        Required. The dataset to be updated.
    gcs_source (Union[str, Sequence[str]]):
        Required. Google Cloud Storage URI(-s) to the
        input file(s). May contain wildcards. For more
        information on wildcards, see
        https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
        examples:
            str: "gs://bucket/file.csv"
            Sequence[str]: ["gs://bucket/file1.csv", "gs://bucket/file2.csv"]
    import_schema_uri (String):
        Required. Points to a YAML file stored on Google Cloud
        Storage describing the import format. Validation will be
        done against the schema. The schema is defined as an
        `OpenAPI 3.0.2 Schema
        Object <https://tinyurl.com/y538mdwt>`__.
    data_item_labels (JsonObject):
        Labels that will be applied to newly imported DataItems. If
        an identical DataItem as one being imported already exists
        in the Dataset, then these labels will be appended to these
        of the already existing one, and if labels with identical
        key is imported before, the old label value will be
        overwritten. If two DataItems are identical in the same
        import data operation, the labels will be combined and if
        key collision happens in this case, one of the values will
        be picked randomly. Two DataItems are considered identical
        if their content bytes are identical (e.g. image bytes or
        pdf bytes). These labels will be overridden by Annotation
        labels specified inside index file refenced by
        ``import_schema_uri``,
        e.g. jsonl file.
Returns:
    dataset (Dataset):
        Instantiated representation of the managed dataset resource.
"""

TextDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_text_dataset/component.yaml'
    )
)
"""
Upload data to existing managed dataset.
Args:
    project (String):
        Required. project to retrieve dataset from.
    location (String):
        Optional location to retrieve dataset from.
    dataset (Dataset):
        Required. The dataset to be updated.
    gcs_source (Union[str, Sequence[str]]):
        Required. Google Cloud Storage URI(-s) to the
        input file(s). May contain wildcards. For more
        information on wildcards, see
        https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
        examples:
            str: "gs://bucket/file.csv"
            Sequence[str]: ["gs://bucket/file1.csv", "gs://bucket/file2.csv"]
    import_schema_uri (String):
        Required. Points to a YAML file stored on Google Cloud
        Storage describing the import format. Validation will be
        done against the schema. The schema is defined as an
        `OpenAPI 3.0.2 Schema
        Object <https://tinyurl.com/y538mdwt>`__.
    data_item_labels (JsonObject):
        Labels that will be applied to newly imported DataItems. If
        an identical DataItem as one being imported already exists
        in the Dataset, then these labels will be appended to these
        of the already existing one, and if labels with identical
        key is imported before, the old label value will be
        overwritten. If two DataItems are identical in the same
        import data operation, the labels will be combined and if
        key collision happens in this case, one of the values will
        be picked randomly. Two DataItems are considered identical
        if their content bytes are identical (e.g. image bytes or
        pdf bytes). These labels will be overridden by Annotation
        labels specified inside index file refenced by
        ``import_schema_uri``,
        e.g. jsonl file.
Returns:
    dataset (Dataset):
        Instantiated representation of the managed dataset resource.
"""

VideoDatasetImportDataOp = components.load_component_from_file(
    os.path.join(
        os.path.dirname(__file__), 'import_video_dataset/component.yaml'
    )
)
"""
Upload data to existing managed dataset.
Args:
    project (String):
        Required. project to retrieve dataset from.
    location (String):
        Optional location to retrieve dataset from.
    dataset (Dataset):
        Required. The dataset to be updated.
    gcs_source (Union[str, Sequence[str]]):
        Required. Google Cloud Storage URI(-s) to the
        input file(s). May contain wildcards. For more
        information on wildcards, see
        https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
        examples:
            str: "gs://bucket/file.csv"
            Sequence[str]: ["gs://bucket/file1.csv", "gs://bucket/file2.csv"]
    import_schema_uri (String):
        Required. Points to a YAML file stored on Google Cloud
        Storage describing the import format. Validation will be
        done against the schema. The schema is defined as an
        `OpenAPI 3.0.2 Schema
        Object <https://tinyurl.com/y538mdwt>`__.
    data_item_labels (JsonObject):
        Labels that will be applied to newly imported DataItems. If
        an identical DataItem as one being imported already exists
        in the Dataset, then these labels will be appended to these
        of the already existing one, and if labels with identical
        key is imported before, the old label value will be
        overwritten. If two DataItems are identical in the same
        import data operation, the labels will be combined and if
        key collision happens in this case, one of the values will
        be picked randomly. Two DataItems are considered identical
        if their content bytes are identical (e.g. image bytes or
        pdf bytes). These labels will be overridden by Annotation
        labels specified inside index file refenced by
        ``import_schema_uri``,
        e.g. jsonl file.
Returns:
    dataset (Dataset):
        Instantiated representation of the managed dataset resource.
"""

GetVertexDatasetOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'get_vertex_dataset/component.yaml')
)
"""
  Resolves a dataset artifact to an existing Vertex Dataset.
  Args:
    dataset_resource_name (str):
      Required. Vertex Dataset resource name in the format of projects/{project}/locations/{location}/datasets/{dataset}
  Returns:
    dataset (Optional[google.VertexDataset]):
      Vertex Dataset artifact with a resourceName attribute in the format of projects/{project}/locations/{location}/datasets/{dataset}
"""
