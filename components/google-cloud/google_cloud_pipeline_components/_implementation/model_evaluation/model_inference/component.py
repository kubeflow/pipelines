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
"""Third party inference component."""
from typing import Any, Dict, List, NamedTuple

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import LLMEvaluationTextGenerationOp
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Metrics
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import pipeline


_IMAGE_URI = 'gcr.io/model-evaluation-dev/llm_eval:clyu-test'


@container_component
def model_inference_component_internal(
    gcp_resources: OutputPath(str),
    gcs_output_directory: Output[Artifact],
    project: str,
    location: str,
    client_api_key_path: str,
    prediction_instances_source_uri: str,
    output_inference_gcs_prefix: str,
    inference_platform: str = 'openai_chat_completions',
    model_id: str = 'gpt-3.5-turbo',
    request_params: Dict[str, Any] = {},
    max_request_per_second: float = 3,
    max_tokens_per_minute: float = 100,
    display_name: str = 'third-party-inference',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: List[str] = [],
    encryption_spec_key_name: str = '',
):
  """Internal component to run Third Party Model Inference.

  Args:
      gcp_resources (str): Serialized gcp_resources proto tracking the custom
        job.
      model_inference_output_gcs_uri: The storage URI pointing toward a GCS
        location to store CSV for third party inference.
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      client_api_key_path: The GCS URI where client API key.
      output_inference_gcs_prefix: GCS file prefix for writing output.
      display_name: display name of the pipeline.
      machine_type: The machine type of this custom job. If not set, defaulted
        to `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      service_account: Sets the default service account for workload run-as
        account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      network: The full name of the Compute Engine network to which the job
        should be peered. For example, projects/12345/global/networks/myVPC.
        Format is of the form projects/{project}/global/networks/{network}.
        Where {project} is a project number, as in 12345, and {network} is a
        network name. Private services access must already be configured for the
        network. If left unspecified, the job is not peered with any network.
      reserved_ip_ranges: A list of names for the reserved ip ranges under the
        VPC network that can be used for this job. If set, we will deploy the
        job within the provided ip ranges. Otherwise, the job will be deployed
        to any ip ranges under the provided VPC network.
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
      gcp_resources (str): Serialized gcp_resources proto tracking the custom
        job.
      model_inference_output_gcs_uri: The storage URI pointing toward a
        GCS location to store CSV for third party inference.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=_IMAGE_URI,
          args=[
              f'--3p_model_inference={True}',
              f'--project={project}',
              f'--location={location}',
              f'--prediction_instances_source_uri={prediction_instances_source_uri}',
              f'--inference_platform={inference_platform}',
              f'--output_inference_gcs_prefix={output_inference_gcs_prefix}',
              f'--model_id={model_id}',
              f'--request_params={request_params}',
              f'--client_api_key_path={client_api_key_path}',
              f'--max_request_per_second={max_request_per_second}',
              f'--max_tokens_per_minute={max_tokens_per_minute}',
              # f'--gcs_output_directory={gcs_output_directory}',
              f'--gcs_output_directory={gcs_output_directory.path}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          reserved_ip_ranges=reserved_ip_ranges,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )


@pipeline(name='ModelEvaluationModelInferenceOp')
def model_inference_component(
    project: str,
    location: str,
    client_api_key_path: str,
    prediction_instances_source_uri: str,
    output_inference_gcs_prefix: str,
    inference_platform: str = 'openai_chat_completions',
    model_id: str = 'gpt-3.5-turbo',
    request_params: Dict[str, Any] = {},
    max_request_per_second: float = 3,
    max_tokens_per_minute: float = 100,
    display_name: str = 'third-party-inference',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: List[str] = [],
    encryption_spec_key_name: str = '',
) -> NamedTuple(
    'outputs',
    gcs_output_directory=Artifact,
):
  """Component to run Third Party Model Inference.

  Args:
    project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      client_api_key_path: The GCS URI where client API key.
      output_inference_gcs_prefix: GCS file prefix for writing output.
      display_name: display name of the pipeline.
      machine_type: The machine type of this custom job. If not set, defaulted
        to `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      service_account: Sets the default service account for workload run-as
        account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      network: The full name of the Compute Engine network to which the job
        should be peered. For example, projects/12345/global/networks/myVPC.
        Format is of the form projects/{project}/global/networks/{network}.
        Where {project} is a project number, as in 12345, and {network} is a
        network name. Private services access must already be configured for the
        network. If left unspecified, the job is not peered with any network.
      reserved_ip_ranges: A list of names for the reserved ip ranges under the
        VPC network that can be used for this job. If set, we will deploy the
        job within the provided ip ranges. Otherwise, the job will be deployed
        to any ip ranges under the provided VPC network.
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
    NamedTuple:
      model_inference_output_gcs_uri: CSV file output containing third
      party prediction results.
  """
  outputs = NamedTuple(
      'outputs',
      gcs_output_directory=Artifact,
  )

  inference_task = model_inference_component_internal(
      project=project,
      location=location,
      client_api_key_path=client_api_key_path,
      prediction_instances_source_uri=prediction_instances_source_uri,
      inference_platform=inference_platform,
      model_id=model_id,
      request_params=request_params,
      max_request_per_second=max_request_per_second,
      max_tokens_per_minute=max_tokens_per_minute,
      output_inference_gcs_prefix=output_inference_gcs_prefix,
      display_name=display_name,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      reserved_ip_ranges=reserved_ip_ranges,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  return outputs(
      gcs_output_directory=inference_task.outputs['gcs_output_directory'],
  )


@pipeline(name='ModelEvaluationModelInferenceAndEvaluationPipeline')
def model_inference_and_evaluation_component(
    project: str,
    location: str,
    client_api_key_path: str,
    prediction_instances_source_uri: str,
    output_inference_gcs_prefix: str,
    target_field_name: str = '',
    inference_platform: str = 'openai_chat_completions',
    model_id: str = 'gpt-3.5-turbo',
    request_params: Dict[str, Any] = {},
    max_request_per_second: float = 3,
    max_tokens_per_minute: float = 100,
    display_name: str = 'third-party-inference',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: List[str] = [],
    encryption_spec_key_name: str = '',
) -> NamedTuple(
    'outputs',
    gcs_output_directory=Artifact,
    evaluation_metrics=Metrics,
):
  """Component tun Third Party Model Inference and evaluation.

  Args:
    project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      client_api_key_path: The GCS URI where client API key.
      output_inference_gcs_prefix: GCS file prefix for writing output.
      display_name: display name of the pipeline.
      machine_type: The machine type of this custom job. If not set, defaulted
        to `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      service_account: Sets the default service account for workload run-as
        account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      network: The full name of the Compute Engine network to which the job
        should be peered. For example, projects/12345/global/networks/myVPC.
        Format is of the form projects/{project}/global/networks/{network}.
        Where {project} is a project number, as in 12345, and {network} is a
        network name. Private services access must already be configured for the
        network. If left unspecified, the job is not peered with any network.
      reserved_ip_ranges: A list of names for the reserved ip ranges under the
        VPC network that can be used for this job. If set, we will deploy the
        job within the provided ip ranges. Otherwise, the job will be deployed
        to any ip ranges under the provided VPC network.
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
    NamedTuple:
      model_inference_output_gcs_uri: CSV file output containing third
      party prediction results.
  """
  outputs = NamedTuple(
      'outputs',
      gcs_output_directory=Artifact,
      evaluation_metrics=Metrics,
  )

  inference_task = model_inference_component_internal(
      project=project,
      location=location,
      client_api_key_path=client_api_key_path,
      prediction_instances_source_uri=prediction_instances_source_uri,
      inference_platform=inference_platform,
      model_id=model_id,
      request_params=request_params,
      max_request_per_second=max_request_per_second,
      max_tokens_per_minute=max_tokens_per_minute,
      output_inference_gcs_prefix=output_inference_gcs_prefix,
      display_name=display_name,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      reserved_ip_ranges=reserved_ip_ranges,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  if inference_platform == 'openai_chat_completions':
    prediction_field_name = 'predictions.0.message.content'
  elif inference_platform == 'anthropic_predictions':
    prediction_field_name = 'predictions'
  else:
    prediction_field_name = ''

  eval_task = LLMEvaluationTextGenerationOp(
      project=project,
      location=location,
      evaluation_task='text-generation',
      target_field_name=target_field_name,
      prediction_field_name=prediction_field_name,
      predictions_format='jsonl',
      joined_predictions_gcs_source=inference_task.outputs[
          'gcs_output_directory'
      ],
      machine_type=machine_type,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  return outputs(
      gcs_output_directory=inference_task.outputs['gcs_output_directory'],
      evaluation_metrics=eval_task.outputs['evaluation_metrics'],
  )
