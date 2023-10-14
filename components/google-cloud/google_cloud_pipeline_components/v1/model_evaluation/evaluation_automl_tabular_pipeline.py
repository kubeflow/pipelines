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

from typing import Any, List, NamedTuple

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model import GetVertexModelOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluationOp
from google_cloud_pipeline_components.types.artifact_types import ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import RegressionMetrics
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from google_cloud_pipeline_components.v1.model_evaluation.classification_component import model_evaluation_classification as ModelEvaluationClassificationOp
from google_cloud_pipeline_components.v1.model_evaluation.regression_component import model_evaluation_regression as ModelEvaluationRegressionOp
import kfp


@kfp.dsl.pipeline(name='evaluation-automl-tabular-classification-pipeline')
def evaluation_automl_tabular_classification_pipeline(  # pylint: disable=dangerous-default-value
    location: str,
    model_name: str,
    target_field_name: str,
    batch_predict_instances_format: str,
    batch_predict_gcs_destination_output_uri: str,
    batch_predict_gcs_source_uris: List[str] = [],  # pylint: disable=g-bare-generic
    batch_predict_bigquery_source_uri: str = '',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_bigquery_destination_output_uri: str = '',
    batch_predict_machine_type: str = 'n1-standard-16',
    batch_predict_starting_replica_count: int = 5,
    batch_predict_max_replica_count: int = 10,
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    slicing_specs: List[Any] = [],  # pylint: disable=g-bare-generic
    evaluation_display_name: str = 'evaluation-automl-tabular-pipeline-{{$.pipeline_job_uuid}}',
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
) -> NamedTuple(
    'outputs',
    evaluation_metrics=ClassificationMetrics,
    evaluation_resource_name=str,
):
  # fmt: off
  """The evaluation AutoML tabular pipeline with no feature attribution for.

  classification models.

  This pipeline guarantees support for AutoML Tabular models. This pipeline does
  not include the target_field_data_remover component, which is needed for many
  tabular custom models.

  Args:
    location: The GCP region that runs the pipeline components.
    model_name: The Vertex model resource name to be imported and used for batch prediction.
    target_field_name: The target field's name. Formatted to be able to find nested columns, delimited by `.`. Prefixed with 'instance.' on the component for Vertex Batch Prediction.
    batch_predict_instances_format: The format in which instances are given, must be one of the Model's supportedInputStorageFormats. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location of the directory where the output is to be written to. In the given directory a new directory is created. Its name is `prediction-<model-display-name>-<job-create-time>`, where timestamp is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files `predictions_0001.<extension>`, `predictions_0002.<extension>`, ..., `predictions_N.<extension>` are created where `<extension>` depends on chosen `predictions_format`, and N may equal 0001 and depends on the total number of successfully predicted instances. If the Model has both `instance` and `prediction` schemata defined then each such file contains predictions as per the `predictions_format`. If prediction for any instance failed (partially or completely), then an additional `errors_0001.<extension>`, `errors_0002.<extension>`,..., `errors_N.<extension>` files are created (N depends on total number of failed predictions). These files contain the failed instances, as per their schema, followed by an additional `error` field which as value has `google.rpc.Status` containing only `code` and `message` fields. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your instances to run batch prediction on. May contain wildcards. For more information on wildcards, see https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_bigquery_source_uri: Google BigQuery URI to your instances to run batch prediction on. May contain wildcards. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the predictions. Must be one of the Model's supportedOutputStorageFormats. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_bigquery_destination_output_uri: The BigQuery project location where the output is to be written to. In the given project a new dataset is created with name `prediction_<model-display-name>_<job-create-time>` where is made BigQuery-dataset-name compatible (for example, most special characters become underscores), and timestamp is in YYYY_MM_DDThh_mm_ss_sssZ "based on ISO-8601" format. In the dataset two tables will be created, `predictions`, and `errors`. If the Model has both `instance` and `prediction` schemata defined then the tables have columns as follows: The `predictions` table contains instances for which the prediction succeeded, it has columns as per a concatenation of the Model's instance and prediction schemata. The `errors` table contains rows for which the prediction has failed, it has instance columns, as per the instance schema, followed by a single "errors" column, which as values has `google.rpc.Status` represented as a STRUCT, and containing only `code` and `message`.  For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_machine_type: The type of machine for running batch prediction on dedicated resources. If the Model supports DEDICATED_RESOURCES this config may be provided (and the job will use these resources). If the Model doesn't support AUTOMATIC_RESOURCES, this config must be provided. For more details about the BatchDedicatedResources, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_starting_replica_count: The number of machine replicas used at the start of the batch operation. If not set, Vertex AI decides starting number, not greater than `max_replica_count`. Only used if `machine_type` is set.
    batch_predict_max_replica_count: The maximum number of machine replicas the batch operation may be scaled to. Only used if `machine_type` is set.
    batch_predict_accelerator_type: The type of accelerator(s) that may be attached to the machine as per `batch_predict_accelerator_count`. Only used if `batch_predict_machine_type` is set. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the `batch_predict_machine_type`. Only used if `batch_predict_machine_type` is set.
    slicing_specs: List of `google.cloud.aiplatform_v1.types.ModelEvaluationSlice.SlicingSpec`. When provided, compute metrics for each defined slice. See sample code in https://cloud.google.com/vertex-ai/docs/pipelines/model-evaluation-component For more details on configuring slices, see https://cloud.google.com/python/docs/reference/aiplatform/latest/google.cloud.aiplatform_v1.types.ModelEvaluationSlice.
    dataflow_machine_type: The Dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation components.
    dataflow_service_account: Custom service account to run Dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. Example: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
    encryption_spec_key_name:  Customer-managed encryption key options. If set, resources created by this pipeline will be encrypted with the provided encryption key. Has the form: `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created.
    evaluation_display_name: The display name of the uploaded evaluation resource to the Vertex AI model.
    force_runner_mode: Indicate the runner mode to use forcely. Valid options are `Dataflow` and `DirectRunner`.
    project: The GCP project that runs the pipeline components. Defaults to the project in which the PipelineJob is run.

  Returns:
    A google.ClassificationMetrics artifact and imported evaluation_resource_name.
  """
  # fmt: on
  outputs = NamedTuple(
      'outputs',
      evaluation_metrics=ClassificationMetrics,
      evaluation_resource_name=str,
  )

  # Get the Vertex AI Model.
  get_model_task = GetVertexModelOp(model_name=model_name)

  # Run Vertex AI Batch Prediction.
  batch_predict_task = ModelBatchPredictOp(
      project=project,
      location=location,
      model=get_model_task.outputs['model'],
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=batch_predict_gcs_source_uris,
      bigquery_source_input_uri=batch_predict_bigquery_source_uri,
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      bigquery_destination_output_uri=batch_predict_bigquery_destination_output_uri,
      machine_type=batch_predict_machine_type,
      starting_replica_count=batch_predict_starting_replica_count,
      max_replica_count=batch_predict_max_replica_count,
      encryption_spec_key_name=encryption_spec_key_name,
      accelerator_type=batch_predict_accelerator_type,
      accelerator_count=batch_predict_accelerator_count,
  )

  # Run evaluation for a classification model.
  eval_task = ModelEvaluationClassificationOp(
      project=project,
      location=location,
      target_field_name=target_field_name,
      predictions_format=batch_predict_predictions_format,
      predictions_gcs_source=batch_predict_task.outputs['gcs_output_directory'],
      predictions_bigquery_source=batch_predict_task.outputs[
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
      model=get_model_task.outputs['model'],
      slicing_specs=slicing_specs,
  )

  # Import the evaluation result to Vertex AI.
  import_evaluation_task = ModelImportEvaluationOp(
      classification_metrics=eval_task.outputs['evaluation_metrics'],
      model=get_model_task.outputs['model'],
      dataset_type=batch_predict_instances_format,
      dataset_path=batch_predict_bigquery_source_uri,
      dataset_paths=batch_predict_gcs_source_uris,
      display_name=evaluation_display_name,
  )

  return outputs(
      evaluation_metrics=eval_task.outputs['evaluation_metrics'],
      evaluation_resource_name=import_evaluation_task.outputs[
          'evaluation_resource_name'
      ],
  )


@kfp.dsl.pipeline(name='evaluation-automl-tabular-regression-pipeline')
def evaluation_automl_tabular_regression_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    location: str,
    model_name: str,
    target_field_name: str,
    batch_predict_instances_format: str,
    batch_predict_gcs_destination_output_uri: str,
    batch_predict_gcs_source_uris: List[str] = [],  # pylint: disable=g-bare-generic
    batch_predict_bigquery_source_uri: str = '',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_bigquery_destination_output_uri: str = '',
    batch_predict_machine_type: str = 'n1-standard-16',
    batch_predict_starting_replica_count: int = 5,
    batch_predict_max_replica_count: int = 10,
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    evaluation_display_name: str = 'evaluation-automl-tabular-pipeline-{{$.pipeline_job_uuid}}',
    force_runner_mode: str = '',
) -> NamedTuple(
    'outputs',
    evaluation_metrics=RegressionMetrics,
    evaluation_resource_name=str,
):
  # fmt: off
  """The evaluation AutoML tabular pipeline with no feature attribution for regression models.

  This pipeline guarantees support for AutoML Tabular models. This pipeline does not include the target_field_data_remover component, which is needed for many tabular custom models.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    model_name: The Vertex model resource name to be imported and used for batch prediction.
    target_field_name: The target field's name. Formatted to be able to find nested columns, delimited by `.`. Prefixed with 'instance.' on the component for Vertex Batch Prediction.
    batch_predict_instances_format: The format in which instances are given, must be one of the Model's supportedInputStorageFormats. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location of the directory where the output is to be written to. In the given directory a new directory is created. Its name is `prediction-<model-display-name>-<job-create-time>`, where timestamp is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files `predictions_0001.<extension>`, `predictions_0002.<extension>`, ..., `predictions_N.<extension>` are created where `<extension>` depends on chosen `predictions_format`, and N may equal 0001 and depends on the total number of successfully predicted instances. If the Model has both `instance` and `prediction` schemata defined then each such file contains predictions as per the `predictions_format`. If prediction for any instance failed (partially or completely), then an additional `errors_0001.<extension>`, `errors_0002.<extension>`,..., `errors_N.<extension>` files are created (N depends on total number of failed predictions). These files contain the failed instances, as per their schema, followed by an additional `error` field which as value has `google.rpc.Status` containing only `code` and `message` fields. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your instances to run batch prediction on. May contain wildcards. For more information on wildcards, see https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_bigquery_source_uri: Google BigQuery URI to your instances to run batch prediction on. May contain wildcards. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the predictions. Must be one of the Model's supportedOutputStorageFormats. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_bigquery_destination_output_uri: The BigQuery project location where the output is to be written to. In the given project a new dataset is created with name `prediction_<model-display-name>_<job-create-time>` where is made BigQuery-dataset-name compatible (for example, most special characters become underscores), and timestamp is in YYYY_MM_DDThh_mm_ss_sssZ "based on ISO-8601" format. In the dataset two tables will be created, `predictions`, and `errors`. If the Model has both `instance` and `prediction` schemata defined then the tables have columns as follows: The `predictions` table contains instances for which the prediction succeeded, it has columns as per a concatenation of the Model's instance and prediction schemata. The `errors` table contains rows for which the prediction has failed, it has instance columns, as per the instance schema, followed by a single "errors" column, which as values has `google.rpc.Status` represented as a STRUCT, and containing only `code` and `message`.  For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_machine_type: The type of machine for running batch prediction on dedicated resources. If the Model supports DEDICATED_RESOURCES this config may be provided (and the job will use these resources). If the Model doesn't support AUTOMATIC_RESOURCES, this config must be provided. For more details about the BatchDedicatedResources, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_starting_replica_count: The number of machine replicas used at the start of the batch operation. If not set, Vertex AI decides starting number, not greater than `max_replica_count`. Only used if `machine_type` is set.
    batch_predict_max_replica_count: The maximum number of machine replicas the batch operation may be scaled to. Only used if `machine_type` is set.
    batch_predict_accelerator_type: The type of accelerator(s) that may be attached to the machine as per `batch_predict_accelerator_count`. Only used if `batch_predict_machine_type` is set. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the `batch_predict_machine_type`. Only used if `batch_predict_machine_type` is set.
    dataflow_machine_type: The Dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation components.
    dataflow_service_account: Custom service account to run Dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. Example: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
    encryption_spec_key_name:  Customer-managed encryption key options. If set, resources created by this pipeline will be encrypted with the provided encryption key. Has the form: `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created.
    evaluation_display_name: The display name of the uploaded evaluation resource to the Vertex AI model.
    force_runner_mode: Indicate the runner mode to use forcely. Valid options are `Dataflow` and `DirectRunner`.

  Returns:
    A google.RegressionMetrics artifact and imported evaluation_resource_name.
  """
  # fmt: on
  outputs = NamedTuple(
      'outputs',
      evaluation_metrics=RegressionMetrics,
      evaluation_resource_name=str,
  )

  # Get the Vertex AI Model.
  get_model_task = GetVertexModelOp(model_name=model_name)

  # Run Vertex AI Batch Prediction.
  batch_predict_task = ModelBatchPredictOp(
      project=project,
      location=location,
      model=get_model_task.outputs['model'],
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=batch_predict_gcs_source_uris,
      bigquery_source_input_uri=batch_predict_bigquery_source_uri,
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      bigquery_destination_output_uri=batch_predict_bigquery_destination_output_uri,
      machine_type=batch_predict_machine_type,
      starting_replica_count=batch_predict_starting_replica_count,
      max_replica_count=batch_predict_max_replica_count,
      encryption_spec_key_name=encryption_spec_key_name,
      accelerator_type=batch_predict_accelerator_type,
      accelerator_count=batch_predict_accelerator_count,
  )

  # Run evaluation for a regression model.
  eval_task = ModelEvaluationRegressionOp(
      project=project,
      location=location,
      target_field_name=target_field_name,
      predictions_format=batch_predict_predictions_format,
      predictions_gcs_source=batch_predict_task.outputs['gcs_output_directory'],
      predictions_bigquery_source=batch_predict_task.outputs[
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
      model=get_model_task.outputs['model'],
  )

  # Import the evaluation result to Vertex AI.
  import_evaluation_task = ModelImportEvaluationOp(
      regression_metrics=eval_task.outputs['evaluation_metrics'],
      model=get_model_task.outputs['model'],
      dataset_type=batch_predict_instances_format,
      dataset_path=batch_predict_bigquery_source_uri,
      dataset_paths=batch_predict_gcs_source_uris,
      display_name=evaluation_display_name,
  )

  return outputs(
      evaluation_metrics=eval_task.outputs['evaluation_metrics'],
      evaluation_resource_name=import_evaluation_task.outputs[
          'evaluation_resource_name'
      ],
  )


@kfp.dsl.pipeline(name='evaluation-automl-tabular-pipeline')
def evaluation_automl_tabular_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    location: str,
    prediction_type: str,
    model_name: str,
    target_field_name: str,
    batch_predict_instances_format: str,
    batch_predict_gcs_destination_output_uri: str,
    batch_predict_gcs_source_uris: List[str] = [],  # pylint: disable=g-bare-generic
    batch_predict_bigquery_source_uri: str = '',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_bigquery_destination_output_uri: str = '',
    batch_predict_machine_type: str = 'n1-standard-16',
    batch_predict_starting_replica_count: int = 5,
    batch_predict_max_replica_count: int = 10,
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    slicing_specs: List[Any] = [],  # pylint: disable=g-bare-generic
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    evaluation_display_name: str = 'evaluation-automl-tabular-pipeline-{{$.pipeline_job_uuid}}',
    force_runner_mode: str = '',
):
  # fmt: off
  """The evaluation AutoML tabular pipeline with no feature attribution.

  This pipeline guarantees support for AutoML Tabular classification and regression models. This pipeline does not include the target_field_data_remover component, which is needed for many tabular custom models and AutoML Tabular Forecasting.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    prediction_type: The type of prediction the model is to produce. "classification" or "regression".
    model_name: The Vertex model resource name to be imported and used for batch prediction.
    target_field_name: The target field's name. Formatted to be able to find nested columns, delimited by `.`. Prefixed with 'instance.' on the component for Vertex Batch Prediction.
    batch_predict_instances_format: The format in which instances are given, must be one of the Model's supportedInputStorageFormats. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location of the directory where the output is to be written to. In the given directory a new directory is created. Its name is `prediction-<model-display-name>-<job-create-time>`, where timestamp is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files `predictions_0001.<extension>`, `predictions_0002.<extension>`, ..., `predictions_N.<extension>` are created where `<extension>` depends on chosen `predictions_format`, and N may equal 0001 and depends on the total number of successfully predicted instances. If the Model has both `instance` and `prediction` schemata defined then each such file contains predictions as per the `predictions_format`. If prediction for any instance failed (partially or completely), then an additional `errors_0001.<extension>`, `errors_0002.<extension>`,..., `errors_N.<extension>` files are created (N depends on total number of failed predictions). These files contain the failed instances, as per their schema, followed by an additional `error` field which as value has `google.rpc.Status` containing only `code` and `message` fields. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your instances to run batch prediction on. May contain wildcards. For more information on wildcards, see https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_bigquery_source_uri: Google BigQuery URI to your instances to run batch prediction on. May contain wildcards. For more details about this input config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the predictions. Must be one of the Model's supportedOutputStorageFormats. For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_bigquery_destination_output_uri: The BigQuery project location where the output is to be written to. In the given project a new dataset is created with name `prediction_<model-display-name>_<job-create-time>` where is made BigQuery-dataset-name compatible (for example, most special characters become underscores), and timestamp is in YYYY_MM_DDThh_mm_ss_sssZ "based on ISO-8601" format. In the dataset two tables will be created, `predictions`, and `errors`. If the Model has both `instance` and `prediction` schemata defined then the tables have columns as follows: The `predictions` table contains instances for which the prediction succeeded, it has columns as per a concatenation of the Model's instance and prediction schemata. The `errors` table contains rows for which the prediction has failed, it has instance columns, as per the instance schema, followed by a single "errors" column, which as values has `google.rpc.Status` represented as a STRUCT, and containing only `code` and `message`.  For more details about this output config, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_machine_type: The type of machine for running batch prediction on dedicated resources. If the Model supports DEDICATED_RESOURCES this config may be provided (and the job will use these resources). If the Model doesn't support AUTOMATIC_RESOURCES, this config must be provided. For more details about the BatchDedicatedResources, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_starting_replica_count: The number of machine replicas used at the start of the batch operation. If not set, Vertex AI decides starting number, not greater than `max_replica_count`. Only used if `machine_type` is set.
    batch_predict_max_replica_count: The maximum number of machine replicas the batch operation may be scaled to. Only used if `machine_type` is set.
    batch_predict_accelerator_type: The type of accelerator(s) that may be attached to the machine as per `batch_predict_accelerator_count`. Only used if `batch_predict_machine_type` is set. For more details about the machine spec, see https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the `batch_predict_machine_type`. Only used if `batch_predict_machine_type` is set.
    slicing_specs: List of `google.cloud.aiplatform_v1.types.ModelEvaluationSlice.SlicingSpec`. When provided, compute metrics for each defined slice. See sample code in https://cloud.google.com/vertex-ai/docs/pipelines/model-evaluation-component For more details on configuring slices, see https://cloud.google.com/python/docs/reference/aiplatform/latest/google.cloud.aiplatform_v1.types.ModelEvaluationSlice.
    dataflow_machine_type: The Dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation components.
    dataflow_service_account: Custom service account to run Dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. Example: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
    encryption_spec_key_name:  Customer-managed encryption key options. If set, resources created by this pipeline will be encrypted with the provided encryption key. Has the form: `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created.
    evaluation_display_name: The display name of the uploaded evaluation resource to the Vertex AI model.
    force_runner_mode: Indicate the runner mode to use forcely. Valid options are `Dataflow` and `DirectRunner`.
  """
  # fmt: on
  with kfp.dsl.Condition(
      prediction_type == 'classification', name='classification'
  ):
    evaluation_automl_tabular_classification_pipeline(
        project=project,
        location=location,
        model_name=model_name,
        target_field_name=target_field_name,
        batch_predict_instances_format=batch_predict_instances_format,
        batch_predict_gcs_destination_output_uri=batch_predict_gcs_destination_output_uri,
        batch_predict_gcs_source_uris=batch_predict_gcs_source_uris,
        batch_predict_bigquery_source_uri=batch_predict_bigquery_source_uri,
        batch_predict_predictions_format=batch_predict_predictions_format,
        batch_predict_bigquery_destination_output_uri=batch_predict_bigquery_destination_output_uri,
        batch_predict_machine_type=batch_predict_machine_type,
        batch_predict_starting_replica_count=batch_predict_starting_replica_count,
        batch_predict_max_replica_count=batch_predict_max_replica_count,
        batch_predict_accelerator_type=batch_predict_accelerator_type,
        batch_predict_accelerator_count=batch_predict_accelerator_count,
        slicing_specs=slicing_specs,
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_num_workers=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
        evaluation_display_name=evaluation_display_name,
        force_runner_mode=force_runner_mode,
    )

  with kfp.dsl.Condition(prediction_type == 'regression', name='regression'):
    evaluation_automl_tabular_regression_pipeline(
        project=project,
        location=location,
        model_name=model_name,
        target_field_name=target_field_name,
        batch_predict_instances_format=batch_predict_instances_format,
        batch_predict_gcs_destination_output_uri=batch_predict_gcs_destination_output_uri,
        batch_predict_gcs_source_uris=batch_predict_gcs_source_uris,
        batch_predict_bigquery_source_uri=batch_predict_bigquery_source_uri,
        batch_predict_predictions_format=batch_predict_predictions_format,
        batch_predict_bigquery_destination_output_uri=batch_predict_bigquery_destination_output_uri,
        batch_predict_machine_type=batch_predict_machine_type,
        batch_predict_starting_replica_count=batch_predict_starting_replica_count,
        batch_predict_max_replica_count=batch_predict_max_replica_count,
        batch_predict_accelerator_type=batch_predict_accelerator_type,
        batch_predict_accelerator_count=batch_predict_accelerator_count,
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_num_workers=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
        evaluation_display_name=evaluation_display_name,
        force_runner_mode=force_runner_mode,
    )
