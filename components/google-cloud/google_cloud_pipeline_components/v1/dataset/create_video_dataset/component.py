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
from kfp.dsl import Output


@container_component
def video_dataset_create(
    project: str,
    # TODO(b/243411151): misalignment of arguments in documentation vs function
    # signature.
    display_name: str,
    dataset: Output[VertexDataset],
    location: str = 'us-central1',
    data_item_labels: Dict[str, str] = {},
    gcs_source: str = '',
    import_schema_uri: str = '',
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
):
  """Creates a new video dataset and optionally imports data into dataset

        when source and import_schema_uri are passed.
        Args:
            display_name (String): Required. The user-defined name of the
              Dataset. The name can be up to 128 characters long and can be
              consist of any UTF-8 characters.
            gcs_source (Union[str, Sequence[str]]): Google Cloud Storage URI(-s)
              to the input file(s). May contain wildcards. For more information
              on wildcards, see
                https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
                examples:
                    str: "gs://bucket/file.csv" Sequence[str]:
                      ["gs://bucket/file1.csv", "gs://bucket/file2.csv"]
            import_schema_uri (String): Points to a YAML file stored on Google
              Cloud Storage describing the import format. Validation will be
              done against the schema. The schema is defined as an `OpenAPI
              3.0.2 Schema Object <https://tinyurl.com/y538mdwt>`__.
            data_item_labels (Dict): Optional. Labels that will be applied to
              newly imported DataItems. If an identical DataItem as one being
              imported already exists in the Dataset, then these labels will be
              appended to these of the already existing one, and if labels with
              identical key is imported before, the old label value will be
              overwritten. If two DataItems are identical in the same import
              data operation, the labels will be combined and if key collision
              happens in this case, one of the values will be picked randomly.
              Two DataItems are considered identical if their content bytes are
              identical (e.g. image bytes or pdf bytes). These labels will be
              overridden by Annotation labels specified inside index file
              refenced by ``import_schema_uri``,
            project (String): Required. project to retrieve dataset from.
            location (String): Optional location to retrieve dataset from.
            labels (Dict): Optional. Labels with user-defined metadata to
              organize your Tensorboards. Label keys and values can be no longer
              than 64 characters (Unicode codepoints), can only contain
              lowercase letters, numeric characters, underscores and dashes.
              International characters are allowed. No more than 64 user labels
              can be associated with one Tensorboard (System labels are
              excluded). See https://goo.gl/xmQnxf for more information and
              examples of labels. System reserved label keys are prefixed with
              "aiplatform.googleapis.com/" and are immutable.
            encryption_spec_key_name (Optional[String]): Optional. The Cloud KMS
              resource identifier of the customer managed encryption key used to
              protect the dataset. Has the
                form:
                  ``projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key``.
                  The key needs to be in the same region as where the compute
                  resource is created. If set, this Dataset and all
                  sub-resources of this Dataset will be secured by this key.
                  Overrides encryption_spec_key_name set in aiplatform.init.

        Returns:
            video_dataset (google.VertexDataset):
                Instantiated representation of the managed video dataset
                resource.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-m',
          'google_cloud_pipeline_components.container.aiplatform.remote_runner',
          '--cls_name', 'VideoDataset', '--method_name', 'create'
      ],
      args=[
          '--method.project',
          project,
          '--method.location',
          location,
          '--method.display_name',
          display_name,
          '--method.data_item_labels',
          data_item_labels,
          IfPresentPlaceholder(
              input_name='gcs_source',
              then=ConcatPlaceholder(['--method.gcs_source', gcs_source])),
          IfPresentPlaceholder(
              input_name='import_schema_uri',
              then=ConcatPlaceholder(
                  ['--method.import_schema_uri', import_schema_uri])),
          '--method.labels',
          labels,
          IfPresentPlaceholder(
              input_name='encryption_spec_key_name',
              then=ConcatPlaceholder([
                  '--method.encryption_spec_key_name', encryption_spec_key_name
              ])),
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          dataset.uri,
      ])
