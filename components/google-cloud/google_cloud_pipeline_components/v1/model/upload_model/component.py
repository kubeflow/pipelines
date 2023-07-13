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
from kfp import dsl
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
  """`Uploads <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/upload>`_ a Google Cloud Vertex `Model <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models>`_ and returns a Model artifact representing the uploaded Model
  resource.

  See `Model upload <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/upload>`_ method for more information.

  Args:
      project: Project to upload this Model to.
      location: Optional location to upload this Model to. If
        not set, defaults to ``us-central1``.
      display_name: The display name of the Model. The name
        can be up to 128 characters long and can be consist of any UTF-8
        characters. `More information. <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models#Model>`_
      description: The description of the Model. `More information. <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models#Model>`_
      parent_model: An artifact of a model which to upload a new version to. Only specify this field when uploading a new version. `More information. <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/upload#request-body>`_
      unmanaged_container_model: The unmanaged container model to be uploaded.  The Model can be passed from an upstream step or imported via a KFP ``dsl.importer``.

        :Examples:
          ::

            from kfp import dsl
            from google_cloud_pipeline_components.types import artifact_types

            importer_spec = dsl.importer(
              artifact_uri='gs://managed-pipeline-gcpc-e2e-test/automl-tabular/model',
              artifact_class=artifact_types.UnmanagedContainerModel,
              metadata={
                'containerSpec': { 'imageUri':
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:prod'
                  }
              })

      explanation_metadata: Metadata describing the Model's
        input and output for explanation. Both ``explanation_metadata`` and ``explanation_parameters`` must be passed together when used. `More information. <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata>`_
      explanation_parameters: Parameters to configure
        explaining for Model's predictions.  `More information. <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#ExplanationParameters>`_
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
      model: Artifact tracking the created Model.
      gcp_resources: Serialized JSON of ``gcp_resources`` `proto <https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto>`_ which tracks the upload Model's long-running operation.
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
              ', "pipeline_job": "',
              f'projects/{project}/locations/{location}/pipelineJobs/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}',
              '"',
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
              then=[
                  '--parent_model_name',
                  parent_model.metadata['resourceName'],
              ],
          ),
      ],
  )
