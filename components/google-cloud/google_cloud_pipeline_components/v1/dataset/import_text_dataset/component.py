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

from typing import Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output


@utils.gcpc_output_name_converter('dataset')
@dsl.container_component
def text_dataset_import(
    dataset: Input[VertexDataset],
    output__dataset: Output[VertexDataset],
    location: Optional[str] = 'us-central1',
    data_item_labels: Optional[dict] = {},
    gcs_source: Optional[str] = None,
    import_schema_uri: Optional[str] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Uploads data to an existing managed [Dataset](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.datasets).

  Args:
      location: Optional location to retrieve Datasetfrom.
      dataset: The Datasetto be updated.
      gcs_source: Google Cloud Storage URI(-s) to the input file(s). May contain wildcards. For more information on wildcards, see https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For example, "gs://bucket/file.csv" or ["gs://bucket/file1.csv", "gs://bucket/file2.csv"].
      import_schema_uri: Points to a YAML file stored on Google Cloud Storage describing the import format. Validation will be done against the schema. The schema is defined as an [OpenAPI 3.0.2 Schema Object](https://tinyurl.com/y538mdwt).
      data_item_labels: Labels that will be applied to newly imported DataItems. If an identical DataItem as one being imported already exists in the Dataset, then these labels will be appended to these of the already existing one, and if labels with identical key is imported before, the old label value will be overwritten. If two DataItems are identical in the same import data operation, the labels will be combined and if key collision happens in this case, one of the values will be picked randomly. Two DataItems are considered identical if their content bytes are identical (e.g. image bytes or pdf bytes). These labels will be overridden by Annotation labels specified inside index file refenced by `import_schema_uri`, e.g. jsonl file.
      project: Project to retrieve Dataset from. Defaults to the project in which the PipelineJob is run.

  Returns:
      dataset: Instantiated representation of the managed Datasetresource.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.v1.aiplatform.remote_runner',
          '--cls_name',
          'TextDataset',
          '--method_name',
          'import_data',
      ],
      args=[
          '--init.project',
          project,
          '--init.location',
          location,
          '--init.dataset_name',
          dataset.metadata['resourceName'],
          '--method.data_item_labels',
          data_item_labels,
          dsl.IfPresentPlaceholder(
              input_name='gcs_source', then=['--method.gcs_source', gcs_source]
          ),
          dsl.IfPresentPlaceholder(
              input_name='import_schema_uri',
              then=['--method.import_schema_uri', import_schema_uri],
          ),
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          output__dataset.uri,
      ],
  )
