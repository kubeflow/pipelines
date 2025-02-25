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
"""Graph component for uploading and deploying a tuned model adapter."""

import json
from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import deploy_llm_model
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import upload_llm_model
import kfp

PipelineOutput = NamedTuple(
    'Outputs', model_resource_name=str, endpoint_resource_name=str
)


@kfp.dsl.pipeline(
    name='llm-deployment-graph',
    description='Uploads a tuned model and deploys it to an endpoint.',
)
def pipeline(
    output_adapter_path: str,
    large_model_reference: str,
    policy_model_reference: str,
    model_display_name: Optional[str] = None,
    deploy_model: bool = True,
    upload_model: bool = True,
    encryption_spec_key_name: str = '',
    upload_location: str = _placeholders.LOCATION_PLACEHOLDER,
    regional_endpoint: str = '',
) -> PipelineOutput:
  # fmt: off
  """Uploads a tuned language model and (optionally) deploys it to an endpoint.

  Args:
    output_adapter_path: Path to the trained model adapter if LoRA tuning was used.
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    policy_model_reference: The name of the model for deployment. The name should be in capitalized snake case format.
    model_display_name: Name of the fine-tuned model shown in the Model Registry. If not provided, a default name will be created.
    deploy_model: Whether to deploy the model to an endpoint in `us-central1`. Default is True.
    encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key. Note that this is not supported for TPU at the moment.
    upload_location: Region to upload and deploy the model to. Default is the location used to run the pipeline components.
    regional_endpoint: Regional endpoint to upload the model.

  Returns:
    model_resource_name: Path to the model uploaded to the Model Registry. This will be an empty string if the model was not deployed.
    endpoint_resource_name: Path the Online Prediction Endpoint. This will be an empty string if the model was not deployed.
  """
  # fmt: on
  upload_task = upload_llm_model.refined_upload_llm_model(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=upload_location,
      regional_endpoint=regional_endpoint,
      artifact_uri=output_adapter_path,
      model_display_name=model_display_name,
      model_reference_name=large_model_reference,
      upload_model=upload_model,
      encryption_spec_key_name=encryption_spec_key_name,
      tune_type='rlhf',
  ).set_display_name('Upload Model')

  deploy_task = deploy_llm_model.deploy_llm_model(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=upload_location,
      model_resource_name=upload_task.outputs['model_resource_name'],
      display_name=model_display_name,
      regional_endpoint=regional_endpoint,
      deploy_model=deploy_model,
      encryption_spec_key_name=encryption_spec_key_name,
  ).set_display_name('Deploy Model')
  return PipelineOutput(
      model_resource_name=upload_task.outputs['model_resource_name'],
      endpoint_resource_name=deploy_task.outputs['endpoint_resource_name'],
  )
