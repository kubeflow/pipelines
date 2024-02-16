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
"""KFP Component for deploy_llm_model."""

from google_cloud_pipeline_components import _image
from kfp import dsl


# pylint: disable=g-import-not-at-top, invalid-name
# pylint: disable=g-doc-args
# pytype: disable=invalid-annotation
# pytype: disable=import-error
@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def deploy_llm_model(
    project: str,
    location: str,
    model_resource_name: str,
    display_name: str,
    regional_endpoint: str,
    endpoint_resource_name: dsl.OutputPath(str),
    create_endpoint_gcp_resources: dsl.OutputPath(str),
    deploy_model_gcp_resources: dsl.OutputPath(str),
    encryption_spec_key_name: str = '',
    service_account: str = '',
    deploy_model: bool = True,
):
  """Creates a vertex endpoint and deploy the specified model.

  Args:
      project: Name of the GCP project.
      location: Location for model upload and deployment.
      model_resource_name: Path to the created Model on Model Registry.
      display_name: Name of the model (shown in Model Registry).
      regional_endpoint: Regional API endpoint.
      encryption_spec_key_name: Customer-managed encryption key.
      service_account: If set, then a custom service account will be used.
      deploy_model: Whether to deploy the model to an endpoint. Default is
        ``True``. If ``False``, the model will not be deployed and output
        artifacts will contain empty strings.

  Returns:
      endpoint_resource_name: Path to the created endpoint on Online Prediction.
      create_endpoint_gcp_resources: Serialized JSON of GCP resources for
          creating an endpoint.
      deploy_model_gcp_resources: Serialized JSON of GCP resources for deploying
          the model.
  """
  import json
  import logging
  import os
  import sys
  from typing import Any, Dict

  try:
    from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner
  except ImportError:
    from google_cloud_pipeline_components.container.v1.gcp_launcher import lro_remote_runner

  def run_lro_remote_runner(
      url: str, payload: Dict[str, Any], gcp_resources: str
  ) -> Any:
    remote_runner = lro_remote_runner.LroRemoteRunner(location)
    lro = remote_runner.create_lro(url, json.dumps(payload), gcp_resources)
    return remote_runner.poll_lro(lro=lro)

  try:
    os.makedirs(os.path.dirname(endpoint_resource_name), exist_ok=True)

    if not deploy_model:
      with open(endpoint_resource_name, 'w') as fout:
        fout.write('')
      return

    regional_endpoint = regional_endpoint.rstrip('/')

    create_endpoint_payload = {
        'displayName': display_name,
    }

    pipeline_labels_str = os.getenv('VERTEX_AI_PIPELINES_RUN_LABELS')
    if pipeline_labels_str:
      create_endpoint_payload['labels'] = json.loads(pipeline_labels_str)

    if encryption_spec_key_name:
      create_endpoint_payload['encryption_spec'] = {
          'kms_key_name': encryption_spec_key_name
      }

    create_endpoint_lro = run_lro_remote_runner(
        url=(
            f'{regional_endpoint}/projects/{project}/locations/{location}'
            '/endpoints'
        ),
        payload=create_endpoint_payload,
        gcp_resources=create_endpoint_gcp_resources,
    )

    response_endpoint = create_endpoint_lro['response']['name']
    with open(endpoint_resource_name, 'w') as fout:
      fout.write(response_endpoint)

    logging.info(
        'Endpoint created successfully. Deploying model %s to endpoint',
        model_resource_name,
    )

    deploy_model_payload = {
        'deployedModel': {
            'model': model_resource_name,
            'displayName': display_name,
            'automaticResources': {'minReplicaCount': 1, 'maxReplicaCount': 1},
        }
    }
    if service_account:
      deploy_model_payload['deployedModel']['service_account'] = service_account

    _ = run_lro_remote_runner(
        url=f'{regional_endpoint}/{response_endpoint}:deployModel',
        payload=deploy_model_payload,
        gcp_resources=deploy_model_gcp_resources,
    )

    logging.info('Model deployed successfully!')
  except Exception as e:  # pylint: disable=broad-exception-caught
    if isinstance(e, ValueError):
      raise
    logging.exception(str(e))
    sys.exit(13)


# pytype: enable=import-error
# pytype: enable=invalid-annotation
# pylint: enable=g-doc-args
# pylint: enable=g-import-not-at-top, invalid-name
