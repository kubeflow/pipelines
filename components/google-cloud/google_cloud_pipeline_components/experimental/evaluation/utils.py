"""Util functions for Model Evaluation pipelines."""

import os
import pathlib
from typing import Any, Dict, List, Tuple


def get_sdk_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    model_name: str,
    target_column_name: str,
    prediction_type: str,
    batch_predict_gcs_source_uris: List[str],
    batch_predict_instances_format: str,
    batch_predict_machine_type: str = 'n1-standard-16',
    batch_predict_starting_replica_count: int = 25,
    batch_predict_max_replica_count: int = 25,
    class_names: List[str] = ['0', '1'],
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_max_num_workers: int = 25,
    dataflow_disk_size_gb: int = 50,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the evaluation sdk pipeline and parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    model_name: The Vertex model resource name to be imported and used for batch
      prediction.
    target_column_name: The target column name.
    prediction_type: The type of prediction the Model is to produce.
      "classification" or "regression".
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your
      instances to run batch prediction on. May contain wildcards. For more
      information on wildcards, see
      https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For
        more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl". For more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
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
      Default is 10.
    class_names: The list of class names that the ground truth can be, in the
      same order they appear in the batch predict predictions output.
    dataflow_machine_type: The dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation
      components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation
      components.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if prediction_type == 'regression':
    prediction_score_column = 'prediction.value'
    prediction_label_column = ''
  elif prediction_type == 'classification':
    prediction_score_column = 'prediction.scores'
    prediction_label_column = 'prediction.classes'

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'model_name':
          model_name,
      'target_column_name':
          target_column_name,
      'prediction_type':
          prediction_type,
      'class_names':
          class_names,
      'batch_predict_gcs_source_uris':
          batch_predict_gcs_source_uris,
      'batch_predict_instances_format':
          batch_predict_instances_format,
      'batch_predict_machine_type':
          batch_predict_machine_type,
      'batch_predict_starting_replica_count':
          batch_predict_starting_replica_count,
      'batch_predict_max_replica_count':
          batch_predict_max_replica_count,
      'prediction_score_column':
          prediction_score_column,
      'prediction_label_column':
          prediction_label_column,
      'dataflow_machine_type':
          dataflow_machine_type,
      'dataflow_max_num_workers':
          dataflow_max_num_workers,
      'dataflow_disk_size_gb':
          dataflow_disk_size_gb,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'templates', 'sdk_pipeline.json')
  return pipeline_definition_path, parameter_values
