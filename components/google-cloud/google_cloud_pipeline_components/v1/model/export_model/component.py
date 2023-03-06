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

from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import OutputPath


@container_component
def model_export(
    model: Input[VertexModel],
    export_format_id: str,
    # TODO(b/243411151): misalignment of arguments in documentation vs function
    # signature.
    output_info: OutputPath(Dict[str, str]),
    gcp_resources: OutputPath(str),
    artifact_destination: str = '',
    image_destination: str = '',
):
  """Exports a trained, exportable, Model to a location specified by the user.

    A Model is considered to be exportable if it has at least one supported
    export format.
    For more details, see
    https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/export.

    Args:
        model (google.VertexModel): Required. The model to be exported.
        export_format_id (str): The ID of the format in which the Model must be
          exported. Each Model lists the export formats it supports. If no value
          is provided here, then the first from the list of the Model's
          supported formats is used by default.
        artifact_destination (Optional[str]): The Cloud Storage location where
          the Model artifact is to be written to. Under the directory given as
          the destination a new one with name
          "``model-export-<model-display-name>-<timestamp-of-export-call>``",
          where timestamp is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format, will
          be created. Inside, the Model and any of its supporting files will be
          written.  This field should only be set when, in
          [Model.supported_export_formats], the value for the key given in
          `export_format_id` contains ``ARTIFACT``.
        image_destination (Optional[str]): The Google Container Registry or
          Artifact Registry URI where the Model container image will be copied
          to. Accepted forms:  -  Google Container Registry path. For example:
          ``gcr.io/projectId/imageName:tag``.  -  Artifact Registry path. For
          example:
            ``us-central1-docker.pkg.dev/projectId/repoName/imageName:tag``.
            This field should only be set when, in
            [Model.supported_export_formats], the value for the key given in
            `export_format_id` contains ``IMAGE``.

    Returns:
        output_info (str):
            Details of the completed export with output destination paths to
            the artifacts or container image.
        gcp_resources (str):
            Serialized gcp_resources proto tracking the export model's long
            running operation.

            For more details, see
            https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-u', '-m',
          'google_cloud_pipeline_components.container.v1.model.export_model.launcher'
      ],
      args=[
          '--type',
          'ExportModel',
          '--payload',
          ConcatPlaceholder([
              '{',
              '"name": "',
              "{{$.inputs.artifacts['model'].metadata['resourceName']}}",
              '"',
              ', "output_config": {',
              '"export_format_id": "',
              export_format_id,
              '"',
              ', "artifact_destination": {',
              '"output_uri_prefix": "',
              artifact_destination,
              '"',
              '}',
              ', "image_destination":  {',
              '"output_uri": "',
              image_destination,
              '"',
              '}',
              '}',
              '}',
          ]),
          '--project',
          '',  # not being used
          '--location',
          '',  # not being used
          '--gcp_resources',
          gcp_resources,
          '--output_info',
          output_info,
      ])
