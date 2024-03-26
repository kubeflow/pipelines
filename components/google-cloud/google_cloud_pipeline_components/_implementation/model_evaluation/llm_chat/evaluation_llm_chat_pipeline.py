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
"""LLM chat evaluation pipeline."""

from google_cloud_pipeline_components._implementation.model_evaluation.llm_chat_preprocessor.component import llm_chat_preprocessor as LLMChatPreprocessorOp
import kfp

_PIPELINE_NAME = 'evaluation-llm-chat-pipeline'


@kfp.dsl.pipeline(name=_PIPELINE_NAME)
def evaluation_llm_chat_pipeline(
    project: str,
    location: str,
    input_dataset_gcs_path: str,
    evaluate_turns: str = 'final',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
):
  """The LLM Chat Evaluation Pipeline.

  Args:
    project: Required. The GCP project that runs the pipeline components.
    location: Required. The GCP region that runs the pipeline components.
    input_dataset_gcs_path: Required. The GCS path for json file containing
      input dataset for chat evaluation.
    evaluate_turns: turns that customers would like to evaluate. Supported
      examples are "all", "final", "1,3,final", etc.
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
    encryption_spec_key_name: Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.
  """

  preprocessing_task = LLMChatPreprocessorOp(
      project=project,
      location=location,
      input_dataset_gcs_path=input_dataset_gcs_path,
      evaluate_turns=evaluate_turns,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )
