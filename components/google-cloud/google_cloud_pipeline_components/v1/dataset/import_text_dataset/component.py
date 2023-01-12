# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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

from typing import Dict

from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Output


@container_component
def text_dataset_import(
    project: str,
    # TODO(b/243411151): misalignment of arguments in documentation vs function
    # signature.
    # TODO(b/243432307): Limitation on duplicate argument names.
    # TODO(b/244344908): Verify whether the input_dataset is needed before releasing v2.
    input_dataset: Input[VertexDataset],
    dataset: Output[VertexDataset],
    data_item_labels: Dict[str, str] = {},
    gcs_source: str = '',
    import_schema_uri: str = '',
    location: str = 'us-central1',
):
  """Upload data to existing managed dataset.

        Args:
            project (String): Required. project to retrieve dataset from.
            location (String): Optional location to retrieve dataset from.
            input_dataset (VertexDataset): Required. The dataset to be updated.
            gcs_source (Union[str, Sequence[str]]): Required. Google Cloud
              Storage URI(-s) to the input file(s). May contain wildcards. For
              more information on wildcards, see
                https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
                examples:
                    str: "gs://bucket/file.csv" Sequence[str]:
                      ["gs://bucket/file1.csv", "gs://bucket/file2.csv"]
            import_schema_uri (String): Required. Points to a YAML file stored
              on Google Cloud Storage describing the import format. Validation
              will be done against the schema. The schema is defined as an
              `OpenAPI 3.0.2 Schema Object <https://tinyurl.com/y538mdwt>`__.
            data_item_labels (Dict): Labels that will be applied to newly
              imported DataItems. If an identical DataItem as one being imported
              already exists in the Dataset, then these labels will be appended
              to these of the already existing one, and if labels with identical
              key is imported before, the old label value will be overwritten.
              If two DataItems are identical in the same import data operation,
              the labels will be combined and if key collision happens in this
              case, one of the values will be picked randomly. Two DataItems are
              considered identical if their content bytes are identical (e.g.
              image bytes or pdf bytes). These labels will be overridden by
              Annotation labels specified inside index file refenced by
              ``import_schema_uri``, e.g. jsonl file.

        Returns:
            dataset (VertexDataset):
                Instantiated representation of the managed dataset resource.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:latest',
      command=[
          'python3', '-m',
          'google_cloud_pipeline_components.container.aiplatform.remote_runner',
          '--cls_name', 'TextDataset', '--method_name', 'import_data'
      ],
      args=[
          '--init.project',
          project,
          '--init.location',
          location,
          '--init.dataset_name',
          "{{$.inputs.artifacts['input_dataset'].metadata['resourceName']}}",
          '--method.data_item_labels',
          data_item_labels,
          IfPresentPlaceholder(
              input_name='gcs_source',
              then=ConcatPlaceholder(['--method.gcs_source', gcs_source])),
          IfPresentPlaceholder(
              input_name='import_schema_uri',
              then=ConcatPlaceholder(
                  ['--method.import_schema_uri', import_schema_uri])),
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          dataset.uri,
      ])
