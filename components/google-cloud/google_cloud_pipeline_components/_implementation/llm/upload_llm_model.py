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
"""KFP Component for upload_llm_model."""

from google_cloud_pipeline_components import _image
from kfp import dsl


# pylint: disable=g-import-not-at-top, invalid-name,
# pylint: disable=g-doc-args
# pytype: disable=invalid-annotation
# pytype: disable=unsupported-operands
# pytype: disable=import-error
@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def refined_upload_llm_model(
    project: str,
    location: str,
    artifact_uri: str,
    model_reference_name: str,
    model_display_name: str,
    regional_endpoint: str,
    model_resource_name: dsl.OutputPath(str),
    gcp_resources: dsl.OutputPath(str),
    encryption_spec_key_name: str = '',
    upload_model: bool = True,
    tune_type: str = '',
):
  """Uploads LLM model.

  Args:
      project: Name of the GCP project.
      location: Location for model upload and deployment.
      artifact_uri: Path to the artifact to upload.
      model_reference_name: Large model reference name.
      model_display_name: Name of the model (shown in Model Registry).
      regional_endpoint: Regional API endpoint.
      encryption_spec_key_name: Customer-managed encryption key.
      upload_model: Whether to upload the model to the Model Registry. Default
        is ``True``. If ``False``, the model will not be uploaded and output
        artifacts will contain empty strings.
      tune_type: Method used to tune the model, e.g. ``rlhf``. If present, this
        value is used to set the ``tune-type`` run label during model upload.

  Returns:
      model_resource_name: Path to the created Model on Model Registry.
      gcp_resources: Serialized JSON of `gcp_resources`.
  """
  import json
  import logging
  import os
  import sys

  try:
    from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
  except ImportError:
    from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner

  try:
    os.makedirs(os.path.dirname(model_resource_name), exist_ok=True)

    if not upload_model:
      with open(model_resource_name, 'w') as fout:
        fout.write('')
      return

    pipeline_labels_str = os.getenv('VERTEX_AI_PIPELINES_RUN_LABELS')
    labels = json.loads(pipeline_labels_str) if pipeline_labels_str else {}
    labels['google-vertex-llm-tuning-base-model-id'] = (
        model_reference_name.replace('@', '-')
    )
    if tune_type:
      labels['tune-type'] = tune_type

    model_upload_payload = {
        'model': {
            'displayName': model_display_name,
            'largeModelReference': {'name': model_reference_name},
            'labels': labels,
            'generatedModelSource': {'genie_source': {'base_model_uri': ''}},
            'artifactUri': artifact_uri,
        }
    }
    if encryption_spec_key_name:
      model_upload_payload['model']['encryption_spec'] = {
          'kms_key_name': encryption_spec_key_name
      }

    regional_endpoint = regional_endpoint.rstrip('/')
    upload_model_uri = (
        f'{regional_endpoint}/projects/{project}/locations/{location}/models:'
        'upload'
    )

    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    upload_model_lro = remote_runner.create_lro(
        upload_model_uri,
        json.dumps(model_upload_payload),
        gcp_resources,
    )
    upload_model_lro = remote_runner.poll_lro(lro=upload_model_lro)
    model_resource = upload_model_lro['response']['model']
    model_version_id = upload_model_lro['response'].get(
        'model_version_id'
    ) or upload_model_lro['response'].get('modelVersionId')
    if model_version_id:
      model_resource += f'@{model_version_id}'

    with open(model_resource_name, 'w') as fout:
      fout.write(model_resource)

  except Exception as e:  # pylint: disable=broad-exception-caught
    if isinstance(e, ValueError):
      raise
    logging.exception(str(e))
    sys.exit(13)


# pytype: enable=import-error
# pytype: enable=unsupported-operands
# pytype: enable=invalid-annotation
# pylint: enable=g-doc-args
# pylint: enable=g-import-not-at-top, invalid-name, broad-exception-caught
