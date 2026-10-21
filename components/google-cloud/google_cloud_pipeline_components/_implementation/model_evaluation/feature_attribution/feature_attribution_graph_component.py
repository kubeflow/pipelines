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
"""Graph Component for feature attribution evaluation."""

from typing import List, NamedTuple

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model_evaluation.data_sampler.component import evaluation_data_sampler as EvaluationDataSamplerOp
from google_cloud_pipeline_components._implementation.model_evaluation.feature_attribution.feature_attribution_component import feature_attribution as ModelEvaluationFeatureAttributionOp
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
import kfp


@kfp.dsl.pipeline(name='feature-attribution-graph-component')
def feature_attribution_graph_component(  # pylint: disable=dangerous-default-value
    location: str,
    prediction_type: str,
    vertex_model: VertexModel,
    batch_predict_instances_format: str,
    batch_predict_gcs_destination_output_uri: str,
    batch_predict_gcs_source_uris: List[str] = [],  # pylint: disable=g-bare-generic
    batch_predict_bigquery_source_uri: str = '',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_bigquery_destination_output_uri: str = '',
    batch_predict_machine_type: str = 'n1-standard-16',
    batch_predict_starting_replica_count: int = 5,
    batch_predict_max_replica_count: int = 10,
    batch_predict_explanation_metadata: dict = {},  # pylint: disable=g-bare-generic
    batch_predict_explanation_parameters: dict = {},  # pylint: disable=g-bare-generic
    batch_predict_explanation_data_sample_size: int = 10000,
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
) -> NamedTuple('outputs', feature_attributions=kfp.dsl.Metrics):
  """A pipeline to compute feature attributions by sampling data for batch explanations.

  This pipeline guarantees support for AutoML Tabular models that contain a
  valid explanation_spec.

  Args:
    location: The GCP region that runs the pipeline components.
    prediction_type: The type of prediction the model is to produce.
      "classification", "regression", or "forecasting".
    vertex_model: The Vertex model artifact used for batch explanation.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. For more details
      about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location
      of the directory where the output is to be written to. In the given
      directory a new directory is created. Its name is
      `prediction-<model-display-name>-<job-create-time>`, where timestamp is in
      YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files
      `predictions_0001.<extension>`, `predictions_0002.<extension>`, ...,
      `predictions_N.<extension>` are created where `<extension>` depends on
      chosen `predictions_format`, and N may equal 0001 and depends on the total
      number of successfully predicted instances. If the Model has both
      `instance` and `prediction` schemata defined then each such file contains
      predictions as per the `predictions_format`. If prediction for any
      instance failed (partially or completely), then an additional
      `errors_0001.<extension>`, `errors_0002.<extension>`,...,
      `errors_N.<extension>` files are created (N depends on total number of
      failed predictions). These files contain the failed instances, as per
      their schema, followed by an additional `error` field which as value has
      `google.rpc.Status` containing only `code` and `message` fields. For more
      details about this output config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your
      instances to run batch prediction on. May contain wildcards. For more
      information on wildcards, see
      https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For
        more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_bigquery_source_uri: Google BigQuery URI to your instances to
      run batch prediction on. May contain wildcards. For more details about
      this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the
      predictions. Must be one of the Model's supportedOutputStorageFormats. For
      more details about this output config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_bigquery_destination_output_uri: The BigQuery project location
      where the output is to be written to. In the given project a new dataset
      is created with name `prediction_<model-display-name>_<job-create-time>`
      where is made BigQuery-dataset-name compatible (for example, most special
      characters become underscores), and timestamp is in
      YYYY_MM_DDThh_mm_ss_sssZ "based on ISO-8601" format. In the dataset two
      tables will be created, `predictions`, and `errors`. If the Model has both
      `instance` and `prediction` schemata defined then the tables have columns
      as follows: The `predictions` table contains instances for which the
      prediction succeeded, it has columns as per a concatenation of the Model's
      instance and prediction schemata. The `errors` table contains rows for
      which the prediction has failed, it has instance columns, as per the
      instance schema, followed by a single "errors" column, which as values has
      `google.rpc.Status` represented as a STRUCT, and containing only `code`
      and `message`.  For more details about this output config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_machine_type: The type of machine for running batch prediction
      on dedicated resources. If the Model supports DEDICATED_RESOURCES this
      config may be provided (and the job will use these resources). If the
      Model doesn't support AUTOMATIC_RESOURCES, this config must be provided.
      For more details about the BatchDedicatedResources, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources.
        For more details about the machine spec, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_starting_replica_count: The number of machine replicas used at
      the start of the batch operation. If not set, Vertex AI decides starting
      number, not greater than `max_replica_count`. Only used if `machine_type`
      is set.
    batch_predict_max_replica_count: The maximum number of machine replicas the
      batch operation may be scaled to. Only used if `machine_type` is set.
    batch_predict_explanation_metadata: Explanation metadata configuration for
      this BatchPredictionJob. Can be specified only if `generate_explanation`
      is set to `True`. This value overrides the value of
      `Model.explanation_metadata`. All fields of `explanation_metadata` are
      optional in the request. If a field of the `explanation_metadata` object
      is not populated, the corresponding field of the
      `Model.explanation_metadata` object is inherited. For more details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.
    batch_predict_explanation_parameters: Parameters to configure explaining for
      Model's predictions. Can be specified only if `generate_explanation` is
      set to `True`. This value overrides the value of
      `Model.explanation_parameters`. All fields of `explanation_parameters` are
      optional in the request. If a field of the `explanation_parameters` object
      is not populated, the corresponding field of the
      `Model.explanation_parameters` object is inherited. For more details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#ExplanationParameters.
    batch_predict_explanation_data_sample_size: Desired size to downsample the
      input dataset that will then be used for batch explanation.
    batch_predict_accelerator_type: The type of accelerator(s) that may be
      attached to the machine as per `batch_predict_accelerator_count`. Only
      used if `batch_predict_machine_type` is set. For more details about the
      machine spec, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the
      `batch_predict_machine_type`. Only used if `batch_predict_machine_type` is
      set.
    dataflow_machine_type: The Dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation
      components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation
      components.
    dataflow_service_account: Custom service account to run Dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
      https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.
    force_runner_mode: Indicate the runner mode to use forcely. Valid options
      are `Dataflow` and `DirectRunner`.
    project: The GCP project that runs the pipeline components. Defaults to the
      project in which the PipelineJob is run.

  Returns:
    A system.Metrics artifact with feature attributions.
  """
  outputs = NamedTuple('outputs', feature_attributions=kfp.dsl.Metrics)

  # Sample the input dataset for a quicker batch explanation.
  data_sampler_task = EvaluationDataSamplerOp(
      project=project,
      location=location,
      gcs_source_uris=batch_predict_gcs_source_uris,
      bigquery_source_uri=batch_predict_bigquery_source_uri,
      instances_format=batch_predict_instances_format,
      sample_size=batch_predict_explanation_data_sample_size,
      force_runner_mode=force_runner_mode,
  )

  # Run batch explain.
  batch_explain_task = ModelBatchPredictOp(
      project=project,
      location=location,
      model=vertex_model,
      job_display_name='model-registry-batch-explain-evaluation-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=data_sampler_task.outputs['gcs_output_directory'],
      bigquery_source_input_uri=data_sampler_task.outputs[
          'bigquery_output_table'
      ],
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      bigquery_destination_output_uri=batch_predict_bigquery_destination_output_uri,
      generate_explanation=True,
      explanation_parameters=batch_predict_explanation_parameters,
      explanation_metadata=batch_predict_explanation_metadata,
      machine_type=batch_predict_machine_type,
      starting_replica_count=batch_predict_starting_replica_count,
      max_replica_count=batch_predict_max_replica_count,
      encryption_spec_key_name=encryption_spec_key_name,
      accelerator_type=batch_predict_accelerator_type,
      accelerator_count=batch_predict_accelerator_count,
  )

  # Generate feature attributions from explanations.
  feature_attribution_task = ModelEvaluationFeatureAttributionOp(
      project=project,
      location=location,
      problem_type=prediction_type,
      predictions_format=batch_predict_predictions_format,
      predictions_gcs_source=batch_explain_task.outputs['gcs_output_directory'],
      predictions_bigquery_source=batch_explain_task.outputs[
          'bigquery_output_table'
      ],
      dataflow_machine_type=dataflow_machine_type,
      dataflow_max_workers_num=dataflow_max_num_workers,
      dataflow_disk_size_gb=dataflow_disk_size_gb,
      dataflow_service_account=dataflow_service_account,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
      force_runner_mode=force_runner_mode,
  )

  return outputs(
      feature_attributions=feature_attribution_task.outputs[
          'feature_attributions'
      ]
  )
