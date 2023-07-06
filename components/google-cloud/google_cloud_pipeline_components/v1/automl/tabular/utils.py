"""Util functions for AutoML Tabular pipeline."""

import json
import math
import os
import pathlib
from typing import Any, Dict, List, Optional, Tuple
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

# Needed because we reference the AutoML Tabular V2 pipeline.
_GCPC_STAGING_PATH = pathlib.Path(
    __file__
).parent.parent.parent.parent.resolve()
_GCPC_PREVIEW_TABULAR_PATH = (
    _GCPC_STAGING_PATH / 'preview' / 'automl' / 'tabular'
)


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
        pathlib.Path(__file__).parent.resolve(), 'automl_tabular_pipeline.yaml'
    )

  # V2 pipeline with FTE
  else:
    pipeline_definition_path = os.path.join(
        _GCPC_PREVIEW_TABULAR_PATH,
        'automl_tabular_v2_pipeline.yaml',
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


def get_skip_evaluation_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column_name: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: Dict[str, Any],
    split_spec: Dict[str, Any],
    data_source: Dict[str, Any],
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_selected_trials: int = _DEFAULT_STAGE_2_NUM_SELECTED_TRAILS,
    weight_column_name: str = '',
    study_spec_override: Optional[Dict[str, Any]] = None,
    optimization_objective_recall_value: float = -1,
    optimization_objective_precision_value: float = -1,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    additional_experiments: Optional[Dict[str, Any]] = None,
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips evaluation.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The transformations to apply.
    split_spec: The split spec.
    data_source: The data source.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    weight_column_name: The weight column name.
    study_spec_override: The dictionary for overriding study spec. The
      dictionary should be of format
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

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  return get_default_pipeline_and_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column_name=target_column_name,
      prediction_type=prediction_type,
      optimization_objective=optimization_objective,
      transformations=transformations,
      split_spec=split_spec,
      data_source=data_source,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      stage_2_num_selected_trials=stage_2_num_selected_trials,
      weight_column_name=weight_column_name,
      study_spec_override=study_spec_override,
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
      run_evaluation=False,
      run_distillation=False,
  )


def get_default_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column_name: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: Dict[str, Any],
    split_spec: Dict[str, Any],
    data_source: Dict[str, Any],
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_selected_trials: int = _DEFAULT_STAGE_2_NUM_SELECTED_TRAILS,
    weight_column_name: str = '',
    study_spec_override: Optional[Dict[str, Any]] = None,
    optimization_objective_recall_value: float = -1,
    optimization_objective_precision_value: float = -1,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    additional_experiments: Optional[Dict[str, Any]] = None,
    dataflow_service_account: str = '',
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type: str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count: int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count: int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers: int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: str = 'n1-standard-16',
    distill_batch_predict_starting_replica_count: int = 25,
    distill_batch_predict_max_replica_count: int = 25,
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular default training pipeline.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The transformations to apply.
    split_spec: The split spec.
    data_source: The data source.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    weight_column_name: The weight column name.
    study_spec_override: The dictionary for overriding study spec. The
      dictionary should be of format
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
    evaluation_dataflow_machine_type: The dataflow machine type for evaluation
      components.
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

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  warnings.warn(
      'This method is deprecated,'
      ' please use get_automl_tabular_pipeline_and_parameters instead.'
  )

  if stage_1_num_parallel_trials <= 0:
    stage_1_num_parallel_trials = _DEFAULT_NUM_PARALLEL_TRAILS

  if stage_2_num_parallel_trials <= 0:
    stage_2_num_parallel_trials = _DEFAULT_NUM_PARALLEL_TRAILS

  hours = float(train_budget_milli_node_hours) / 1000.0
  multiplier = stage_1_num_parallel_trials * hours / 500.0
  stage_1_single_run_max_secs = int(math.sqrt(multiplier) * 2400.0)
  phase_2_rounds = int(
      math.sqrt(multiplier) * 100 / stage_2_num_parallel_trials + 0.5
  )
  if phase_2_rounds < 1:
    phase_2_rounds = 1

  # All of magic number "1.3" above is because the trial doesn't always finish
  # in time_per_trial. 1.3 is an empirical safety margin here.
  stage_1_deadline_secs = int(
      hours * 3600.0 - 1.3 * stage_1_single_run_max_secs * phase_2_rounds
  )

  if stage_1_deadline_secs < hours * 3600.0 * 0.5:
    stage_1_deadline_secs = int(hours * 3600.0 * 0.5)
    # Phase 1 deadline is the same as phase 2 deadline in this case. Phase 2
    # can't finish in time after the deadline is cut, so adjust the time per
    # trial to meet the deadline.
    stage_1_single_run_max_secs = int(
        stage_1_deadline_secs / (1.3 * phase_2_rounds)
    )

  reduce_search_space_mode = 'minimal'
  if multiplier > 2:
    reduce_search_space_mode = 'regular'
  if multiplier > 4:
    reduce_search_space_mode = 'full'

  # Stage 2 number of trials is stage_1_num_selected_trials *
  # _NUM_FOLDS, which should be equal to phase_2_rounds *
  # stage_2_num_parallel_trials. Use this information to calculate
  # stage_1_num_selected_trials:
  stage_1_num_selected_trials = int(
      phase_2_rounds * stage_2_num_parallel_trials / _NUM_FOLDS
  )
  stage_1_deadline_hours = stage_1_deadline_secs / 3600.0

  stage_2_deadline_hours = hours - stage_1_deadline_hours
  stage_2_single_run_max_secs = stage_1_single_run_max_secs

  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column_name': target_column_name,
      'prediction_type': prediction_type,
      'optimization_objective': optimization_objective,
      'transformations': input_dictionary_to_parameter(transformations),
      'split_spec': input_dictionary_to_parameter(split_spec),
      'data_source': input_dictionary_to_parameter(data_source),
      'stage_1_deadline_hours': stage_1_deadline_hours,
      'stage_1_num_parallel_trials': stage_1_num_parallel_trials,
      'stage_1_num_selected_trials': stage_1_num_selected_trials,
      'stage_1_single_run_max_secs': stage_1_single_run_max_secs,
      'reduce_search_space_mode': reduce_search_space_mode,
      'stage_2_deadline_hours': stage_2_deadline_hours,
      'stage_2_num_parallel_trials': stage_2_num_parallel_trials,
      'stage_2_num_selected_trials': stage_2_num_selected_trials,
      'stage_2_single_run_max_secs': stage_2_single_run_max_secs,
      'weight_column_name': weight_column_name,
      'optimization_objective_recall_value': (
          optimization_objective_recall_value
      ),
      'optimization_objective_precision_value': (
          optimization_objective_precision_value
      ),
      'study_spec_override': input_dictionary_to_parameter(study_spec_override),
      'stage_1_tuner_worker_pool_specs_override': input_dictionary_to_parameter(
          stage_1_tuner_worker_pool_specs_override
      ),
      'cv_trainer_worker_pool_specs_override': input_dictionary_to_parameter(
          cv_trainer_worker_pool_specs_override
      ),
      'export_additional_model_without_custom_ops': (
          export_additional_model_without_custom_ops
      ),
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
      'transform_dataflow_max_num_workers': transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb': transform_dataflow_disk_size_gb,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name': encryption_spec_key_name,
  }
  if additional_experiments:
    parameter_values.update(
        {
            'additional_experiments': input_dictionary_to_parameter(
                additional_experiments
            )
        }
    )
  if run_evaluation:
    parameter_values.update({
        'dataflow_service_account': dataflow_service_account,
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
        'evaluation_dataflow_max_num_workers': (
            evaluation_dataflow_max_num_workers
        ),
        'evaluation_dataflow_disk_size_gb': evaluation_dataflow_disk_size_gb,
        'run_evaluation': run_evaluation,
    })
  if run_distillation:
    # All of magic number "1.3" above is because the trial doesn't always finish
    # in time_per_trial. 1.3 is an empirical safety margin here.
    distill_stage_1_deadline_hours = (
        math.ceil(
            float(_DISTILL_TOTAL_TRIALS)
            / parameter_values['stage_1_num_parallel_trials']
        )
        * parameter_values['stage_1_single_run_max_secs']
        * 1.3
        / 3600.0
    )

    parameter_values.update({
        'distill_stage_1_deadline_hours': distill_stage_1_deadline_hours,
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
    })
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'deprecated/default_pipeline.json',
  )
  return pipeline_definition_path, parameter_values


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


def get_distill_skip_evaluation_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column_name: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: Dict[str, Any],
    split_spec: Dict[str, Any],
    data_source: Dict[str, Any],
    train_budget_milli_node_hours: float,
    stage_1_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_selected_trials: int = _DEFAULT_STAGE_2_NUM_SELECTED_TRAILS,
    weight_column_name: str = '',
    study_spec_override: Optional[Dict[str, Any]] = None,
    optimization_objective_recall_value: float = -1,
    optimization_objective_precision_value: float = -1,
    stage_1_tuner_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    additional_experiments: Optional[Dict[str, Any]] = None,
    distill_batch_predict_machine_type: str = 'n1-standard-16',
    distill_batch_predict_starting_replica_count: int = 25,
    distill_batch_predict_max_replica_count: int = 25,
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that distill and skips evaluation.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The transformations to apply.
    split_spec: The split spec.
    data_source: The data source.
    train_budget_milli_node_hours: The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials: Number of parallel trails for stage 1.
    stage_2_num_parallel_trials: Number of parallel trails for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    weight_column_name: The weight column name.
    study_spec_override: The dictionary for overriding study spec. The
      dictionary should be of format
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
    distill_batch_predict_machine_type: The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count: The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count: The max number of prediction server
      for batch predict component in the model distillation.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  warnings.warn(
      'Depreciated. Please use get_automl_tabular_pipeline_and_parameters.'
  )

  return get_default_pipeline_and_parameters(
      project=project,
      location=location,
      root_dir=root_dir,
      target_column_name=target_column_name,
      prediction_type=prediction_type,
      optimization_objective=optimization_objective,
      transformations=transformations,
      split_spec=split_spec,
      data_source=data_source,
      train_budget_milli_node_hours=train_budget_milli_node_hours,
      stage_1_num_parallel_trials=stage_1_num_parallel_trials,
      stage_2_num_parallel_trials=stage_2_num_parallel_trials,
      stage_2_num_selected_trials=stage_2_num_selected_trials,
      weight_column_name=weight_column_name,
      study_spec_override=study_spec_override,
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
      distill_batch_predict_machine_type=distill_batch_predict_machine_type,
      distill_batch_predict_starting_replica_count=distill_batch_predict_starting_replica_count,
      distill_batch_predict_max_replica_count=distill_batch_predict_max_replica_count,
      run_evaluation=False,
      run_distillation=True,
  )
