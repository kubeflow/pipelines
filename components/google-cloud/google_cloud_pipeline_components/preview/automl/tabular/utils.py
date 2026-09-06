"""Util functions for AutoML Tabular pipeline."""

import json
import os
import pathlib
from typing import Any, Dict, List, Optional, Tuple, Union
import uuid

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
_FEATURE_SELECTION_EXECUTION_ENGINE_BIGQUERY = 'bigquery'

# Needed because we reference the AutoML Tabular V1 pipeline.
_GCPC_STAGING_PATH = pathlib.Path(
    __file__
).parent.parent.parent.parent.resolve()
_GCPC_GA_TABULAR_PATH = str(_GCPC_STAGING_PATH / 'v1' / 'automl' / 'tabular')


def _generate_model_display_name() -> str:
  """Automatically generates a model_display_name.

  Returns:
    model_display_name.
  """
  return f'tabular-workflow-model-{uuid.uuid4()}'


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
    enable_fte: bool = False,
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
    enable_fte: Whether to enable the Feature Transform Engine.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  if not study_spec_parameters_override:
    study_spec_parameters_override = []
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []  # pyrefly: ignore[bad-assignment]
  if not cv_trainer_worker_pool_specs_override:
    cv_trainer_worker_pool_specs_override = []  # pyrefly: ignore[bad-assignment]
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

  # V1 pipeline
  if not enable_fte:
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

  # V2 pipeline (with FTE)
  else:
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
    enable_fte: bool = False,
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
    enable_fte: Whether to enable the Feature Transform Engine.

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
      weight_column=weight_column,  # pyrefly: ignore[bad-argument-type]
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
      enable_fte=enable_fte,
  )

  # V1 pipeline without FTE
  if not enable_fte:
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
      weight_column=weight_column,  # pyrefly: ignore[bad-argument-type]
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

  return get_automl_tabular_pipeline_and_parameters(  # pytype: disable=wrong-arg-types
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
      run_distillation=None,  # pyrefly: ignore[bad-argument-type]
      distill_batch_predict_machine_type=None,
      distill_batch_predict_starting_replica_count=None,
      distill_batch_predict_max_replica_count=None,
      stage_1_tuning_result_artifact_uri=stage_1_tuning_result_artifact_uri,
      quantiles=[],
      enable_probabilistic_inference=False,
  )


def get_feature_selection_pipeline_and_parameters(
    root_dir: str,
    project: str,
    location: str,
    target_column: str,
    prediction_type: str,
    optimization_objective: str,
    dataset_level_custom_transformation_definitions: Optional[
        List[Dict[str, Any]]
    ] = None,
    dataset_level_transformations: Optional[List[Dict[str, Any]]] = None,
    run_feature_selection: Optional[bool] = None,
    feature_selection_algorithm: Optional[str] = None,
    feature_selection_execution_engine: Optional[
        str
    ] = _FEATURE_SELECTION_EXECUTION_ENGINE_BIGQUERY,
    max_selected_features: Optional[int] = None,
    predefined_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    tf_auto_transform_features: Optional[
        Union[List[str], Dict[str, List[str]]]
    ] = None,
    weight_column: Optional[str] = None,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    bigquery_staging_full_dataset_id: Optional[str] = None,
    dataflow_machine_type: Optional[str] = None,
    dataflow_max_num_workers: Optional[int] = None,
    dataflow_disk_size_gb: Optional[int] = None,
    dataflow_subnetwork: Optional[str] = None,
    dataflow_use_public_ips: Optional[bool] = None,
    encryption_spec_key_name: Optional[str] = None,
    stage_1_deadline_hours: Optional[float] = None,
    stage_2_deadline_hours: Optional[float] = None,
):
  """Returns feature transform engine pipeline and formatted parameters."""

  if isinstance(tf_auto_transform_features, list):
    tf_auto_transform_features = {'auto': tf_auto_transform_features}

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'feature_selection_pipeline.yaml'
  )

  parameter_values = {
      'root_dir': root_dir,
      'project': project,
      'location': location,
      'target_column': target_column,
      'weight_column': weight_column,
      'prediction_type': prediction_type,
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
      'feature_selection_execution_engine': feature_selection_execution_engine,
      'max_selected_features': max_selected_features,
      'predefined_split_key': predefined_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction,
      'tf_auto_transform_features': tf_auto_transform_features,
      'optimization_objective': optimization_objective,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'bigquery_staging_full_dataset_id': bigquery_staging_full_dataset_id,
      'dataflow_machine_type': dataflow_machine_type,
      'dataflow_max_num_workers': dataflow_max_num_workers,
      'dataflow_disk_size_gb': dataflow_disk_size_gb,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
      'stage_1_deadline_hours': stage_1_deadline_hours,
      'stage_2_deadline_hours': stage_2_deadline_hours,
  }

  parameter_values = {
      param: value
      for param, value in parameter_values.items()
      if value is not None
  }

  return pipeline_definition_path, parameter_values
