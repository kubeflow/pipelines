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
"""Endpoint batch predict component used in KFP pipelines."""

from typing import Dict, List, NamedTuple, Optional, Union
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER

_IMAGE_URI = 'us-docker.pkg.dev/vertex-evaluation/public/llm:v0.5'


@dsl.component(base_image=version.LLM_EVAL_IMAGE_TAG)
def add_json_escape_parameters(parameters: dict) -> str:
  if not parameters:
    return
  import json

  json_escaped_parameters = json.dumps(parameters).replace('"', '\\"')
  return json_escaped_parameters


@dsl.component(base_image=version.LLM_EVAL_IMAGE_TAG)
def add_json_escape_paths(paths: list) -> str:
  if not paths:
    return
  import json

  json_escaped_paths = json.dumps(paths).replace('"', '\\"')
  return json_escaped_paths


@container_component
def endpoint_batch_predict(
    gcp_resources: OutputPath(str),
    gcs_output_directory: Output[Artifact],
    project: str,
    location: str,
    source_gcs_uris: str,
    gcs_destination_output_uri_prefix: Optional[str] = '',
    model_parameters: Optional[str] = None,
    endpoint_id: Optional[str] = '',
    publisher_model: Optional[str] = '',
    qms_override: Optional[str] = None,
    enable_retry: bool = False,
    display_name: str = 'endpoint_batch_predict',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
):
  """Returns the batch prediction results for a given batch of instances.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      source_gcs_uris: Google Cloud Storage URI(-s) to your instances to run
        prediction on. The stored file format should be jsonl and each line
        contains one Prediction instance. Instance should match Deployed model's
        instance schema
      gcs_destination_output_uri_prefix: The Google Cloud Storage location of
        the directory where the output is to be written to. In the given
        directory a new directory is created. Its name is
        `prediction-model-<job-create-time>`, where timestamp is in
        YYYY-MM-DD-hh:mm:ss.sss format. Inside of it is file results.jsonl
      endpoint_id: Required if no publisher_model is provided. The Endpoint ID
        of the deployed the LLM to serve the prediction. When endpoint_id and
        publisher_model are both provided, publisher_model will be used.
      model_parameters: The parameters that govern the prediction.
      publisher_model: Required if no endpoint_id is provided. Name of the
        Publisher model.
      location: Project the LLM Model is in.
      qms_override: Manual control of a large language model's qms. Write up
        when there's an approved quota increase for a LLM. Write down when
        limiting qms of a LLM for this pipeline. Should be provided as a
        dictionary, for example {'text-bison': 20}. For deployed model which
        doesn't have google-vertex-llm-tuning-base-model-id label, override the
        default here.
      enable_retry: Retry for Service Unavailable error. When enabled, the
        component resend the request in 60 seconds
      display_name: The name of the Evaluation job.
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
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.
      gcs_output_directory (str):
        GCS directory where endpoint batch prediction results are stored.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=_IMAGE_URI,
          args=[
              f'--endpoint_batch_predict={True}',
              f'--project={project}',
              f'--location={location}',
              f'--source_gcs_uris={source_gcs_uris}',
              f'--model_parameters={model_parameters}',
              f'--gcs_destination_output_uri_prefix={gcs_destination_output_uri_prefix}',
              f'--endpoint_id={endpoint_id}',
              f'--publisher_model={publisher_model}',
              f'--qms_override={qms_override}',
              f'--enable_retry={enable_retry}',
              f'--gcs_output_directory={gcs_output_directory.path}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--gcp_resources={gcp_resources}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )


@dsl.pipeline(name='evaluation-llm-endpoint-batch-predict')
def evaluation_llm_endpoint_batch_predict_pipeline_graph_component(
    project: str,
    location: str,
    source_gcs_uris: List[str],
    gcs_destination_output_uri_prefix: str = f'{PIPELINE_ROOT_PLACEHOLDER}/batch_predict_output',
    model_parameters: Optional[Dict[str, Union[int, float]]] = {},
    endpoint_id: Optional[str] = '',
    publisher_model: Optional[str] = '',
    qms_override: Optional[Dict[str, Union[int, float]]] = {},
    enable_retry: Optional[bool] = False,
    display_name: str = 'endpoint_batch_predict',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
) -> NamedTuple('outputs', gcs_output_directory=Artifact):
  """The First Party Model Endpoint Batch Predict Pipeline.

  Args:
    project: Required. The GCP project that runs the pipeline components.
    location: Required. The GCP region that runs the pipeline components.
    source_gcs_uris: Google Cloud Storage URI to your instances to run
      prediction on. The stored file format should be jsonl and each line
      contains one Prediction instance. Instance should match Deployed model's
      instance schema
    gcs_destination_output_uri_prefix: The Google Cloud Storage location of the
      directory where the output is to be written to. In the given directory a
      new directory is created. Its name is
      `prediction-model-<job-create-time>`, where timestamp is in
      YYYY-MM-DD-hh:mm:ss.sss format. Inside of it is file results.jsonl
    endpoint_id: Required if no publisher_model is provided. The Endpoint ID of
      the deployed the LLM to serve the prediction. When endpoint_id and
      publisher_model are both provided, publisher_model will be used.
    model_parameters: The parameters that govern the prediction.
    publisher_model: Required if no endpoint_id is provided. Name of the
      Publisher model.
    location: Project the LLM Model is in.
    qms_override: Manual control of a large language model's qms. Write up when
      there's an approved quota increase for a LLM. Write down when limiting qms
      of a LLM for this pipeline. Should be provided as a dictionary, for
      example {'text-bison': 20}. For deployed model which doesn't have
      google-vertex-llm-tuning-base-model-id label, override the default here.
    enable_retry: Retry for Service Unavailable error. When enabled, the
      component resend the request in 60 seconds
    display_name: The name of the Evaluation job.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
      https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account. If
      unspecified, the Vertex AI Custom Code Service
      Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
      for the CustomJob's project.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, projects/12345/global/networks/myVPC. Format is of
      the form projects/{project}/global/networks/{network}. Where {project} is
      a project number, as in 12345, and {network} is a network name. Private
      services access must already be configured for the network. If left
      unspecified, the job is not peered with any network.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    NamedTuple:
      gcs_output_directory:
        Artifact tracking the batch prediction job output.
  """
  outputs = NamedTuple('outputs', gcs_output_directory=Artifact)

  endpoint_batch_predict_task = endpoint_batch_predict(
      project=project,
      location=location,
      source_gcs_uris=add_json_escape_paths(paths=source_gcs_uris).output,
      gcs_destination_output_uri_prefix=gcs_destination_output_uri_prefix,
      model_parameters=add_json_escape_parameters(
          parameters=model_parameters
      ).output,
      endpoint_id=endpoint_id,
      publisher_model=publisher_model,
      qms_override=add_json_escape_parameters(parameters=qms_override).output,
      enable_retry=enable_retry,
      display_name=display_name,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  return outputs(
      gcs_output_directory=endpoint_batch_predict_task.outputs[
          'gcs_output_directory'
      ]
  )
