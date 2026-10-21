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
"""Text Generation LLM Evaluation component."""

from typing import List

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Metrics
from kfp.dsl import Output
from kfp.dsl import OutputPath


@dsl.container_component
def model_evaluation_text_generation(
    gcp_resources: OutputPath(str),
    evaluation_metrics: Output[Metrics],
    row_based_metrics: Output[Metrics],
    project: str,
    location: str,
    model_name: str,
    evaluation_task: str = 'text-generation',
    target_field_name: str = 'instance.output_text',
    prediction_field_name: str = 'predictions.content',
    predictions_format: str = 'jsonl',
    joined_predictions_gcs_source: dsl.Input[Artifact] = None,
    predictions_gcs_source: dsl.Input[Artifact] = None,
    ground_truth_gcs_source: str = '',
    enable_row_based_metrics: bool = False,
    display_name: str = 'model-evaluation-text-generation',
    machine_type: str = 'e2-standard-4',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: List[str] = [],
    encryption_spec_key_name: str = '',
):
  """Computes evaluation metrics of a text generation model.

  Supports evaluating large language models performing the following generative
  tasks: `summarization`, `question-answering`, and `text-generation`.

  Args:
    project: The GCP project that runs the pipeline component.
    location: The GCP region that runs the pipeline component.
    model_name: The name of the model to be evaluated.
    evaluation_task: The task that the large language model will be evaluated
      on. The evaluation component computes a set of metrics relevant to that
      specific task. Currently supported tasks are: `summarization`,
      `question-answering`, and `text-generation`.
    target_field_name: The full name path of the features target field in the
      predictions file. Formatted to be able to find nested columns, delimited
      by `.`. Alternatively referred to as the ground truth (or
      ground_truth_column) field. If not set, defaulted to `inputs.output_text`.
    prediction_field_name: The full name path of the prediction field in the
      prediction file. Formatted to be able to find nested columns, delimited by
      `.`. If not set, defaulted to `predictions.content`.
    predictions_format: The file format for the LLM Batch Prediction results.
      `jsonl` is currently the only allowed format. If not set, defaulted to
      `jsonl`.
    joined_predictions_gcs_source: An Artifact with an URI pointing toward a GCS
      directory or a GCS file with joined prediction & ground truth files to be
      used for this evaluation.
    predictions_gcs_source: An Artifact with an URI pointing toward a GCS
      directory with only prediction files to be used for this evaluation.
    ground_truth_gcs_source: A storage URI pointing toward a GCS directory with
      only ground truth files to be used for this evaluation.
    display_name: The name of the evaluation custom job.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-standard-4`. More details:
      https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account. If
      unspecified, the Vertex AI Custom Code Service
      Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
      for the CustomJob's project.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, `projects/12345/global/networks/myVPC`. Format is
      of the form `projects/{project}/global/networks/{network}`. Where
      `{project}` is a project number, as in `12345`, and `{network}` is a
      network name, as in `myVPC`. To specify this field, you must have already
      configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    reserved_ip_ranges: A list of names for the reserved ip ranges under the VPC
      network that can be used for this job. If set, we will deploy the job
      within the provided ip ranges. Otherwise, the job will be deployed to any
      ip ranges under the provided VPC network.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.

  Returns:
    gcp_resources: Serialized gcp_resources proto tracking the custom job.
      For more details, see
      https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
    evaluation_metrics: `Metrics` artifact representing the language model
      evaluation metrics.
    row_based_metrics: `Metrics` artifact representing the language model
      evaluation metrics of each instance. This is only available if
      enable_row_based_metrics is set to True.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              f'--model_name={model_name}',
              f'--evaluation_task={evaluation_task}',
              f'--target_field_name={target_field_name}',
              f'--prediction_field_name={prediction_field_name}',
              f'--predictions_format={predictions_format}',
              f'--joined_predictions_gcs_source={joined_predictions_gcs_source.uri}',
              f'--predictions_gcs_source={predictions_gcs_source.uri}',
              f'--ground_truth_gcs_source={ground_truth_gcs_source}',
              f'--evaluation_metrics_output_path={evaluation_metrics.path}',
              f'--enable_row_based_metrics={enable_row_based_metrics}',
              f'--row_based_metrics_output_path={row_based_metrics.path}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          reserved_ip_ranges=reserved_ip_ranges,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
