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
"""Vertex LLM standalone Evaluation for text generation task."""

from typing import List, NamedTuple

from google_cloud_pipeline_components._implementation.model_evaluation import LLMEvaluationTextGenerationOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluationOp
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from kfp import dsl
from kfp.dsl import Metrics

_PIPELINE_NAME = 'evaluation-llm-text-generation-pipeline'


@dsl.pipeline(name=_PIPELINE_NAME)
def evaluation_llm_text_generation_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    location: str,
    batch_predict_gcs_source_uris: List[str],
    batch_predict_gcs_destination_output_uri: str,
    model_name: str = 'publishers/google/models/text-bison@001',
    evaluation_task: str = 'text-generation',
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
) -> NamedTuple(
    'outputs', evaluation_metrics=Metrics, evaluation_resource_name=str
):
  """LLM Text Generation Evaluation pipeline.

  This pipeline supports evaluating large language models, publisher or managed
  models, performing the following generative tasks: `summarization`,
  `question-answering`, and `text-generation`.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your
      instances data to run batch prediction on. The instances data should also
      contain the ground truth (target) data, used for evaluation. May contain
      wildcards. For more information on wildcards, see
      https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For
        more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location
      of the directory where the output is to be written to.
    model_name: The Model name used to run evaluation. Must be a publisher Model
      or a managed Model sharing the same ancestor location. Starting this job
      has no impact on any existing deployments of the Model and their
      resources.
    evaluation_task: The task that the large language model will be evaluated
      on. The evaluation component computes a set of metrics relevant to that
      specific task. Currently supported tasks are: `summarization`,
      `question-answering`, `text-generation`.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. Only "jsonl" is
      currently supported. For more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the
      predictions. Must be one of the Model's supportedOutputStorageFormats.
      Only "jsonl" is currently supported. For more details about this output
      config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
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
      be peered. For example, `projects/12345/global/networks/myVPC`. Format is
      of the form `projects/{project}/global/networks/{network}`. Where
      `{project}` is a project number, as in `12345`, and `{network}` is a
      network name, as in `myVPC`. To specify this field, you must have already
      configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.

  Returns:
    evaluation_metrics: Metrics Artifact for LLM Text Generation.
    evaluation_resource_name: If run on an user's managed VertexModel, the
      imported evaluation resource name. Empty if run on a publisher model.
  """
  outputs = NamedTuple(
      'outputs',
      evaluation_metrics=Metrics,
      evaluation_resource_name=str,
  )

  get_vertex_model_task = dsl.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

  batch_predict_task = ModelBatchPredictOp(
      project=project,
      location=location,
      model=get_vertex_model_task.outputs['artifact'],
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=batch_predict_gcs_source_uris,
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  eval_task = LLMEvaluationTextGenerationOp(
      project=project,
      location=location,
      evaluation_task=evaluation_task,
      target_field_name='instance.ground_truth',
      prediction_field_name='predictions.content',
      predictions_format=batch_predict_predictions_format,
      joined_predictions_gcs_source=batch_predict_task.outputs[
          'gcs_output_directory'
      ],
      display_name=_PIPELINE_NAME,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  import_evaluation_task = ModelImportEvaluationOp(
      metrics=eval_task.outputs['evaluation_metrics'],
      model=get_vertex_model_task.outputs['artifact'],
      problem_type=evaluation_task,
      dataset_type=batch_predict_predictions_format,
      dataset_paths=batch_predict_gcs_source_uris,
      display_name=_PIPELINE_NAME,
  )

  return outputs(
      evaluation_metrics=eval_task.outputs['evaluation_metrics'],
      evaluation_resource_name=import_evaluation_task.outputs[
          'evaluation_resource_name'
      ],
  )
