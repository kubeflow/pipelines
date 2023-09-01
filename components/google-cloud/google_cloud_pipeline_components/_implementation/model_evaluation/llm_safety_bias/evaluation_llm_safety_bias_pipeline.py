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
"""Vertex LLM Safety Bias Evaluation Pipeline."""

from typing import NamedTuple

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model_evaluation.llm_safety_bias.component import llm_safety_bias_metrics as LLMSafetyBiasMetricsOp
from google_cloud_pipeline_components.types.artifact_types import VertexBatchPredictionJob
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Output
from kfp.dsl import OutputPath

_PRIVATE_BP_IMAGE = (
    'us-docker.pkg.dev/vertex-ai-restricted/llm-eval/private-bp:v0.1'
)


@container_component
def private_model_batch_predict(
    location: str,
    model_name: str,
    gcp_resources: OutputPath(str),
    batchpredictionjob: Output[VertexBatchPredictionJob],
    gcs_output_directory: Output[Artifact],
    bp_output_gcs_uri: OutputPath(str),
    predictions_format: str = 'jsonl',
    job_display_name: str = 'evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
    accelerator_type: str = '',
    accelerator_count: int = 0,
    encryption_spec_key_name: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--project',
          project,
          '--location',
          location,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"display_name": "',
              job_display_name,
              '", ',
              '"job_spec": {"worker_pool_specs": [{"replica_count":"1',
              '", "machine_spec": {"machine_type": "e2-standard-4',
              '"},',
              '"container_spec": {"image_uri":"',
              _PRIVATE_BP_IMAGE,
              '", "args": ["--project=',
              project,
              '", "--location=',
              location,
              '", "--model=',
              model_name,
              '", "--instances_format=',
              'jsonl',
              '", "--predictions_format=',
              predictions_format,
              '", "--accelerator_type=',
              accelerator_type,
              '", "--accelerator_count=',
              accelerator_count,
              '", "--bp_output_gcs_uri=',
              bp_output_gcs_uri,
              '", "--executor_input={{$.json_escape[1]}}"]}}]',
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name,
              '"}',
              '}}',
          ]),
      ],
  )


@dsl.pipeline(name='evaluation-llm-safety-bias-pipeline')
def evaluation_llm_safety_bias_pipeline(
    project: str,
    location: str,
    model_name: str,
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
) -> NamedTuple('outputs', llm_safety_bias_evaluation_metrics=Artifact):
  """LLM RAI Safety Bias Evaluation pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    model_name: The Model name used to run evaluation. Must be a publisher Model
      or a managed Model sharing the same ancestor location. Starting this job
      has no impact on any existing deployments of the Model and their
      resources.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16. More details:
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
    encryption_spec_key_name: Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.

    Returns:
      llm_safety_bias_evaluation_metrics: Metrics Artifact for LLM Safety Bias.
  """
  outputs = NamedTuple(
      'outputs',
      llm_safety_bias_evaluation_metrics=Artifact,
  )
  batch_predict_task = private_model_batch_predict(
      project=project,
      location=location,
      model_name=model_name,
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      predictions_format='jsonl',
      encryption_spec_key_name=encryption_spec_key_name,
  )

  llm_safety_bias_task = LLMSafetyBiasMetricsOp(
      project=project,
      predictions_gcs_source=batch_predict_task.outputs['bp_output_gcs_uri'],
      slice_spec_gcs_source=(
          'gs://vertex-evaluation-llm-dataset-us-central1/safety_slicing_spec/all_subgroup.json'
      ),
      location=location,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )
  return outputs(
      llm_safety_bias_evaluation_metrics=llm_safety_bias_task.outputs[
          'llm_safety_bias_evaluation_metrics'
      ]
  )
