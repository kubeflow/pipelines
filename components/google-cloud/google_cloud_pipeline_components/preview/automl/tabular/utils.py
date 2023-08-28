"""Util functions for AutoML Tabular pipeline."""

import json
import os
import pathlib
from typing import Any, Dict, List, Optional, Tuple, Union
import uuid
import warnings

_DEFAULT_NUM_PARALLEL_TRAILS = 35
_DEFAULT_STAGE_2_NUM_SELECTED_TRAILS = 5
_NUM_FOLDS = 5
_DISTILL_TOTAL_TRIALS = 100
_EVALUATION_BATCH_PREDICT_MACHINE_TYPE = 'n1-highmem-8'
_EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT = 20
_EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT = 20
_EVALUATION_BATCH_EXPLAIN_MACHINE_TYPE = 'n1-highmem-8'
_EVALUATION_BATCH_EXPLAIN_STARTING_REPLICA_COUNT = 10
_EVALUATION_BATCH_EXPLAIN_MAX_REPLICA_COUNT = 10
_EVALUATION_DATAFLOW_MACHINE_TYPE = 'n1-standard-4'
_EVALUATION_DATAFLOW_STARTING_NUM_WORKERS = 10
_EVALUATION_DATAFLOW_MAX_NUM_WORKERS = 100
_EVALUATION_DATAFLOW_DISK_SIZE_GB = 50

# Needed because we reference the AutoML Tabular V1 pipeline.
_GCPC_STAGING_PATH = pathlib.Path(
    __file__
).parent.parent.parent.parent.resolve()
_GCPC_GA_TABULAR_PATH = str(_GCPC_STAGING_PATH / 'v1' / 'automl' / 'tabular')


def _update_parameters(
    parameter_values: Dict[str, Any], new_params: Dict[str, Any]
):
  parameter_values.update(
      {param: value for param, value in new_params.items() if value is not None}
  )


def _generate_model_display_name() -> str:
  """Automatically generates a model_display_name.

  Returns:
    model_display_name.
  """
  return f'tabular-workflow-model-{uuid.uuid4()}'


# TODO(b/277393122): Once we finish L2L+FTE integration, add use_fte flag
# to signify FTE usage instead of the presence of num_selected_features.
def _get_default_pipeline_params(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    stage_2_num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[float] = None,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    optimization_objective_recall_value: Optional[float] = None,
    optimization_objective_precision_value: Optional[float] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: Optional[str] = None,
    stats_and_example_gen_dataflow_max_num_workers: Optional[int] = None,
    stats_and_example_gen_dataflow_disk_size_gb: Optional[int] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: Optional[str] = None,
    additional_experiments: Optional[Dict[str, Any]] = None,
    dataflow_service_account: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    apply_feature_selection_tuning: bool = False,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_batch_explain_machine_type: Optional[str] = None,
    evaluation_batch_explain_starting_replica_count: Optional[int] = None,
    evaluation_batch_explain_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    quantiles: Optional[List[float]] = None,
    enable_probabilistic_inference: bool = False,
    num_selected_features: Optional[int] = None,
    model_display_name: str = '',
    model_description: str = '',
) -> Dict[str, Any]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    stratified_split_key: The stratified_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    weight_column: The weight column name.
    study_spec_parameters_override: The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value: Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value: Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override: The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops: Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type: The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers: The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb: Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.
    additional_experiments: Use this field to config private preview features.
    dataflow_service_account: Custom service account to run dataflow jobs.
    max_selected_features: number of features to select for training,
    apply_feature_selection_tuning: tuning feature selection rate if true.
    run_evaluation: Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_batch_explain_machine_type: The prediction server machine type
      for batch explain components during evaluation.
    evaluation_batch_explain_starting_replica_count: The initial number of
      prediction server for batch explain components during evaluation.
    evaluation_batch_explain_max_replica_count: The max number of prediction
      server for batch explain components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    run_distillation: Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type: The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count: The max number of prediction server
      for batch predict component in the model distillation.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS
      URI.
    quantiles: Quantiles to use for probabilistic inference. Up to 5 quantiles
      are allowed of values between 0 and 1, exclusive. Represents the quantiles
      to use for that objective. Quantiles must be unique.
    enable_probabilistic_inference: If probabilistic inference is enabled, the
      model will fit a distribution that captures the uncertainty of a
      prediction. At inference time, the predictive distribution is used to make
      a point prediction that minimizes the optimization objective. For example,
      the mean of a predictive distribution is the point prediction that
      minimizes RMSE loss. If quantiles are specified, then the quantiles of the
      distribution are also returned.
    num_selected_features: Number of selected features for feature selection,
      defaults to None, in which case all features are used. If specified,
      enable_probabilistic_inference and run_distillation cannot be enabled.
    model_display_name: The display name of the uploaded Vertex model.
    model_description: The description for the uploaded model.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if not study_spec_parameters_override:
    study_spec_parameters_override = []
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []
  if not cv_trainer_worker_pool_specs_override:
    cv_trainer_worker_pool_specs_override = []
  if not quantiles:
    quantiles = []

  parameter_values = {}
  parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'prediction_type': prediction_type,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'optimization_objective': optimization_objective,
      'train_budget_milli_node_hours': train_budget_milli_node_hours,
      'stage_1_num_parallel_trials': stage_1_num_parallel_trials,
      'stage_2_num_parallel_trials': stage_2_num_parallel_trials,
      'stage_2_num_selected_trials': stage_2_num_selected_trials,
      'weight_column': weight_column,
      'optimization_objective_recall_value': (
          optimization_objective_recall_value
      ),
      'optimization_objective_precision_value': (
          optimization_objective_precision_value
      ),
      'study_spec_parameters_override': study_spec_parameters_override,
      'stage_1_tuner_worker_pool_specs_override': (
          stage_1_tuner_worker_pool_specs_override
      ),
      'cv_trainer_worker_pool_specs_override': (
          cv_trainer_worker_pool_specs_override
      ),
      'export_additional_model_without_custom_ops': (
          export_additional_model_without_custom_ops
      ),
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'dataflow_service_account': dataflow_service_account,
      'encryption_spec_key_name': encryption_spec_key_name,
      'max_selected_features': max_selected_features,
      'stage_1_tuning_result_artifact_uri': stage_1_tuning_result_artifact_uri,
      'quantiles': quantiles,
      'enable_probabilistic_inference': enable_probabilistic_inference,
      'model_display_name': model_display_name,
      'model_description': model_description,
  }
  parameter_values.update(
      {param: value for param, value in parameters.items() if value is not None}
  )

  if run_evaluation:
    eval_parameters = {
        'evaluation_batch_predict_machine_type': (
            evaluation_batch_predict_machine_type
        ),
        'evaluation_batch_predict_starting_replica_count': (
            evaluation_batch_predict_starting_replica_count
        ),
        'evaluation_batch_predict_max_replica_count': (
            evaluation_batch_predict_max_replica_count
        ),
        'evaluation_batch_explain_machine_type': (
            evaluation_batch_explain_machine_type
        ),
        'evaluation_batch_explain_starting_replica_count': (
            evaluation_batch_explain_starting_replica_count
        ),
        'evaluation_batch_explain_max_replica_count': (
            evaluation_batch_explain_max_replica_count
        ),
        'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
        'evaluation_dataflow_starting_num_workers': (
            evaluation_dataflow_starting_num_workers
        ),
        'evaluation_dataflow_max_num_workers': (
            evaluation_dataflow_max_num_workers
        ),
        'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
        'run_evaluation': run_evaluation,
    }
    parameter_values.update(
        {
            param: value
            for param, value in eval_parameters.items()
            if value is not None
        }
    )

  # V1 pipeline without FTE
  if num_selected_features is None:
    if not additional_experiments:
      additional_experiments = {}

    parameters = {
        'transformations': transformations,
        'stats_and_example_gen_dataflow_machine_type': (
            stats_and_example_gen_dataflow_machine_type
        ),
        'stats_and_example_gen_dataflow_max_num_workers': (
            stats_and_example_gen_dataflow_max_num_workers
        ),
        'stats_and_example_gen_dataflow_disk_size_gb': (
            stats_and_example_gen_dataflow_disk_size_gb
        ),
        'transform_dataflow_machine_type': transform_dataflow_machine_type,
        'transform_dataflow_max_num_workers': (
            transform_dataflow_max_num_workers
        ),
        'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
        'additional_experiments': additional_experiments,
    }
    parameter_values.update(
        {
            param: value
            for param, value in parameters.items()
            if value is not None
        }
    )

    if apply_feature_selection_tuning:
      parameter_values.update({
          'apply_feature_selection_tuning': apply_feature_selection_tuning,
      })

    if run_distillation:
      distillation_parameters = {
          'distill_batch_predict_machine_type': (
              distill_batch_predict_machine_type
          ),
          'distill_batch_predict_starting_replica_count': (
              distill_batch_predict_starting_replica_count
          ),
          'distill_batch_predict_max_replica_count': (
              distill_batch_predict_max_replica_count
          ),
          'run_distillation': run_distillation,
      }
      parameter_values.update(
          {
              param: value
              for param, value in distillation_parameters.items()
              if value is not None
          }
      )

  # V2 pipeline (with FTE)
  else:
    if run_distillation:
      raise ValueError(
          'Distillation is currently not supported'
          ' when num_selected_features is specified.'
      )

    parameters = {
        'num_selected_features': num_selected_features,
        'dataset_level_custom_transformation_definitions': [],
        'dataset_level_transformations': [],
        'tf_auto_transform_features': {},
        'tf_custom_transformation_definitions': [],
        'legacy_transformations_path': transformations,
        'feature_transform_engine_dataflow_machine_type': (
            transform_dataflow_machine_type
        ),
        'feature_transform_engine_dataflow_max_num_workers': (
            transform_dataflow_max_num_workers
        ),
        'feature_transform_engine_dataflow_disk_size_gb': (
            transform_dataflow_disk_size_gb
        ),
    }
    parameter_values.update(
        {
            param: value
            for param, value in parameters.items()
            if value is not None
        }
    )

  return parameter_values


def get_automl_tabular_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    stage_2_num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    optimization_objective_recall_value: Optional[float] = None,
    optimization_objective_precision_value: Optional[float] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: Optional[str] = None,
    stats_and_example_gen_dataflow_max_num_workers: Optional[int] = None,
    stats_and_example_gen_dataflow_disk_size_gb: Optional[int] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: Optional[str] = None,
    additional_experiments: Optional[Dict[str, Any]] = None,
    dataflow_service_account: Optional[str] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_batch_explain_machine_type: Optional[str] = None,
    evaluation_batch_explain_starting_replica_count: Optional[int] = None,
    evaluation_batch_explain_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None,
    stage_1_tuning_result_artifact_uri: Optional[str] = None,
    quantiles: Optional[List[float]] = None,
    enable_probabilistic_inference: bool = False,
    num_selected_features: Optional[int] = None,
    model_display_name: str = '',
    model_description: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    stratified_split_key: The stratified_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    weight_column: The weight column name.
    study_spec_parameters_override: The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value: Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value: Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override: The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops: Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type: The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers: The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb: Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.
    additional_experiments: Use this field to config private preview features.
    dataflow_service_account: Custom service account to run dataflow jobs.
    run_evaluation: Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_batch_explain_machine_type: The prediction server machine type
      for batch explain components during evaluation.
    evaluation_batch_explain_starting_replica_count: The initial number of
      prediction server for batch explain components during evaluation.
    evaluation_batch_explain_max_replica_count: The max number of prediction
      server for batch explain components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    run_distillation: Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type: The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count: The max number of prediction server
      for batch predict component in the model distillation.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS
      URI.
    quantiles: Quantiles to use for probabilistic inference. Up to 5 quantiles
      are allowed of values between 0 and 1, exclusive. Represents the quantiles
      to use for that objective. Quantiles must be unique.
    enable_probabilistic_inference: If probabilistic inference is enabled, the
      model will fit a distribution that captures the uncertainty of a
      prediction. At inference time, the predictive distribution is used to make
      a point prediction that minimizes the optimization objective. For example,
      the mean of a predictive distribution is the point prediction that
      minimizes RMSE loss. If quantiles are specified, then the quantiles of the
      distribution are also returned.
    num_selected_features: Number of selected features for feature selection,
      defaults to None, in which case all features are used.
    model_display_name: The display name of the uploaded Vertex model.
    model_description: The description for the uploaded model.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  parameter_values = _get_default_pipeline_params(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      prediction_type=prediction_type,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      stage_2_num_selected_trials=stage_2_num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      timestamp_split_key=timestamp_split_key,
      stratified_split_key=stratified_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      study_spec_parameters_override=study_spec_parameters_override,
      optimization_objective_recall_value=optimization_objective_recall_value,
      optimization_objective_precision_value=optimization_objective_precision_value,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      cv_trainer_worker_pool_specs_override=cv_trainer_worker_pool_specs_override,
      export_additional_model_without_custom_ops=export_additional_model_without_custom_ops,
      stats_and_example_gen_dataflow_machine_type=stats_and_example_gen_dataflow_machine_type,
      stats_and_example_gen_dataflow_max_num_workers=stats_and_example_gen_dataflow_max_num_workers,
      stats_and_example_gen_dataflow_disk_size_gb=stats_and_example_gen_dataflow_disk_size_gb,
      transform_dataflow_machine_type=transform_dataflow_machine_type,
      transform_dataflow_max_num_workers=transform_dataflow_max_num_workers,
      transform_dataflow_disk_size_gb=transform_dataflow_disk_size_gb,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
      additional_experiments=additional_experiments,
      dataflow_service_account=dataflow_service_account,
      run_evaluation=run_evaluation,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_batch_explain_machine_type=evaluation_batch_explain_machine_type,
      evaluation_batch_explain_starting_replica_count=evaluation_batch_explain_starting_replica_count,
      evaluation_batch_explain_max_replica_count=evaluation_batch_explain_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      run_distillation=run_distillation,
      distill_batch_predict_machine_type=distill_batch_predict_machine_type,
      distill_batch_predict_starting_replica_count=distill_batch_predict_starting_replica_count,
      distill_batch_predict_max_replica_count=distill_batch_predict_max_replica_count,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      quantiles=quantiles,
      enable_probabilistic_inference=enable_probabilistic_inference,
      num_selected_features=num_selected_features,
      model_display_name=model_display_name,
      model_description=model_description,
  )

  # V1 pipeline without FTE
  if num_selected_features is None:
    pipeline_definition_path = os.path.join(
        _GCPC_GA_TABULAR_PATH, 'automl_tabular_pipeline.yaml'
    )

  # V2 pipeline with FTE
  else:
    pipeline_definition_path = os.path.join(
        pathlib.Path(__file__).parent.resolve(),
        'automl_tabular_v2_pipeline.yaml',
    )

  return pipeline_definition_path, parameter_values


def get_automl_tabular_feature_selection_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: Optional[int] = None,
    stage_2_num_parallel_trials: Optional[int] = None,
    stage_2_num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    optimization_objective_recall_value: Optional[float] = None,
    optimization_objective_precision_value: Optional[float] = None,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: Optional[str] = None,
    stats_and_example_gen_dataflow_max_num_workers: Optional[int] = None,
    stats_and_example_gen_dataflow_disk_size_gb: Optional[int] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: Optional[str] = None,
    additional_experiments: Optional[Dict[str, Any]] = None,
    dataflow_service_account: Optional[str] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_batch_explain_machine_type: Optional[str] = None,
    evaluation_batch_explain_starting_replica_count: Optional[int] = None,
    evaluation_batch_explain_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    max_selected_features: int = 1000,
    apply_feature_selection_tuning: bool = False,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None,
    model_display_name: str = '',
    model_description: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    stratified_split_key: The stratified_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    weight_column: The weight column name.
    study_spec_parameters_override: The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value: Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value: Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override: The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override: The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops: Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type: The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers: The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb: Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.
    additional_experiments: Use this field to config private preview features.
    dataflow_service_account: Custom service account to run dataflow jobs.
    run_evaluation: Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_batch_explain_machine_type: The prediction server machine type
      for batch explain components during evaluation.
    evaluation_batch_explain_starting_replica_count: The initial number of
      prediction server for batch explain components during evaluation.
    evaluation_batch_explain_max_replica_count: The max number of prediction
      server for batch explain components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    max_selected_features: number of features to select for training,
    apply_feature_selection_tuning: tuning feature selection rate if true.
    run_distillation: Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type: The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count: The max number of prediction server
      for batch predict component in the model distillation.
    model_display_name: The display name of the uploaded Vertex model.
    model_description: The description for the uploaded model.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  model_display_name = (
      model_display_name
      if model_display_name
      else _generate_model_display_name()
  )

  parameter_values = _get_default_pipeline_params(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      prediction_type=prediction_type,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      stage_2_num_selected_trials=stage_2_num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      timestamp_split_key=timestamp_split_key,
      stratified_split_key=stratified_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      study_spec_parameters_override=study_spec_parameters_override,
      optimization_objective_recall_value=optimization_objective_recall_value,
      optimization_objective_precision_value=optimization_objective_precision_value,
      stage_1_tuner_worker_pool_specs_override=stage_1_tuner_worker_pool_specs_override,
      cv_trainer_worker_pool_specs_override=cv_trainer_worker_pool_specs_override,
      export_additional_model_without_custom_ops=export_additional_model_without_custom_ops,
      stats_and_example_gen_dataflow_machine_type=stats_and_example_gen_dataflow_machine_type,
      stats_and_example_gen_dataflow_max_num_workers=stats_and_example_gen_dataflow_max_num_workers,
      stats_and_example_gen_dataflow_disk_size_gb=stats_and_example_gen_dataflow_disk_size_gb,
      transform_dataflow_machine_type=transform_dataflow_machine_type,
      transform_dataflow_max_num_workers=transform_dataflow_max_num_workers,
      transform_dataflow_disk_size_gb=transform_dataflow_disk_size_gb,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
      additional_experiments=additional_experiments,
      dataflow_service_account=dataflow_service_account,
      max_selected_features=max_selected_features,
      apply_feature_selection_tuning=apply_feature_selection_tuning,
      run_evaluation=run_evaluation,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_batch_explain_machine_type=evaluation_batch_explain_machine_type,
      evaluation_batch_explain_starting_replica_count=evaluation_batch_explain_starting_replica_count,
      evaluation_batch_explain_max_replica_count=evaluation_batch_explain_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      run_distillation=run_distillation,
      distill_batch_predict_machine_type=distill_batch_predict_machine_type,
      distill_batch_predict_starting_replica_count=distill_batch_predict_starting_replica_count,
      distill_batch_predict_max_replica_count=distill_batch_predict_max_replica_count,
      model_display_name=model_display_name,
      model_description=model_description,
  )

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'automl_tabular_feature_selection_pipeline.yaml',
  )
  return pipeline_definition_path, parameter_values


def input_dictionary_to_parameter(input_dict: Optional[Dict[str, Any]]) -> str:
  """Convert json input dict to encoded parameter string.

  This function is required due to the limitation on YAML component definition
  that YAML definition does not have a keyword for apply quote escape, so the
  JSON argument's quote must be manually escaped using this function.

  Args:
    input_dict: The input json dictionary.

  Returns:
    The encoded string used for parameter.
  """
  if not input_dict:
    return ''
  out = json.dumps(json.dumps(input_dict))
  return out[1:-1]  # remove the outside quotes, e.g., "foo" -> foo


def get_skip_architecture_search_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: str,
    train_budget_milli_node_hours: float,
    stage_1_tuning_result_artifact_uri: str,
    stage_2_num_parallel_trials: Optional[int] = None,
    stage_2_num_selected_trials: Optional[int] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: Optional[str] = None,
    optimization_objective_recall_value: Optional[float] = None,
    optimization_objective_precision_value: Optional[float] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: Optional[str] = None,
    stats_and_example_gen_dataflow_max_num_workers: Optional[int] = None,
    stats_and_example_gen_dataflow_disk_size_gb: Optional[int] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: Optional[str] = None,
    additional_experiments: Optional[Dict[str, Any]] = None,
    dataflow_service_account: Optional[str] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_batch_explain_machine_type: Optional[str] = None,
    evaluation_batch_explain_starting_replica_count: Optional[int] = None,
    evaluation_batch_explain_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips architecture search.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The transformations to apply.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS
      URI.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    predefined_split_key: The predefined_split column name.
    timestamp_split_key: The timestamp_split column name.
    stratified_split_key: The stratified_split column name.
    training_fraction: The training fraction.
    validation_fraction: The validation fraction.
    test_fraction: float = The test fraction.
    weight_column: The weight column name.
    optimization_objective_recall_value: Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value: Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    cv_trainer_worker_pool_specs_override: The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops: Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type: The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers: The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb: Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.
    additional_experiments: Use this field to config private preview features.
    dataflow_service_account: Custom service account to run dataflow jobs.
    run_evaluation: Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_batch_explain_machine_type: The prediction server machine type
      for batch explain components during evaluation.
    evaluation_batch_explain_starting_replica_count: The initial number of
      prediction server for batch explain components during evaluation.
    evaluation_batch_explain_max_replica_count: The max number of prediction
      server for batch explain components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """

  return get_automl_tabular_pipeline_and_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column=target_column,
      prediction_type=prediction_type,
      optimization_objective=optimization_objective,
      transformations=transformations,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      stage_1_num_parallel_trials=None,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      stage_2_num_selected_trials=stage_2_num_selected_trials,
      data_source_csv_filenames=data_source_csv_filenames,
      data_source_bigquery_table_path=data_source_bigquery_table_path,
      predefined_split_key=predefined_split_key,
      timestamp_split_key=timestamp_split_key,
      stratified_split_key=stratified_split_key,
      training_fraction=training_fraction,
      validation_fraction=validation_fraction,
      test_fraction=test_fraction,
      weight_column=weight_column,
      study_spec_parameters_override=[],
      optimization_objective_recall_value=optimization_objective_recall_value,
      optimization_objective_precision_value=optimization_objective_precision_value,
      stage_1_tuner_worker_pool_specs_override={},
      cv_trainer_worker_pool_specs_override=cv_trainer_worker_pool_specs_override,
      export_additional_model_without_custom_ops=export_additional_model_without_custom_ops,
      stats_and_example_gen_dataflow_machine_type=stats_and_example_gen_dataflow_machine_type,
      stats_and_example_gen_dataflow_max_num_workers=stats_and_example_gen_dataflow_max_num_workers,
      stats_and_example_gen_dataflow_disk_size_gb=stats_and_example_gen_dataflow_disk_size_gb,
      transform_dataflow_machine_type=transform_dataflow_machine_type,
      transform_dataflow_max_num_workers=transform_dataflow_max_num_workers,
      transform_dataflow_disk_size_gb=transform_dataflow_disk_size_gb,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
      additional_experiments=additional_experiments,
      dataflow_service_account=dataflow_service_account,
      run_evaluation=run_evaluation,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_batch_explain_machine_type=evaluation_batch_explain_machine_type,
      evaluation_batch_explain_starting_replica_count=evaluation_batch_explain_starting_replica_count,
      evaluation_batch_explain_max_replica_count=evaluation_batch_explain_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      run_distillation=None,
      distill_batch_predict_machine_type=None,
      distill_batch_predict_starting_replica_count=None,
      distill_batch_predict_max_replica_count=None,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      quantiles=[],
      enable_probabilistic_inference=False,
  )


def get_wide_and_deep_trainer_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    learning_rate: float,
    dnn_learning_rate: float,
    transform_config: Optional[str] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: bool = False,
    feature_selection_algorithm: Optional[str] = None,
    materialized_examples_format: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_transform_execution_engine: Optional[str] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    optimizer_type: str = 'adam',
    max_steps: int = -1,
    max_train_secs: int = -1,
    l1_regularization_strength: float = 0,
    l2_regularization_strength: float = 0,
    l2_shrinkage_regularization_strength: float = 0,
    beta_1: float = 0.9,
    beta_2: float = 0.999,
    hidden_units: str = '30,30,30',
    use_wide: bool = True,
    embed_categories: bool = True,
    dnn_dropout: float = 0,
    dnn_optimizer_type: str = 'adam',
    dnn_l1_regularization_strength: float = 0,
    dnn_l2_regularization_strength: float = 0,
    dnn_l2_shrinkage_regularization_strength: float = 0,
    dnn_beta_1: float = 0.9,
    dnn_beta_2: float = 0.999,
    enable_profiler: bool = False,
    cache_data: str = 'auto',
    seed: int = 1,
    eval_steps: int = 0,
    batch_size: int = 100,
    measurement_selection_type: Optional[str] = None,
    optimization_metric: Optional[str] = None,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: str = '',
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_starting_num_workers: int = _EVALUATION_DATAFLOW_STARTING_NUM_WORKERS,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the Wide & Deep training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      'classification' or 'regression'.
    learning_rate: The learning rate used by the linear optimizer.
    dnn_learning_rate: The learning rate for training the deep part of the
      model.
    transform_config: Path to v1 TF transformation configuration.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    materialized_examples_format: The format for the materialized examples.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_transform_execution_engine: The execution engine used to execute TF-based
      transformations.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    optimizer_type: The type of optimizer to use. Choices are "adam", "ftrl" and
      "sgd" for the Adam, FTRL, and Gradient Descent Optimizers, respectively.
    max_steps: Number of steps to run the trainer for.
    max_train_secs: Amount of time in seconds to run the trainer for.
    l1_regularization_strength: L1 regularization strength for
      optimizer_type="ftrl".
    l2_regularization_strength: L2 regularization strength for
      optimizer_type="ftrl".
    l2_shrinkage_regularization_strength: L2 shrinkage regularization strength
      for optimizer_type="ftrl".
    beta_1: Beta 1 value for optimizer_type="adam".
    beta_2: Beta 2 value for optimizer_type="adam".
    hidden_units: Hidden layer sizes to use for DNN feature columns, provided in
      comma-separated layers.
    use_wide: If set to true, the categorical columns will be used in the wide
      part of the DNN model.
    embed_categories: If set to true, the categorical columns will be used
      embedded and used in the deep part of the model. Embedding size is the
      square root of the column cardinality.
    dnn_dropout: The probability we will drop out a given coordinate.
    dnn_optimizer_type: The type of optimizer to use for the deep part of the
      model. Choices are "adam", "ftrl" and "sgd". for the Adam, FTRL, and
      Gradient Descent Optimizers, respectively.
    dnn_l1_regularization_strength: L1 regularization strength for
      dnn_optimizer_type="ftrl".
    dnn_l2_regularization_strength: L2 regularization strength for
      dnn_optimizer_type="ftrl".
    dnn_l2_shrinkage_regularization_strength: L2 shrinkage regularization
      strength for dnn_optimizer_type="ftrl".
    dnn_beta_1: Beta 1 value for dnn_optimizer_type="adam".
    dnn_beta_2: Beta 2 value for dnn_optimizer_type="adam".
    enable_profiler: Enables profiling and saves a trace during evaluation.
    cache_data: Whether to cache data or not. If set to 'auto', caching is
      determined based on the dataset size.
    seed: Seed to be used for this run.
    eval_steps: Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    batch_size: Batch size for training.
    measurement_selection_type: Which measurement to use if/when the service
      automatically selects the final measurement from previously reported
      intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    optimization_metric: Optimization metric used for
      `measurement_selection_type`. Default is "rmse" for regression and "auc"
      for classification.
    eval_frequency_secs: Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override: The dictionary for overriding training and
      evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  if transform_config and tf_transformations_path:
    raise ValueError(
        'Only one of transform_config and tf_transformations_path can '
        'be specified.'
    )

  elif transform_config:
    warnings.warn(
        'transform_config parameter is deprecated. '
        'Please use the flattened transform config arguments instead.'
    )
    tf_transformations_path = transform_config

  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {}
  training_and_eval_parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'prediction_type': prediction_type,
      'learning_rate': learning_rate,
      'dnn_learning_rate': dnn_learning_rate,
      'optimizer_type': optimizer_type,
      'max_steps': max_steps,
      'max_train_secs': max_train_secs,
      'l1_regularization_strength': l1_regularization_strength,
      'l2_regularization_strength': l2_regularization_strength,
      'l2_shrinkage_regularization_strength': (
          l2_shrinkage_regularization_strength
      ),
      'beta_1': beta_1,
      'beta_2': beta_2,
      'hidden_units': hidden_units,
      'use_wide': use_wide,
      'embed_categories': embed_categories,
      'dnn_dropout': dnn_dropout,
      'dnn_optimizer_type': dnn_optimizer_type,
      'dnn_l1_regularization_strength': dnn_l1_regularization_strength,
      'dnn_l2_regularization_strength': dnn_l2_regularization_strength,
      'dnn_l2_shrinkage_regularization_strength': (
          dnn_l2_shrinkage_regularization_strength
      ),
      'dnn_beta_1': dnn_beta_1,
      'dnn_beta_2': dnn_beta_2,
      'enable_profiler': enable_profiler,
      'cache_data': cache_data,
      'seed': seed,
      'eval_steps': eval_steps,
      'batch_size': batch_size,
      'measurement_selection_type': measurement_selection_type,
      'optimization_metric': optimization_metric,
      'eval_frequency_secs': eval_frequency_secs,
      'weight_column': weight_column,
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'worker_pool_specs_override': worker_pool_specs_override,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }
  _update_parameters(parameter_values, training_and_eval_parameters)

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
      'materialized_examples_format': (
          materialized_examples_format
          if materialized_examples_format
          else 'tfrecords_gzip'
      ),
      'tf_transform_execution_engine': (
          tf_transform_execution_engine
          if tf_transform_execution_engine
          else 'dataflow'
      ),
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'wide_and_deep_trainer_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: List[Dict[str, Any]],
    max_trial_count: int,
    parallel_trial_count: int,
    algorithm: str,
    enable_profiler: bool = False,
    seed: int = 1,
    eval_steps: int = 0,
    eval_frequency_secs: int = 600,
    transform_config: Optional[str] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_transform_execution_engine: Optional[str] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_starting_num_workers: int = _EVALUATION_DATAFLOW_STARTING_NUM_WORKERS,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the built-in algorithm HyperparameterTuningJob pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    study_spec_metric_id: Metric to optimize, possible values: [ 'loss',
      'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc', 'precision',
      'recall'].
    study_spec_metric_goal: Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override: List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count: The desired total number of trials.
    parallel_trial_count: The desired number of trials to run in parallel.
    algorithm: Algorithm to train. One of "tabnet" and "wide_and_deep".
    enable_profiler: Enables profiling and saves a trace during evaluation.
    seed: Seed to be used for this run.
    eval_steps: Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs: Frequency at which evaluation and checkpointing will
      take place.
    transform_config: Path to v1 TF transformation configuration.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_transform_execution_engine: The execution engine used to execute TF-based
      transformations.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    max_failed_trial_count: The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm: The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type: Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override: The dictionary for overriding training and
      evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  warnings.warn(
      'This method is deprecated. Please use'
      ' get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters or'
      ' get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters'
      ' instead.'
  )

  if algorithm == 'tabnet':
    return get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(
        project=project,
        location=location,
        root_dir=root_dir,
        target_column=target_column,
        prediction_type=prediction_type,
        study_spec_metric_id=study_spec_metric_id,
        study_spec_metric_goal=study_spec_metric_goal,
        study_spec_parameters_override=study_spec_parameters_override,
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        transform_config=transform_config,
        dataset_level_custom_transformation_definitions=dataset_level_custom_transformation_definitions,
        dataset_level_transformations=dataset_level_transformations,
        predefined_split_key=predefined_split_key,
        stratified_split_key=stratified_split_key,
        training_fraction=training_fraction,
        validation_fraction=validation_fraction,
        test_fraction=test_fraction,
        tf_transform_execution_engine=tf_transform_execution_engine,
        tf_auto_transform_features=tf_auto_transform_features,
        tf_custom_transformation_definitions=tf_custom_transformation_definitions,
        tf_transformations_path=tf_transformations_path,
        enable_profiler=enable_profiler,
        seed=seed,
        eval_steps=eval_steps,
        eval_frequency_secs=eval_frequency_secs,
        data_source_csv_filenames=data_source_csv_filenames,
        data_source_bigquery_table_path=data_source_bigquery_table_path,
        bigquery_staging_full_dataset_id=bigquery_staging_full_dataset_id,
        weight_column=weight_column,
        max_failed_trial_count=max_failed_trial_count,
        study_spec_algorithm=study_spec_algorithm,
        study_spec_measurement_selection_type=study_spec_measurement_selection_type,
        transform_dataflow_machine_type=transform_dataflow_machine_type,
        transform_dataflow_max_num_workers=transform_dataflow_max_num_workers,
        transform_dataflow_disk_size_gb=transform_dataflow_disk_size_gb,
        worker_pool_specs_override=worker_pool_specs_override,
        run_evaluation=run_evaluation,
        evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
        evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
        evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
        evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
        evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
        evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
        evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
    )
  elif algorithm == 'wide_and_deep':
    return get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
        project=project,
        location=location,
        root_dir=root_dir,
        target_column=target_column,
        prediction_type=prediction_type,
        study_spec_metric_id=study_spec_metric_id,
        study_spec_metric_goal=study_spec_metric_goal,
        study_spec_parameters_override=study_spec_parameters_override,
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        transform_config=transform_config,
        dataset_level_custom_transformation_definitions=dataset_level_custom_transformation_definitions,
        dataset_level_transformations=dataset_level_transformations,
        predefined_split_key=predefined_split_key,
        stratified_split_key=stratified_split_key,
        training_fraction=training_fraction,
        validation_fraction=validation_fraction,
        test_fraction=test_fraction,
        tf_transform_execution_engine=tf_transform_execution_engine,
        tf_auto_transform_features=tf_auto_transform_features,
        tf_custom_transformation_definitions=tf_custom_transformation_definitions,
        tf_transformations_path=tf_transformations_path,
        enable_profiler=enable_profiler,
        seed=seed,
        eval_steps=eval_steps,
        eval_frequency_secs=eval_frequency_secs,
        data_source_csv_filenames=data_source_csv_filenames,
        data_source_bigquery_table_path=data_source_bigquery_table_path,
        bigquery_staging_full_dataset_id=bigquery_staging_full_dataset_id,
        weight_column=weight_column,
        max_failed_trial_count=max_failed_trial_count,
        study_spec_algorithm=study_spec_algorithm,
        study_spec_measurement_selection_type=study_spec_measurement_selection_type,
        transform_dataflow_machine_type=transform_dataflow_machine_type,
        transform_dataflow_max_num_workers=transform_dataflow_max_num_workers,
        transform_dataflow_disk_size_gb=transform_dataflow_disk_size_gb,
        worker_pool_specs_override=worker_pool_specs_override,
        run_evaluation=run_evaluation,
        evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
        evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
        evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
        evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
        evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
        evaluation_dataflow_starting_num_workers=evaluation_dataflow_starting_num_workers,
        evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
    )
  else:
    raise ValueError(
        'Invalid algorithm provided. Supported values are "tabnet" and'
        ' "wide_and_deep".'
    )


def get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: List[Dict[str, Any]],
    max_trial_count: int,
    parallel_trial_count: int,
    transform_config: Optional[str] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: bool = False,
    feature_selection_algorithm: Optional[str] = None,
    materialized_examples_format: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_transform_execution_engine: Optional[str] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    enable_profiler: bool = False,
    cache_data: str = 'auto',
    seed: int = 1,
    eval_steps: int = 0,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_starting_num_workers: int = _EVALUATION_DATAFLOW_STARTING_NUM_WORKERS,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the TabNet HyperparameterTuningJob pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    study_spec_metric_id: Metric to optimize, possible values: [ 'loss',
      'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc', 'precision',
      'recall'].
    study_spec_metric_goal: Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override: List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count: The desired total number of trials.
    parallel_trial_count: The desired number of trials to run in parallel.
    transform_config: Path to v1 TF transformation configuration.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    materialized_examples_format: The format for the materialized examples.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_transform_execution_engine: The execution engine used to execute TF-based
      transformations.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    enable_profiler: Enables profiling and saves a trace during evaluation.
    cache_data: Whether to cache data or not. If set to 'auto', caching is
      determined based on the dataset size.
    seed: Seed to be used for this run.
    eval_steps: Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs: Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    max_failed_trial_count: The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm: The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type: Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override: The dictionary for overriding training and
      evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  if transform_config and tf_transformations_path:
    raise ValueError(
        'Only one of transform_config and tf_transformations_path can '
        'be specified.'
    )

  elif transform_config:
    warnings.warn(
        'transform_config parameter is deprecated. '
        'Please use the flattened transform config arguments instead.'
    )
    tf_transformations_path = transform_config

  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'prediction_type': prediction_type,
      'study_spec_metric_id': study_spec_metric_id,
      'study_spec_metric_goal': study_spec_metric_goal,
      'study_spec_parameters_override': study_spec_parameters_override,
      'max_trial_count': max_trial_count,
      'parallel_trial_count': parallel_trial_count,
      'enable_profiler': enable_profiler,
      'cache_data': cache_data,
      'seed': seed,
      'eval_steps': eval_steps,
      'eval_frequency_secs': eval_frequency_secs,
      'weight_column': weight_column,
      'max_failed_trial_count': max_failed_trial_count,
      'study_spec_algorithm': study_spec_algorithm,
      'study_spec_measurement_selection_type': (
          study_spec_measurement_selection_type
      ),
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'worker_pool_specs_override': worker_pool_specs_override,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
      'materialized_examples_format': (
          materialized_examples_format
          if materialized_examples_format
          else 'tfrecords_gzip'
      ),
      'tf_transform_execution_engine': (
          tf_transform_execution_engine
          if tf_transform_execution_engine
          else 'dataflow'
      ),
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'tabnet_hyperparameter_tuning_job_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: List[Dict[str, Any]],
    max_trial_count: int,
    parallel_trial_count: int,
    transform_config: Optional[str] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: bool = False,
    feature_selection_algorithm: Optional[str] = None,
    materialized_examples_format: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_transform_execution_engine: Optional[str] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    enable_profiler: bool = False,
    cache_data: str = 'auto',
    seed: int = 1,
    eval_steps: int = 0,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_starting_num_workers: int = _EVALUATION_DATAFLOW_STARTING_NUM_WORKERS,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the Wide & Deep algorithm HyperparameterTuningJob pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    study_spec_metric_id: Metric to optimize, possible values: [ 'loss',
      'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc', 'precision',
      'recall'].
    study_spec_metric_goal: Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override: List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count: The desired total number of trials.
    parallel_trial_count: The desired number of trials to run in parallel.
    transform_config: Path to v1 TF transformation configuration.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    materialized_examples_format: The format for the materialized examples.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_transform_execution_engine: The execution engine used to execute TF-based
      transformations.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    enable_profiler: Enables profiling and saves a trace during evaluation.
    cache_data: Whether to cache data or not. If set to 'auto', caching is
      determined based on the dataset size.
    seed: Seed to be used for this run.
    eval_steps: Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs: Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    max_failed_trial_count: The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm: The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type: Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override: The dictionary for overriding training and
      evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  if transform_config and tf_transformations_path:
    raise ValueError(
        'Only one of transform_config and tf_transformations_path can '
        'be specified.'
    )

  elif transform_config:
    warnings.warn(
        'transform_config parameter is deprecated. '
        'Please use the flattened transform config arguments instead.'
    )
    tf_transformations_path = transform_config

  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'prediction_type': prediction_type,
      'study_spec_metric_id': study_spec_metric_id,
      'study_spec_metric_goal': study_spec_metric_goal,
      'study_spec_parameters_override': study_spec_parameters_override,
      'max_trial_count': max_trial_count,
      'parallel_trial_count': parallel_trial_count,
      'enable_profiler': enable_profiler,
      'cache_data': cache_data,
      'seed': seed,
      'eval_steps': eval_steps,
      'eval_frequency_secs': eval_frequency_secs,
      'weight_column': weight_column,
      'max_failed_trial_count': max_failed_trial_count,
      'study_spec_algorithm': study_spec_algorithm,
      'study_spec_measurement_selection_type': (
          study_spec_measurement_selection_type
      ),
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'worker_pool_specs_override': worker_pool_specs_override,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
      'materialized_examples_format': (
          materialized_examples_format
          if materialized_examples_format
          else 'tfrecords_gzip'
      ),
      'tf_transform_execution_engine': (
          tf_transform_execution_engine
          if tf_transform_execution_engine
          else 'dataflow'
      ),
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'wide_and_deep_hyperparameter_tuning_job_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values


def get_tabnet_trainer_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    learning_rate: float,
    transform_config: Optional[str] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: bool = False,
    feature_selection_algorithm: Optional[str] = None,
    materialized_examples_format: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_transform_execution_engine: Optional[str] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    max_steps: int = -1,
    max_train_secs: int = -1,
    large_category_dim: int = 1,
    large_category_thresh: int = 300,
    yeo_johnson_transform: bool = True,
    feature_dim: int = 64,
    feature_dim_ratio: float = 0.5,
    num_decision_steps: int = 6,
    relaxation_factor: float = 1.5,
    decay_every: float = 100,
    decay_rate: float = 0.95,
    gradient_thresh: float = 2000,
    sparsity_loss_weight: float = 0.00001,
    batch_momentum: float = 0.95,
    batch_size_ratio: float = 0.25,
    num_transformer_layers: int = 4,
    num_transformer_layers_ratio: float = 0.25,
    class_weight: float = 1.0,
    loss_function_type: str = 'default',
    alpha_focal_loss: float = 0.25,
    gamma_focal_loss: float = 2.0,
    enable_profiler: bool = False,
    cache_data: str = 'auto',
    seed: int = 1,
    eval_steps: int = 0,
    batch_size: int = 100,
    measurement_selection_type: Optional[str] = None,
    optimization_metric: Optional[str] = None,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: str = '',
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_starting_num_workers: int = _EVALUATION_DATAFLOW_STARTING_NUM_WORKERS,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
) -> Tuple[str, Dict[str, Any]]:
  """Get the TabNet training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    learning_rate: The learning rate used by the linear optimizer.
    transform_config: Path to v1 TF transformation configuration.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    materialized_examples_format: The format for the materialized examples.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_transform_execution_engine: The execution engine used to execute TF-based
      transformations.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    max_steps: Number of steps to run the trainer for.
    max_train_secs: Amount of time in seconds to run the trainer for.
    large_category_dim: Embedding dimension for categorical feature with large
      number of categories.
    large_category_thresh: Threshold for number of categories to apply
      large_category_dim embedding dimension to.
    yeo_johnson_transform: Enables trainable Yeo-Johnson power transform.
    feature_dim: Dimensionality of the hidden representation in feature
      transformation block.
    feature_dim_ratio: The ratio of output dimension (dimensionality of the
      outputs of each decision step) to feature dimension.
    num_decision_steps: Number of sequential decision steps.
    relaxation_factor: Relaxation factor that promotes the reuse of each feature
      at different decision steps. When it is 1, a feature is enforced to be
      used only at one decision step and as it increases, more flexibility is
      provided to use a feature at multiple decision steps.
    decay_every: Number of iterations for periodically applying learning rate
      decaying.
    decay_rate: Learning rate decaying.
    gradient_thresh: Threshold for the norm of gradients for clipping.
    sparsity_loss_weight: Weight of the loss for sparsity regularization
      (increasing it will yield more sparse feature selection).
    batch_momentum: Momentum in ghost batch normalization.
    batch_size_ratio: The ratio of virtual batch size (size of the ghost batch
      normalization) to batch size.
    num_transformer_layers: The number of transformer layers for each decision
      step. used only at one decision step and as it increases, more flexibility
      is provided to use a feature at multiple decision steps.
    num_transformer_layers_ratio: The ratio of shared transformer layer to
      transformer layers.
    class_weight: The class weight is used to computes a weighted cross entropy
      which is helpful in classify imbalanced dataset. Only used for
      classification.
    loss_function_type: Loss function type. Loss function in classification
      [cross_entropy, weighted_cross_entropy, focal_loss], default is
      cross_entropy. Loss function in regression: [rmse, mae, mse], default is
      mse.
    alpha_focal_loss: Alpha value (balancing factor) in focal_loss function.
      Only used for classification.
    gamma_focal_loss: Gamma value (modulating factor) for focal loss for focal
      loss. Only used for classification.
    enable_profiler: Enables profiling and saves a trace during evaluation.
    cache_data: Whether to cache data or not. If set to 'auto', caching is
      determined based on the dataset size.
    seed: Seed to be used for this run.
    eval_steps: Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    batch_size: Batch size for training.
    measurement_selection_type: Which measurement to use if/when the service
      automatically selects the final measurement from previously reported
      intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    optimization_metric: Optimization metric used for
      `measurement_selection_type`. Default is "rmse" for regression and "auc"
      for classification.
    eval_frequency_secs: Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override: The dictionary for overriding training and
      evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  if transform_config and tf_transformations_path:
    raise ValueError(
        'Only one of transform_config and tf_transformations_path can '
        'be specified.'
    )

  elif transform_config:
    warnings.warn(
        'transform_config parameter is deprecated. '
        'Please use the flattened transform config arguments instead.'
    )
    tf_transformations_path = transform_config

  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {}
  training_and_eval_parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'prediction_type': prediction_type,
      'learning_rate': learning_rate,
      'max_steps': max_steps,
      'max_train_secs': max_train_secs,
      'large_category_dim': large_category_dim,
      'large_category_thresh': large_category_thresh,
      'yeo_johnson_transform': yeo_johnson_transform,
      'feature_dim': feature_dim,
      'feature_dim_ratio': feature_dim_ratio,
      'num_decision_steps': num_decision_steps,
      'relaxation_factor': relaxation_factor,
      'decay_every': decay_every,
      'decay_rate': decay_rate,
      'gradient_thresh': gradient_thresh,
      'sparsity_loss_weight': sparsity_loss_weight,
      'batch_momentum': batch_momentum,
      'batch_size_ratio': batch_size_ratio,
      'num_transformer_layers': num_transformer_layers,
      'num_transformer_layers_ratio': num_transformer_layers_ratio,
      'class_weight': class_weight,
      'loss_function_type': loss_function_type,
      'alpha_focal_loss': alpha_focal_loss,
      'gamma_focal_loss': gamma_focal_loss,
      'enable_profiler': enable_profiler,
      'cache_data': cache_data,
      'seed': seed,
      'eval_steps': eval_steps,
      'batch_size': batch_size,
      'measurement_selection_type': measurement_selection_type,
      'optimization_metric': optimization_metric,
      'eval_frequency_secs': eval_frequency_secs,
      'weight_column': weight_column,
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'worker_pool_specs_override': worker_pool_specs_override,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }
  _update_parameters(parameter_values, training_and_eval_parameters)

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
      'materialized_examples_format': (
          materialized_examples_format
          if materialized_examples_format
          else 'tfrecords_gzip'
      ),
      'tf_transform_execution_engine': (
          tf_transform_execution_engine
          if tf_transform_execution_engine
          else 'dataflow'
      ),
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'tabnet_trainer_pipeline.yaml'
  )

  return pipeline_definition_path, parameter_values


def get_tabnet_study_spec_parameters_override(
    dataset_size_bucket: str, prediction_type: str, training_budget_bucket: str
) -> List[Dict[str, Any]]:
  """Get study_spec_parameters_override for a TabNet hyperparameter tuning job.

  Args:
    dataset_size_bucket: Size of the dataset. One of "small" (< 1M rows),
      "medium" (1M - 100M rows), or "large" (> 100M rows).
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    training_budget_bucket: Bucket of the estimated training budget. One of
      "small" (< $600), "medium" ($600 - $2400), or "large" (> $2400). This
      parameter is only used as a hint for the hyperparameter search space,
      unrelated to the real cost.

  Returns:
    List of study_spec_parameters_override.
  """

  if dataset_size_bucket not in ['small', 'medium', 'large']:
    raise ValueError(
        'Invalid dataset_size_bucket provided. Supported values '
        ' are "small", "medium" or "large".'
    )
  if training_budget_bucket not in ['small', 'medium', 'large']:
    raise ValueError(
        'Invalid training_budget_bucket provided. Supported values '
        'are "small", "medium" or "large".'
    )

  param_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      f'configs/tabnet_params_{dataset_size_bucket}_data_{training_budget_bucket}_search_space.json',
  )
  with open(param_path, 'r') as f:
    param_content = f.read()
    params = json.loads(param_content)

  if prediction_type == 'regression':
    return _format_tabnet_regression_study_spec_parameters_override(
        params, training_budget_bucket
    )
  return params


def _format_tabnet_regression_study_spec_parameters_override(
    params: List[Dict[str, Any]], training_budget_bucket: str
) -> List[Dict[str, Any]]:
  """Get regression study_spec_parameters_override for a TabNet hyperparameter tuning job.

  Args:
    params: List of dictionaries representing parameters to optimize. The
      dictionary key is the parameter_id, which is passed to training job as a
      command line argument, and the dictionary value is the parameter
      specification of the metric.
    training_budget_bucket: Bucket of the estimated training budget. One of
      "small" (< $600), "medium" ($600 - $2400), or "large" (> $2400). This
      parameter is only used as a hint for the hyperparameter search space,
      unrelated to the real cost.

  Returns:
    List of study_spec_parameters_override for regression.
  """

  # To get regression study_spec_parameters, we need to set
  # `loss_function_type` to ‘mae’ (‘mae’ and ‘mse’ for "large" search space),
  # remove the `alpha_focal_loss`, `gamma_focal_loss`
  # and `class_weight` parameters and increase the max for
  # `sparsity_loss_weight` to 100.
  formatted_params = []
  for param in params:
    if param['parameter_id'] in [
        'alpha_focal_loss',
        'gamma_focal_loss',
        'class_weight',
    ]:
      continue
    elif param['parameter_id'] == 'sparsity_loss_weight':
      param['double_value_spec']['max_value'] = 100
    elif param['parameter_id'] == 'loss_function_type':
      if training_budget_bucket == 'large':
        param['categorical_value_spec']['values'] = ['mae', 'mse']
      else:
        param['categorical_value_spec']['values'] = ['mae']

    formatted_params.append(param)

  return formatted_params


def get_wide_and_deep_study_spec_parameters_override() -> List[Dict[str, Any]]:
  """Get study_spec_parameters_override for a Wide & Deep hyperparameter tuning job.

  Returns:
    List of study_spec_parameters_override.
  """
  param_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'configs/wide_and_deep_params.json',
  )
  with open(param_path, 'r') as f:
    param_content = f.read()
    params = json.loads(param_content)

  return params


def get_xgboost_study_spec_parameters_override() -> List[Dict[str, Any]]:
  """Get study_spec_parameters_override for an XGBoost hyperparameter tuning job.

  Returns:
    List of study_spec_parameters_override.
  """
  param_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'configs/xgboost_params.json'
  )
  with open(param_path, 'r') as f:
    param_content = f.read()
    params = json.loads(param_content)

  return params


def get_xgboost_trainer_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    objective: str,
    eval_metric: Optional[str] = None,
    num_boost_round: Optional[int] = None,
    early_stopping_rounds: Optional[int] = None,
    base_score: Optional[float] = None,
    disable_default_eval_metric: Optional[int] = None,
    seed: Optional[int] = None,
    seed_per_iteration: Optional[bool] = None,
    booster: Optional[str] = None,
    eta: Optional[float] = None,
    gamma: Optional[float] = None,
    max_depth: Optional[int] = None,
    min_child_weight: Optional[float] = None,
    max_delta_step: Optional[float] = None,
    subsample: Optional[float] = None,
    colsample_bytree: Optional[float] = None,
    colsample_bylevel: Optional[float] = None,
    colsample_bynode: Optional[float] = None,
    reg_lambda: Optional[float] = None,
    reg_alpha: Optional[float] = None,
    tree_method: Optional[str] = None,
    scale_pos_weight: Optional[float] = None,
    updater: Optional[str] = None,
    refresh_leaf: Optional[int] = None,
    process_type: Optional[str] = None,
    grow_policy: Optional[str] = None,
    sampling_method: Optional[str] = None,
    monotone_constraints: Optional[str] = None,
    interaction_constraints: Optional[str] = None,
    sample_type: Optional[str] = None,
    normalize_type: Optional[str] = None,
    rate_drop: Optional[float] = None,
    one_drop: Optional[int] = None,
    skip_drop: Optional[float] = None,
    num_parallel_tree: Optional[int] = None,
    feature_selector: Optional[str] = None,
    top_k: Optional[int] = None,
    max_cat_to_onehot: Optional[int] = None,
    max_leaves: Optional[int] = None,
    max_bin: Optional[int] = None,
    tweedie_variance_power: Optional[float] = None,
    huber_slope: Optional[float] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: Optional[bool] = None,
    feature_selection_algorithm: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: Optional[str] = None,
    training_machine_type: Optional[str] = None,
    training_total_replica_count: Optional[int] = None,
    training_accelerator_type: Optional[str] = None,
    training_accelerator_count: Optional[int] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    run_evaluation: Optional[bool] = None,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: Optional[bool] = None,
    encryption_spec_key_name: Optional[str] = None,
):
  """Get the XGBoost training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    objective: Specifies the learning task and the learning objective. Must be
      one of [reg:squarederror, reg:squaredlogerror,
      reg:logistic, reg:gamma, reg:tweedie, reg:pseudohubererror,
      binary:logistic, multi:softprob].
    eval_metric: Evaluation metrics for validation data represented as a
      comma-separated string.
    num_boost_round: Number of boosting iterations.
    early_stopping_rounds: Activates early stopping. Validation error needs to
      decrease at least every early_stopping_rounds round(s) to continue
      training.
    base_score: The initial prediction score of all instances, global bias.
    disable_default_eval_metric: Flag to disable default metric. Set to >0 to
      disable. Default to 0.
    seed: Random seed.
    seed_per_iteration: Seed PRNG determnisticly via iterator number.
    booster: Which booster to use, can be gbtree, gblinear or dart. gbtree and
      dart use tree based model while gblinear uses linear function.
    eta: Learning rate.
    gamma: Minimum loss reduction required to make a further partition on a leaf
      node of the tree.
    max_depth: Maximum depth of a tree.
    min_child_weight: Minimum sum of instance weight(hessian) needed in a child.
    max_delta_step: Maximum delta step we allow each tree's weight estimation to
      be.
    subsample: Subsample ratio of the training instance.
    colsample_bytree: Subsample ratio of columns when constructing each tree.
    colsample_bylevel: Subsample ratio of columns for each split, in each level.
    colsample_bynode: Subsample ratio of columns for each node (split).
    reg_lambda: L2 regularization term on weights.
    reg_alpha: L1 regularization term on weights.
    tree_method: The tree construction algorithm used in XGBoost. Choices:
      ["auto", "exact", "approx", "hist", "gpu_exact", "gpu_hist"].
    scale_pos_weight: Control the balance of positive and negative weights.
    updater: A comma separated string defining the sequence of tree updaters to
      run.
    refresh_leaf: Refresh updater plugin. Update tree leaf and nodes's stats if
      True. When it is False, only node stats are updated.
    process_type: A type of boosting process to run. Choices:["default",
      "update"]
    grow_policy: Controls a way new nodes are added to the tree. Only supported
      if tree_method is hist. Choices:["depthwise", "lossguide"]
    sampling_method: The method to use to sample the training instances.
    monotone_constraints: Constraint of variable monotonicity.
    interaction_constraints: Constraints for interaction representing permitted
      interactions.
    sample_type: [dart booster only] Type of sampling algorithm.
      Choices:["uniform", "weighted"]
    normalize_type: [dart booster only] Type of normalization algorithm,
      Choices:["tree", "forest"]
    rate_drop: [dart booster only] Dropout rate.'
    one_drop: [dart booster only] When this flag is enabled, at least one tree
      is always dropped during the dropout (allows Binomial-plus-one or
      epsilon-dropout from the original DART paper).
    skip_drop: [dart booster only] Probability of skipping the dropout procedure
      during a boosting iteration.
    num_parallel_tree: Number of parallel trees constructed during each
      iteration. This option is used to support boosted random forest.
    feature_selector: [linear booster only] Feature selection and ordering
      method.
    top_k: The number of top features to select in greedy and thrifty feature
      selector. The value of 0 means using all the features.
    max_cat_to_onehot: A threshold for deciding whether XGBoost should use
      one-hot encoding based split for categorical data.
    max_leaves: Maximum number of nodes to be added.
    max_bin: Maximum number of discrete bins to bucket continuous features.
    tweedie_variance_power: Parameter that controls the variance of the Tweedie
      distribution.
    huber_slope: A parameter used for Pseudo-Huber loss to define the delta
      term.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    training_machine_type: Machine type.
    training_total_replica_count: Number of workers.
    training_accelerator_type: Accelerator type.
    training_accelerator_count: Accelerator count.
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  parameter_values = {}
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  training_and_eval_parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'objective': objective,
      'eval_metric': eval_metric,
      'num_boost_round': num_boost_round,
      'early_stopping_rounds': early_stopping_rounds,
      'base_score': base_score,
      'disable_default_eval_metric': disable_default_eval_metric,
      'seed': seed,
      'seed_per_iteration': seed_per_iteration,
      'booster': booster,
      'eta': eta,
      'gamma': gamma,
      'max_depth': max_depth,
      'min_child_weight': min_child_weight,
      'max_delta_step': max_delta_step,
      'subsample': subsample,
      'colsample_bytree': colsample_bytree,
      'colsample_bylevel': colsample_bylevel,
      'colsample_bynode': colsample_bynode,
      'reg_lambda': reg_lambda,
      'reg_alpha': reg_alpha,
      'tree_method': tree_method,
      'scale_pos_weight': scale_pos_weight,
      'updater': updater,
      'refresh_leaf': refresh_leaf,
      'process_type': process_type,
      'grow_policy': grow_policy,
      'sampling_method': sampling_method,
      'monotone_constraints': monotone_constraints,
      'interaction_constraints': interaction_constraints,
      'sample_type': sample_type,
      'normalize_type': normalize_type,
      'rate_drop': rate_drop,
      'one_drop': one_drop,
      'skip_drop': skip_drop,
      'num_parallel_tree': num_parallel_tree,
      'feature_selector': feature_selector,
      'top_k': top_k,
      'max_cat_to_onehot': max_cat_to_onehot,
      'max_leaves': max_leaves,
      'max_bin': max_bin,
      'tweedie_variance_power': tweedie_variance_power,
      'huber_slope': huber_slope,
      'weight_column': weight_column,
      'training_machine_type': training_machine_type,
      'training_total_replica_count': training_total_replica_count,
      'training_accelerator_type': training_accelerator_type,
      'training_accelerator_count': training_accelerator_count,
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }
  _update_parameters(parameter_values, training_and_eval_parameters)

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'xgboost_trainer_pipeline.yaml'
  )

  return pipeline_definition_path, parameter_values


def get_xgboost_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    objective: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    max_trial_count: int,
    parallel_trial_count: int,
    study_spec_parameters_override: Optional[List[Dict[str, Any]]] = None,
    eval_metric: Optional[str] = None,
    disable_default_eval_metric: Optional[int] = None,
    seed: Optional[int] = None,
    seed_per_iteration: Optional[bool] = None,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: Optional[bool] = None,
    feature_selection_algorithm: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    tf_custom_transformation_definitions: Optional[List[Dict[str, Any]]] = None,
    tf_transformations_path: Optional[str] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    weight_column: Optional[str] = None,
    max_failed_trial_count: Optional[int] = None,
    training_machine_type: Optional[str] = None,
    training_total_replica_count: Optional[int] = None,
    training_accelerator_type: Optional[str] = None,
    training_accelerator_count: Optional[int] = None,
    study_spec_algorithm: Optional[str] = None,
    study_spec_measurement_selection_type: Optional[str] = None,
    transform_dataflow_machine_type: Optional[str] = None,
    transform_dataflow_max_num_workers: Optional[int] = None,
    transform_dataflow_disk_size_gb: Optional[int] = None,
    run_evaluation: Optional[bool] = None,
    evaluation_batch_predict_machine_type: Optional[str] = None,
    evaluation_batch_predict_starting_replica_count: Optional[int] = None,
    evaluation_batch_predict_max_replica_count: Optional[int] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_starting_num_workers: Optional[int] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    dataflow_service_account: Optional[str] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: Optional[bool] = None,
    encryption_spec_key_name: Optional[str] = None,
):
  """Get the XGBoost HyperparameterTuningJob pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column: The target column name.
    objective: Specifies the learning task and the learning objective. Must be
      one of [reg:squarederror, reg:squaredlogerror,
      reg:logistic, reg:gamma, reg:tweedie, reg:pseudohubererror,
      binary:logistic, multi:softprob].
    study_spec_metric_id: Metric to optimize. For options, please look under
      'eval_metric' at
      https://xgboost.readthedocs.io/en/stable/parameter.html#learning-task-parameters.
    study_spec_metric_goal: Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    max_trial_count: The desired total number of trials.
    parallel_trial_count: The desired number of trials to run in parallel.
    study_spec_parameters_override: List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    eval_metric: Evaluation metrics for validation data represented as a
      comma-separated string.
    disable_default_eval_metric: Flag to disable default metric. Set to >0 to
      disable. Default to 0.
    seed: Random seed.
    seed_per_iteration: Seed PRNG determnisticly via iterator number.
    dataset_level_custom_transformation_definitions: Dataset-level custom
      transformation definitions in string format.
    dataset_level_transformations: Dataset-level transformation configuration in
      string format.
    run_feature_selection: Whether to enable feature selection.
    feature_selection_algorithm: Feature selection algorithm.
    max_selected_features: Maximum number of features to select.
    predefined_split_key: Predefined split key.
    stratified_split_key: Stratified split key.
    training_fraction: Training fraction.
    validation_fraction: Validation fraction.
    test_fraction: Test fraction.
    tf_auto_transform_features: List of auto transform features in the
      comma-separated string format.
    tf_custom_transformation_definitions: TF custom transformation definitions
      in string format.
    tf_transformations_path: Path to TF transformation configuration.
    data_source_csv_filenames: The CSV data source.
    data_source_bigquery_table_path: The BigQuery data source.
    bigquery_staging_full_dataset_id: The BigQuery staging full dataset id for
      storing intermediate tables.
    weight_column: The weight column name.
    max_failed_trial_count: The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    training_machine_type: Machine type.
    training_total_replica_count: Number of workers.
    training_accelerator_type: Accelerator type.
    training_accelerator_count: Accelerator count.
    study_spec_algorithm: The search algorithm specified for the study. One of
      'ALGORITHM_UNSPECIFIED', 'GRID_SEARCH', or 'RANDOM_SEARCH'.
    study_spec_measurement_selection_type:  Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      transform component.
    run_evaluation: Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type: The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count: The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
    evaluation_dataflow_starting_num_workers: The initial number of Dataflow
      workers for evaluation components.
    evaluation_dataflow_max_num_workers: The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb: Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account: Custom service account to run dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  parameter_values = {}
  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  training_and_eval_parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column': target_column,
      'objective': objective,
      'eval_metric': eval_metric,
      'study_spec_metric_id': study_spec_metric_id,
      'study_spec_metric_goal': study_spec_metric_goal,
      'max_trial_count': max_trial_count,
      'parallel_trial_count': parallel_trial_count,
      'study_spec_parameters_override': (
          study_spec_parameters_override
          if study_spec_parameters_override
          else []
      ),
      'disable_default_eval_metric': disable_default_eval_metric,
      'seed': seed,
      'seed_per_iteration': seed_per_iteration,
      'weight_column': weight_column,
      'max_failed_trial_count': max_failed_trial_count,
      'training_machine_type': training_machine_type,
      'training_total_replica_count': training_total_replica_count,
      'training_accelerator_type': training_accelerator_type,
      'training_accelerator_count': training_accelerator_count,
      'study_spec_algorithm': study_spec_algorithm,
      'study_spec_measurement_selection_type': (
          study_spec_measurement_selection_type
      ),
      'transform_dataflow_machine_type': transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'run_evaluation': run_evaluation,
      'evaluation_batch_predict_machine_type': (
          evaluation_batch_predict_machine_type
      ),
      'evaluation_batch_predict_starting_replica_count': (
          evaluation_batch_predict_starting_replica_count
      ),
      'evaluation_batch_predict_max_replica_count': (
          evaluation_batch_predict_max_replica_count
      ),
      'evaluation_dataflow_machine_type': evaluation_dataflow_machine_type,
      'evaluation_dataflow_starting_num_workers': (
          evaluation_dataflow_starting_num_workers
      ),
      'evaluation_dataflow_max_num_workers': (
          evaluation_dataflow_max_num_workers
      ),
      'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
      'dataflow_service_account': dataflow_service_account,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }
  _update_parameters(parameter_values, training_and_eval_parameters)

  fte_params = {
      'dataset_level_custom_transformation_definitions': (
          dataset_level_custom_transformation_definitions
          if dataset_level_custom_transformation_definitions
          else []
      ),
      'dataset_level_transformations': (
          dataset_level_transformations if dataset_level_transformations else []
      ),
      'run_feature_selection': run_feature_selection,
      'feature_selection_algorithm': feature_selection_algorithm,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': (
          tf_auto_transform_features if tf_auto_transform_features else {}
      ),
      'tf_custom_transformation_definitions': (
          tf_custom_transformation_definitions
          if tf_custom_transformation_definitions
          else []
      ),
      'tf_transformations_path': tf_transformations_path,
  }
  _update_parameters(parameter_values, fte_params)

  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
  }
  _update_parameters(parameter_values, data_source_and_split_parameters)

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'xgboost_hyperparameter_tuning_job_pipeline.yaml',
  )

  return pipeline_definition_path, parameter_values
