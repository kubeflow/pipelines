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

from typing import Dict

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def model_upload(
    project: str,
    display_name: str,
    gcp_resources: OutputPath(str),
    model: Output[VertexModel],
    location: str = 'us-central1',
    description: str = '',
    parent_model: Input[VertexModel] = None,
    unmanaged_container_model: Input[UnmanagedContainerModel] = None,
    explanation_metadata: Dict[str, str] = {},
    explanation_parameters: Dict[str, str] = {},
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
):
  # fmt: off
  """Uploads a model and returns a Model representing the uploaded Model
  resource.

  For more details, see
  https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/upload.

  Args:
      project: Project to upload this model to.
      location: Optional location to upload this model to. If
        not set, default to us-central1.
      display_name: The display name of the Model. The name
        can be up to 128 characters long and can be consist of any UTF-8
        characters.
      description: The description of the model.
      parent_model: An artifact of a model
        which to upload a new version to. Only specify this field when
        uploading a new version.
      unmanaged_container_model: The unmanaged container model to be uploaded.  The model can
        be passed from an upstream step, or imported via an importer.

        Examples::

          from kfp.dsl import importer
          from
          google_cloud_pipeline_components.google_cloud_pipeline_components.types
          import artifact_types

          importer_spec = importer(
            artifact_uri='gs://managed-pipeline-gcpc-e2e-test/automl-tabular/model',
            artifact_class=artifact_types.UnmanagedContainerModel, metadata={
              'containerSpec': { 'imageUri':
                'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:prod'
                }
            })

      explanation_metadata: Metadata describing the Model's
        input and output for explanation. Both `explanation_metadata` and
        `explanation_parameters` must be passed together when used.  For more
        details, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.
      explanation_parameters: Parameters to configure
        explaining for Model's predictions.  For more details, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.
      encryption_spec_key_name: Customer-managed encryption
        key spec for a Model. If set, this Model and all sub-resources of this
        Model will be secured by this key.  Has the form:
        ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
        The key needs to be in the same region as where the compute resource
        is created.
      labels: The labels with user-defined metadata to
        organize your model.  Label keys and values can be no longer than 64
        characters (Unicode codepoints), can only contain lowercase letters,
        numeric characters, underscores and dashes. International characters
        are allowed.  See https://goo.gl/xmQnxf for more information and
        examples of labels.

  Returns:
      model: Artifact tracking the created model.
      gcp_resources: Serialized gcp_resources proto tracking the upload model's long
          running operation. For more details, see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.model.upload_model.launcher',
      ],
      args=[
          '--type',
          'UploadModel',
          '--payload',
          ConcatPlaceholder([
              '{',
              '"display_name": "',
              display_name,
              '"',
              ', "description": "',
              description,
              '"',
              ', "explanation_spec": {',
              '"parameters": ',
              explanation_parameters,
              ', "metadata": ',
              explanation_metadata,
              '}',
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name,
              '"}',
              ', "labels": ',
              labels,
              '}',
          ]),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
          IfPresentPlaceholder(
              input_name='parent_model',
              then=ConcatPlaceholder([
                  '--parent_model_name ',
                  parent_model.metadata['resourceName'],
              ]),
          ),
      ],
  )
