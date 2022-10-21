"""Util functions for AutoML Tabular pipeline."""

import json
import math
import os
import pathlib
from typing import Any, Dict, List, Tuple, Optional
import warnings

_DEFAULT_NUM_PARALLEL_TRAILS = 35
_DEFAULT_STAGE_2_NUM_SELECTED_TRAILS = 5
_NUM_FOLDS = 5
_DISTILL_TOTAL_TRIALS = 100
_EVALUATION_BATCH_PREDICT_MACHINE_TYPE = 'n1-standard-16'
_EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT = 25
_EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT = 25
_EVALUATION_DATAFLOW_MACHINE_TYPE = 'n1-standard-4'
_EVALUATION_DATAFLOW_MAX_NUM_WORKERS = 25
_EVALUATION_DATAFLOW_DISK_SIZE_GB = 50


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
    evaluation_batch_predict_starting_replica_count: Optional[str] = None,
    evaluation_batch_predict_max_replica_count: Optional[str] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None,
    quantiles: Optional[List[float]] = None,
    enable_probabilistic_inference: bool = False
) -> Dict[str, Any]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials:
      Number of parallel trails for stage 1.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      float = The test fraction.
    weight_column:
      The weight column name.
    study_spec_parameters_override:
      The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override:
      The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    max_selected_features:
      number of features to select for training,
    apply_feature_selection_tuning:
      tuning feature selection rate if true.
    run_evaluation:
      Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    run_distillation:
      Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type:
      The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count:
      The max number of prediction server
      for batch predict component in the model distillation.
    quantiles:
      Quantiles to use for probabilistic inference. Up to 5 quantiles are
      allowed of values between 0 and 1, exclusive. Represents the quantiles to
      use for that objective. Quantiles must be unique.
    enable_probabilistic_inference:
      If probabilistic inference is enabled, the model will fit a
      distribution that captures the uncertainty of a prediction. At inference
      time, the predictive distribution is used to make a point prediction that
      minimizes the optimization objective. For example, the mean of a
      predictive distribution is the point prediction that minimizes RMSE loss.
      If quantiles are specified, then the quantiles of the distribution are
      also returned.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not study_spec_parameters_override:
    study_spec_parameters_override = {}
  if not stage_1_tuner_worker_pool_specs_override:
    stage_1_tuner_worker_pool_specs_override = []
  if not cv_trainer_worker_pool_specs_override:
    cv_trainer_worker_pool_specs_override = []
  if not additional_experiments:
    additional_experiments = {}
  if not quantiles:
    quantiles = []

  parameter_values = {}
  parameters = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'data_source_csv_filenames':
          data_source_csv_filenames,
      'data_source_bigquery_table_path':
          data_source_bigquery_table_path,
      'predefined_split_key':
          predefined_split_key,
      'timestamp_split_key':
          timestamp_split_key,
      'stratified_split_key':
          stratified_split_key,
      'training_fraction':
          training_fraction,
      'validation_fraction':
          validation_fraction,
      'test_fraction':
          test_fraction,
      'optimization_objective':
          optimization_objective,
      'transformations':
          transformations,
      'train_budget_milli_node_hours':
          train_budget_milli_node_hours,
      'stage_1_num_parallel_trials':
          stage_1_num_parallel_trials,
      'stage_2_num_parallel_trials':
          stage_2_num_parallel_trials,
      'stage_2_num_selected_trials':
          stage_2_num_selected_trials,
      'weight_column':
          weight_column,
      'optimization_objective_recall_value':
          optimization_objective_recall_value,
      'optimization_objective_precision_value':
          optimization_objective_precision_value,
      'study_spec_parameters_override':
          study_spec_parameters_override,
      'stage_1_tuner_worker_pool_specs_override':
          stage_1_tuner_worker_pool_specs_override,
      'cv_trainer_worker_pool_specs_override':
          cv_trainer_worker_pool_specs_override,
      'export_additional_model_without_custom_ops':
          export_additional_model_without_custom_ops,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'dataflow_service_account':
          dataflow_service_account,
      'encryption_spec_key_name':
          encryption_spec_key_name,
      'additional_experiments':
          additional_experiments,
      'max_selected_features':
          max_selected_features,
      'quantiles':
          quantiles,
      'enable_probabilistic_inference':
          enable_probabilistic_inference,
  }
  parameter_values.update({
      param: value for param, value in parameters.items() if value is not None
  })

  if apply_feature_selection_tuning:
    parameter_values.update({
        'apply_feature_selection_tuning': apply_feature_selection_tuning,
    })

  if run_evaluation:
    eval_parameters = {
        'evaluation_batch_predict_machine_type':
            evaluation_batch_predict_machine_type,
        'evaluation_batch_predict_starting_replica_count':
            evaluation_batch_predict_starting_replica_count,
        'evaluation_batch_predict_max_replica_count':
            evaluation_batch_predict_max_replica_count,
        'evaluation_dataflow_machine_type':
            evaluation_dataflow_machine_type,
        'evaluation_dataflow_max_num_workers':
            evaluation_dataflow_max_num_workers,
        'evaluation_dataflow_disk_size_gb':
            evaluation_dataflow_disk_size_gb,
        'run_evaluation':
            run_evaluation,
    }
    parameter_values.update({
        param: value
        for param, value in eval_parameters.items()
        if value is not None
    })
  if run_distillation:
    distillation_parameters = {
        'distill_batch_predict_machine_type':
            distill_batch_predict_machine_type,
        'distill_batch_predict_starting_replica_count':
            distill_batch_predict_starting_replica_count,
        'distill_batch_predict_max_replica_count':
            distill_batch_predict_max_replica_count,
        'run_distillation':
            run_distillation,
    }
    parameter_values.update({
        param: value
        for param, value in distillation_parameters.items()
        if value is not None
    })
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
    evaluation_batch_predict_starting_replica_count: Optional[str] = None,
    evaluation_batch_predict_max_replica_count: Optional[str] = None,
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None,
    quantiles: Optional[List[float]] = None,
    enable_probabilistic_inference: bool = False
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials:
      Number of parallel trails for stage 1.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      float = The test fraction.
    weight_column:
      The weight column name.
    study_spec_parameters_override:
      The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override:
      The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    run_evaluation:
      Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    run_distillation:
      Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type:
      The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count:
      The max number of prediction server
      for batch predict component in the model distillation.
    quantiles:
      Quantiles to use for probabilistic inference. Up to 5 quantiles are
      allowed of values between 0 and 1, exclusive. Represents the quantiles to
      use for that objective. Quantiles must be unique.
    enable_probabilistic_inference:
      If probabilistic inference is enabled, the model will fit a
      distribution that captures the uncertainty of a prediction. At inference
      time, the predictive distribution is used to make a point prediction that
      minimizes the optimization objective. For example, the mean of a
      predictive distribution is the point prediction that minimizes RMSE loss.
      If quantiles are specified, then the quantiles of the distribution are
      also returned.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
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
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      run_distillation=run_distillation,
      distill_batch_predict_machine_type=distill_batch_predict_machine_type,
      distill_batch_predict_starting_replica_count=distill_batch_predict_starting_replica_count,
      distill_batch_predict_max_replica_count=distill_batch_predict_max_replica_count,
      quantiles=quantiles,
      enable_probabilistic_inference=enable_probabilistic_inference,
  )

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'automl_tabular_pipeline.json')
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
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None,
    max_selected_features: int = 1000,
    apply_feature_selection_tuning: bool = False,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: Optional[str] = None,
    distill_batch_predict_starting_replica_count: Optional[int] = None,
    distill_batch_predict_max_replica_count: Optional[int] = None
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular v1 default training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The path to a GCS file containing the transformations to
      apply.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials:
      Number of parallel trails for stage 1.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      float = The test fraction.
    weight_column:
      The weight column name.
    study_spec_parameters_override:
      The list for overriding study spec. The list
      should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override:
      The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    run_evaluation:
      Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    max_selected_features:
      number of features to select for training,
    apply_feature_selection_tuning:
      tuning feature selection rate if true.
    run_distillation:
      Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type:
      The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count:
      The max number of prediction server
      for batch predict component in the model distillation.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
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
      max_selected_features=max_selected_features,
      apply_feature_selection_tuning=apply_feature_selection_tuning,
      run_evaluation=run_evaluation,
      evaluation_batch_predict_machine_type=evaluation_batch_predict_machine_type,
      evaluation_batch_predict_starting_replica_count=evaluation_batch_predict_starting_replica_count,
      evaluation_batch_predict_max_replica_count=evaluation_batch_predict_max_replica_count,
      evaluation_dataflow_machine_type=evaluation_dataflow_machine_type,
      evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
      evaluation_dataflow_disk_size_gb=evaluation_dataflow_disk_size_gb,
      run_distillation=run_distillation,
      distill_batch_predict_machine_type=distill_batch_predict_machine_type,
      distill_batch_predict_starting_replica_count=distill_batch_predict_starting_replica_count,
      distill_batch_predict_max_replica_count=distill_batch_predict_max_replica_count
  )

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'automl_tabular_feature_selection_pipeline.json')
  return pipeline_definition_path, parameter_values


def input_dictionary_to_parameter(input_dict: Optional[Dict[str, Any]]) -> str:
  """Convert json input dict to encoded parameter string.

  This function is required due to the limitation on YAML component definition
  that YAML definition does not have a keyword for apply quote escape, so the
  JSON argument's quote must be manually escaped using this function.

  Args:
    input_dict:
      The input json dictionary.

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
    additional_experiments: Optional[Dict[str, Any]] = None
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips evaluation.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column_name:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The transformations to apply.
    split_spec:
      The split spec.
    data_source:
      The data source.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials:
      Number of parallel trails for stage 1.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    weight_column_name:
      The weight column name.
    study_spec_override:
      The dictionary for overriding study spec. The
      dictionary should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override:
      The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
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
      run_distillation=False)


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
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    run_distillation: bool = False,
    distill_batch_predict_machine_type: str = 'n1-standard-16',
    distill_batch_predict_starting_replica_count: int = 25,
    distill_batch_predict_max_replica_count: int = 25
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular default training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column_name:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The transformations to apply.
    split_spec:
      The split spec.
    data_source:
      The data source.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_num_parallel_trials:
      Number of parallel trails for stage 1.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    weight_column_name:
      The weight column name.
    study_spec_override:
      The dictionary for overriding study spec. The
      dictionary should be of format
      https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/study.proto#L181.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    stage_1_tuner_worker_pool_specs_override:
      The dictionary for overriding.
      stage 1 tuner worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    run_evaluation:
      Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    run_distillation:
      Whether to run distill in the training pipeline.
    distill_batch_predict_machine_type:
      The prediction server machine type for
      batch predict component in the model distillation.
    distill_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict component in the model distillation.
    distill_batch_predict_max_replica_count:
      The max number of prediction server
      for batch predict component in the model distillation.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  warnings.warn(
      'This method is deprecated,'
      ' please use get_automl_tabular_pipeline_and_parameters instead.')

  if stage_1_num_parallel_trials <= 0:
    stage_1_num_parallel_trials = _DEFAULT_NUM_PARALLEL_TRAILS

  if stage_2_num_parallel_trials <= 0:
    stage_2_num_parallel_trials = _DEFAULT_NUM_PARALLEL_TRAILS

  hours = float(train_budget_milli_node_hours) / 1000.0
  multiplier = stage_1_num_parallel_trials * hours / 500.0
  stage_1_single_run_max_secs = int(math.sqrt(multiplier) * 2400.0)
  phase_2_rounds = int(
      math.sqrt(multiplier) * 100 / stage_2_num_parallel_trials + 0.5)
  if phase_2_rounds < 1:
    phase_2_rounds = 1

  # All of magic number "1.3" above is because the trial doesn't always finish
  # in time_per_trial. 1.3 is an empirical safety margin here.
  stage_1_deadline_secs = int(hours * 3600.0 - 1.3 *
                              stage_1_single_run_max_secs * phase_2_rounds)

  if stage_1_deadline_secs < hours * 3600.0 * 0.5:
    stage_1_deadline_secs = int(hours * 3600.0 * 0.5)
    # Phase 1 deadline is the same as phase 2 deadline in this case. Phase 2
    # can't finish in time after the deadline is cut, so adjust the time per
    # trial to meet the deadline.
    stage_1_single_run_max_secs = int(stage_1_deadline_secs /
                                      (1.3 * phase_2_rounds))

  reduce_search_space_mode = 'minimal'
  if multiplier > 2:
    reduce_search_space_mode = 'regular'
  if multiplier > 4:
    reduce_search_space_mode = 'full'

  # Stage 2 number of trials is stage_1_num_selected_trials *
  # _NUM_FOLDS, which should be equal to phase_2_rounds *
  # stage_2_num_parallel_trials. Use this information to calculate
  # stage_1_num_selected_trials:
  stage_1_num_selected_trials = int(phase_2_rounds *
                                    stage_2_num_parallel_trials / _NUM_FOLDS)
  stage_1_deadline_hours = stage_1_deadline_secs / 3600.0

  stage_2_deadline_hours = hours - stage_1_deadline_hours
  stage_2_single_run_max_secs = stage_1_single_run_max_secs

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column_name':
          target_column_name,
      'prediction_type':
          prediction_type,
      'optimization_objective':
          optimization_objective,
      'transformations':
          input_dictionary_to_parameter(transformations),
      'split_spec':
          input_dictionary_to_parameter(split_spec),
      'data_source':
          input_dictionary_to_parameter(data_source),
      'stage_1_deadline_hours':
          stage_1_deadline_hours,
      'stage_1_num_parallel_trials':
          stage_1_num_parallel_trials,
      'stage_1_num_selected_trials':
          stage_1_num_selected_trials,
      'stage_1_single_run_max_secs':
          stage_1_single_run_max_secs,
      'reduce_search_space_mode':
          reduce_search_space_mode,
      'stage_2_deadline_hours':
          stage_2_deadline_hours,
      'stage_2_num_parallel_trials':
          stage_2_num_parallel_trials,
      'stage_2_num_selected_trials':
          stage_2_num_selected_trials,
      'stage_2_single_run_max_secs':
          stage_2_single_run_max_secs,
      'weight_column_name':
          weight_column_name,
      'optimization_objective_recall_value':
          optimization_objective_recall_value,
      'optimization_objective_precision_value':
          optimization_objective_precision_value,
      'study_spec_override':
          input_dictionary_to_parameter(study_spec_override),
      'stage_1_tuner_worker_pool_specs_override':
          input_dictionary_to_parameter(stage_1_tuner_worker_pool_specs_override
                                       ),
      'cv_trainer_worker_pool_specs_override':
          input_dictionary_to_parameter(cv_trainer_worker_pool_specs_override),
      'export_additional_model_without_custom_ops':
          export_additional_model_without_custom_ops,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }
  if additional_experiments:
    parameter_values.update({
        'additional_experiments':
            input_dictionary_to_parameter(additional_experiments)
    })
  if run_evaluation:
    parameter_values.update({
        'dataflow_service_account':
            dataflow_service_account,
        'evaluation_batch_predict_machine_type':
            evaluation_batch_predict_machine_type,
        'evaluation_batch_predict_starting_replica_count':
            evaluation_batch_predict_starting_replica_count,
        'evaluation_batch_predict_max_replica_count':
            evaluation_batch_predict_max_replica_count,
        'evaluation_dataflow_machine_type':
            evaluation_dataflow_machine_type,
        'evaluation_dataflow_max_num_workers':
            evaluation_dataflow_max_num_workers,
        'evaluation_dataflow_disk_size_gb':
            evaluation_dataflow_disk_size_gb,
        'run_evaluation':
            run_evaluation,
    })
  if run_distillation:
    # All of magic number "1.3" above is because the trial doesn't always finish
    # in time_per_trial. 1.3 is an empirical safety margin here.
    distill_stage_1_deadline_hours = math.ceil(
        float(_DISTILL_TOTAL_TRIALS) /
        parameter_values['stage_1_num_parallel_trials']
    ) * parameter_values['stage_1_single_run_max_secs'] * 1.3 / 3600.0

    parameter_values.update({
        'distill_stage_1_deadline_hours':
            distill_stage_1_deadline_hours,
        'distill_batch_predict_machine_type':
            distill_batch_predict_machine_type,
        'distill_batch_predict_starting_replica_count':
            distill_batch_predict_starting_replica_count,
        'distill_batch_predict_max_replica_count':
            distill_batch_predict_max_replica_count,
        'run_distillation':
            run_distillation,
    })
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'deprecated/default_pipeline.json')
  return pipeline_definition_path, parameter_values


def get_feature_selection_pipeline_and_parameters(
    project: str, location: str, root_dir: str, target_column: str,
    algorithm: str, prediction_type: str,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    max_selected_features: Optional[int] = None,
    dataflow_machine_type: str = 'n1-standard-16',
    dataflow_max_num_workers: int = 25,
    dataflow_disk_size_gb: int = 40,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    dataflow_service_account: str = ''):
  """Get the feature selection pipeline that generates feature ranking and selected features.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    algorithm:
      Algorithm to select features, default to be AMI.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    data_source_csv_filenames:
      A string that represents a list of comma
      separated CSV filenames.
    data_source_bigquery_table_path:
      The BigQuery table path.
    max_selected_features:
      number of features to be selected.
    dataflow_machine_type:
      The dataflow machine type for
      feature_selection component.
    dataflow_max_num_workers:
      The max number of Dataflow
      workers for feature_selection component.
    dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for feature_selection component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    dataflow_service_account:
      Custom service account to run dataflow jobs.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """

  parameter_values = {}
  data_source_parameters = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'target_column_name': target_column,
      'algorithm': algorithm,
      'prediction_type': prediction_type,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'max_selected_features': max_selected_features,
      'dataflow_machine_type':
          dataflow_machine_type,
      'dataflow_max_num_workers':
          dataflow_max_num_workers,
      'dataflow_disk_size_gb':
          dataflow_disk_size_gb,
      'dataflow_service_account':
          dataflow_service_account,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
  }

  parameter_values.update({
      param: value
      for param, value in data_source_parameters.items()
      if value is not None
  })

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'feature_selection_pipeline.json')

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
    evaluation_dataflow_machine_type: Optional[str] = None,
    evaluation_dataflow_max_num_workers: Optional[int] = None,
    evaluation_dataflow_disk_size_gb: Optional[int] = None
) -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips architecture search.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    optimization_objective:
      For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations:
      The transformations to apply.
    train_budget_milli_node_hours:
      The train budget of creating this model,
      expressed in milli node hours i.e. 1,000 value in this field means 1 node
      hour.
    stage_1_tuning_result_artifact_uri:
      The stage 1 tuning result artifact GCS
      URI.
    stage_2_num_parallel_trials:
      Number of parallel trails for stage 2.
    stage_2_num_selected_trials:
      Number of selected trials for stage 2.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      float = The test fraction.
    weight_column:
      The weight column name.
    optimization_objective_recall_value:
      Required when optimization_objective is
      "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
    optimization_objective_precision_value:
      Required when optimization_objective
      is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
    cv_trainer_worker_pool_specs_override:
      The dictionary for overriding stage
      cv trainer worker pool spec. The dictionary should be of format
        https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    export_additional_model_without_custom_ops:
      Whether to export additional
      model without custom TensorFlow operators.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.
    additional_experiments:
      Use this field to config private preview features.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    run_evaluation:
      Whether to run evaluation in the training pipeline.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not cv_trainer_worker_pool_specs_override:
    cv_trainer_worker_pool_specs_override = []
  if not additional_experiments:
    additional_experiments = {}
  parameter_values = {}
  parameters = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'data_source_csv_filenames':
          data_source_csv_filenames,
      'data_source_bigquery_table_path':
          data_source_bigquery_table_path,
      'predefined_split_key':
          predefined_split_key,
      'timestamp_split_key':
          timestamp_split_key,
      'stratified_split_key':
          stratified_split_key,
      'training_fraction':
          training_fraction,
      'validation_fraction':
          validation_fraction,
      'test_fraction':
          test_fraction,
      'optimization_objective':
          optimization_objective,
      'transformations':
          transformations,
      'train_budget_milli_node_hours':
          train_budget_milli_node_hours,
      'stage_1_tuning_result_artifact_uri':
          stage_1_tuning_result_artifact_uri,
      'stage_2_num_parallel_trials':
          stage_2_num_parallel_trials,
      'stage_2_num_selected_trials':
          stage_2_num_selected_trials,
      'weight_column':
          weight_column,
      'optimization_objective_recall_value':
          optimization_objective_recall_value,
      'optimization_objective_precision_value':
          optimization_objective_precision_value,
      'cv_trainer_worker_pool_specs_override':
          cv_trainer_worker_pool_specs_override,
      'export_additional_model_without_custom_ops':
          export_additional_model_without_custom_ops,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
      'additional_experiments':
          additional_experiments,
  }
  parameter_values.update({
      param: value for param, value in parameters.items() if value is not None
  })
  if run_evaluation:
    eval_parameters = {
        'dataflow_service_account':
            dataflow_service_account,
        'evaluation_batch_predict_machine_type':
            evaluation_batch_predict_machine_type,
        'evaluation_batch_predict_starting_replica_count':
            evaluation_batch_predict_starting_replica_count,
        'evaluation_batch_predict_max_replica_count':
            evaluation_batch_predict_max_replica_count,
        'evaluation_dataflow_machine_type':
            evaluation_dataflow_machine_type,
        'evaluation_dataflow_max_num_workers':
            evaluation_dataflow_max_num_workers,
        'evaluation_dataflow_disk_size_gb':
            evaluation_dataflow_disk_size_gb,
        'run_evaluation':
            run_evaluation
    }
    parameter_values.update({
        param: value
        for param, value in eval_parameters.items()
        if value is not None
    })

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'skip_architecture_search_pipeline.json')
  return pipeline_definition_path, parameter_values


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
    distill_batch_predict_max_replica_count: int = 25
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
    Tuple of pipeline_definiton_path and parameter_values.
  """
  warnings.warn(
      'Depreciated. Please use get_automl_tabular_pipeline_and_parameters.')

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
      run_distillation=True)


def get_wide_and_deep_trainer_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    transform_config: str,
    learning_rate: float,
    dnn_learning_rate: float,
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
    seed: int = 1,
    eval_steps: int = 0,
    batch_size: int = 100,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: str = '',
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the Wide & Deep training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      'classification' or 'regression'.
    transform_config:
      The path to a GCS file containing the transformations to
      apply.
    learning_rate:
      The learning rate used by the linear optimizer.
    dnn_learning_rate:
      The learning rate for training the deep part of the
      model.
    optimizer_type:
      The type of optimizer to use. Choices are "adam", "ftrl" and
      "sgd" for the Adam, FTRL, and Gradient Descent Optimizers, respectively.
    max_steps:
      Number of steps to run the trainer for.
    max_train_secs:
      Amount of time in seconds to run the trainer for.
    l1_regularization_strength:
      L1 regularization strength for
      optimizer_type="ftrl".
    l2_regularization_strength:
      L2 regularization strength for
      optimizer_type="ftrl".
    l2_shrinkage_regularization_strength:
      L2 shrinkage regularization strength
      for optimizer_type="ftrl".
    beta_1:
      Beta 1 value for optimizer_type="adam".
    beta_2:
      Beta 2 value for optimizer_type="adam".
    hidden_units:
      Hidden layer sizes to use for DNN feature columns, provided in
      comma-separated layers.
    use_wide:
      If set to true, the categorical columns will be used in the wide
      part of the DNN model.
    embed_categories:
      If set to true, the categorical columns will be used
      embedded and used in the deep part of the model. Embedding size is the
      square root of the column cardinality.
    dnn_dropout:
      The probability we will drop out a given coordinate.
    dnn_optimizer_type:
      The type of optimizer to use for the deep part of the
      model. Choices are "adam", "ftrl" and "sgd". for the Adam, FTRL, and
      Gradient Descent Optimizers, respectively.
    dnn_l1_regularization_strength:
      L1 regularization strength for
      dnn_optimizer_type="ftrl".
    dnn_l2_regularization_strength:
      L2 regularization strength for
      dnn_optimizer_type="ftrl".
    dnn_l2_shrinkage_regularization_strength:
      L2 shrinkage regularization
      strength for dnn_optimizer_type="ftrl".
    dnn_beta_1:
      Beta 1 value for dnn_optimizer_type="adam".
    dnn_beta_2:
      Beta 2 value for dnn_optimizer_type="adam".
    enable_profiler:
      Enables profiling and saves a trace during evaluation.
    seed:
      Seed to be used for this run.
    eval_steps:
      Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    batch_size:
      Batch size for training.
    eval_frequency_secs:
      Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      The test fraction.
    weight_column:
      The weight column name.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override:
      The dictionary for overriding training and
        evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation:
      Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'transform_config':
          transform_config,
      'learning_rate':
          learning_rate,
      'dnn_learning_rate':
          dnn_learning_rate,
      'optimizer_type':
          optimizer_type,
      'max_steps':
          max_steps,
      'max_train_secs':
          max_train_secs,
      'l1_regularization_strength':
          l1_regularization_strength,
      'l2_regularization_strength':
          l2_regularization_strength,
      'l2_shrinkage_regularization_strength':
          l2_shrinkage_regularization_strength,
      'beta_1':
          beta_1,
      'beta_2':
          beta_2,
      'hidden_units':
          hidden_units,
      'use_wide':
          use_wide,
      'embed_categories':
          embed_categories,
      'dnn_dropout':
          dnn_dropout,
      'dnn_optimizer_type':
          dnn_optimizer_type,
      'dnn_l1_regularization_strength':
          dnn_l1_regularization_strength,
      'dnn_l2_regularization_strength':
          dnn_l2_regularization_strength,
      'dnn_l2_shrinkage_regularization_strength':
          dnn_l2_shrinkage_regularization_strength,
      'dnn_beta_1':
          dnn_beta_1,
      'dnn_beta_2':
          dnn_beta_2,
      'enable_profiler':
          enable_profiler,
      'seed':
          seed,
      'eval_steps':
          eval_steps,
      'batch_size':
          batch_size,
      'eval_frequency_secs':
          eval_frequency_secs,
      'weight_column':
          weight_column,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'worker_pool_specs_override':
          worker_pool_specs_override,
      'run_evaluation':
          run_evaluation,
      'evaluation_batch_predict_machine_type':
          evaluation_batch_predict_machine_type,
      'evaluation_batch_predict_starting_replica_count':
          evaluation_batch_predict_starting_replica_count,
      'evaluation_batch_predict_max_replica_count':
          evaluation_batch_predict_max_replica_count,
      'evaluation_dataflow_machine_type':
          evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers':
          evaluation_dataflow_max_num_workers,
      'evaluation_dataflow_disk_size_gb':
          evaluation_dataflow_disk_size_gb,
      'dataflow_service_account':
          dataflow_service_account,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }
  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction
  }
  parameter_values.update({
      param: value
      for param, value in data_source_and_split_parameters.items()
      if value is not None
  })
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'wide_and_deep_trainer_pipeline.json')

  return pipeline_definition_path, parameter_values


def get_builtin_algorithm_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    transform_config: str,
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
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the built-in algorithm HyperparameterTuningJob pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    transform_config:
      The path to a GCS file containing the transformations to
      apply.
    study_spec_metric_id:
      Metric to optimize, possible values: [
      'loss', 'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc',
      'precision', 'recall'].
    study_spec_metric_goal:
      Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override:
      List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count:
      The desired total number of trials.
    parallel_trial_count:
      The desired number of trials to run in parallel.
    algorithm:
      Algorithm to train. One of "tabnet" and "wide_and_deep".
    enable_profiler:
      Enables profiling and saves a trace during evaluation.
    seed:
      Seed to be used for this run.
    eval_steps:
      Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs:
      Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      The test fraction.
    weight_column:
      The weight column name.
    max_failed_trial_count:
      The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm:
      The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type:
      Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override:
      The dictionary for overriding training and
        evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation:
      Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  warnings.warn(
      'This method is deprecated. Please use get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters '
      'or get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters instead.'
  )

  if algorithm == 'tabnet':
    return get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(
        project=project,
        location=location,
        root_dir=root_dir,
        target_column=target_column,
        prediction_type=prediction_type,
        transform_config=transform_config,
        study_spec_metric_id=study_spec_metric_id,
        study_spec_metric_goal=study_spec_metric_goal,
        study_spec_parameters_override=study_spec_parameters_override,
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        enable_profiler=enable_profiler,
        seed=seed,
        eval_steps=eval_steps,
        eval_frequency_secs=eval_frequency_secs,
        data_source_csv_filenames=data_source_csv_filenames,
        data_source_bigquery_table_path=data_source_bigquery_table_path,
        predefined_split_key=predefined_split_key,
        timestamp_split_key=timestamp_split_key,
        stratified_split_key=stratified_split_key,
        training_fraction=training_fraction,
        validation_fraction=validation_fraction,
        test_fraction=test_fraction,
        weight_column=weight_column,
        max_failed_trial_count=max_failed_trial_count,
        study_spec_algorithm=study_spec_algorithm,
        study_spec_measurement_selection_type=study_spec_measurement_selection_type,
        stats_and_example_gen_dataflow_machine_type=stats_and_example_gen_dataflow_machine_type,
        stats_and_example_gen_dataflow_max_num_workers=stats_and_example_gen_dataflow_max_num_workers,
        stats_and_example_gen_dataflow_disk_size_gb=stats_and_example_gen_dataflow_disk_size_gb,
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
        evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name)
  elif algorithm == 'wide_and_deep':
    return get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
        project=project,
        location=location,
        root_dir=root_dir,
        target_column=target_column,
        prediction_type=prediction_type,
        transform_config=transform_config,
        study_spec_metric_id=study_spec_metric_id,
        study_spec_metric_goal=study_spec_metric_goal,
        study_spec_parameters_override=study_spec_parameters_override,
        max_trial_count=max_trial_count,
        parallel_trial_count=parallel_trial_count,
        enable_profiler=enable_profiler,
        seed=seed,
        eval_steps=eval_steps,
        eval_frequency_secs=eval_frequency_secs,
        data_source_csv_filenames=data_source_csv_filenames,
        data_source_bigquery_table_path=data_source_bigquery_table_path,
        predefined_split_key=predefined_split_key,
        timestamp_split_key=timestamp_split_key,
        stratified_split_key=stratified_split_key,
        training_fraction=training_fraction,
        validation_fraction=validation_fraction,
        test_fraction=test_fraction,
        weight_column=weight_column,
        max_failed_trial_count=max_failed_trial_count,
        study_spec_algorithm=study_spec_algorithm,
        study_spec_measurement_selection_type=study_spec_measurement_selection_type,
        stats_and_example_gen_dataflow_machine_type=stats_and_example_gen_dataflow_machine_type,
        stats_and_example_gen_dataflow_max_num_workers=stats_and_example_gen_dataflow_max_num_workers,
        stats_and_example_gen_dataflow_disk_size_gb=stats_and_example_gen_dataflow_disk_size_gb,
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
        evaluation_dataflow_max_num_workers=evaluation_dataflow_max_num_workers,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name)
  else:
    raise ValueError(
        'Invalid algorithm provided. Supported values are "tabnet" and "wide_and_deep".'
    )


def get_tabnet_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    transform_config: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: List[Dict[str, Any]],
    max_trial_count: int,
    parallel_trial_count: int,
    enable_profiler: bool = False,
    seed: int = 1,
    eval_steps: int = 0,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the TabNet HyperparameterTuningJob pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    transform_config:
      The path to a GCS file containing the transformations to
      apply.
    study_spec_metric_id:
      Metric to optimize, possible values: [
      'loss', 'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc',
      'precision', 'recall'].
    study_spec_metric_goal:
      Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override:
      List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count:
      The desired total number of trials.
    parallel_trial_count:
      The desired number of trials to run in parallel.
    enable_profiler:
      Enables profiling and saves a trace during evaluation.
    seed:
      Seed to be used for this run.
    eval_steps:
      Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs:
      Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      The test fraction.
    weight_column:
      The weight column name.
    max_failed_trial_count:
      The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm:
      The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type:
      Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override:
      The dictionary for overriding training and
        evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation:
      Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'transform_config':
          transform_config,
      'study_spec_metric_id':
          study_spec_metric_id,
      'study_spec_metric_goal':
          study_spec_metric_goal,
      'study_spec_parameters_override':
          study_spec_parameters_override,
      'max_trial_count':
          max_trial_count,
      'parallel_trial_count':
          parallel_trial_count,
      'enable_profiler':
          enable_profiler,
      'seed':
          seed,
      'eval_steps':
          eval_steps,
      'eval_frequency_secs':
          eval_frequency_secs,
      'weight_column':
          weight_column,
      'max_failed_trial_count':
          max_failed_trial_count,
      'study_spec_algorithm':
          study_spec_algorithm,
      'study_spec_measurement_selection_type':
          study_spec_measurement_selection_type,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'worker_pool_specs_override':
          worker_pool_specs_override,
      'run_evaluation':
          run_evaluation,
      'evaluation_batch_predict_machine_type':
          evaluation_batch_predict_machine_type,
      'evaluation_batch_predict_starting_replica_count':
          evaluation_batch_predict_starting_replica_count,
      'evaluation_batch_predict_max_replica_count':
          evaluation_batch_predict_max_replica_count,
      'evaluation_dataflow_machine_type':
          evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers':
          evaluation_dataflow_max_num_workers,
      'evaluation_dataflow_disk_size_gb':
          evaluation_dataflow_disk_size_gb,
      'dataflow_service_account':
          dataflow_service_account,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }
  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction
  }
  parameter_values.update({
      param: value
      for param, value in data_source_and_split_parameters.items()
      if value is not None
  })

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'tabnet_hyperparameter_tuning_job_pipeline.json')

  return pipeline_definition_path, parameter_values


def get_wide_and_deep_hyperparameter_tuning_job_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    transform_config: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: List[Dict[str, Any]],
    max_trial_count: int,
    parallel_trial_count: int,
    enable_profiler: bool = False,
    seed: int = 1,
    eval_steps: int = 0,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: str = '',
    max_failed_trial_count: int = 0,
    study_spec_algorithm: str = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: str = 'BEST_MEASUREMENT',
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the Wide & Deep algorithm HyperparameterTuningJob pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    transform_config:
      The path to a GCS file containing the transformations to
      apply.
    study_spec_metric_id:
      Metric to optimize, possible values: [
      'loss', 'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc',
      'precision', 'recall'].
    study_spec_metric_goal:
      Optimization goal of the metric, possible values:
      "MAXIMIZE", "MINIMIZE".
    study_spec_parameters_override:
      List of dictionaries representing parameters
      to optimize. The dictionary key is the parameter_id, which is passed to
      training job as a command line argument, and the dictionary value is the
      parameter specification of the metric.
    max_trial_count:
      The desired total number of trials.
    parallel_trial_count:
      The desired number of trials to run in parallel.
    enable_profiler:
      Enables profiling and saves a trace during evaluation.
    seed:
      Seed to be used for this run.
    eval_steps:
      Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    eval_frequency_secs:
      Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      The test fraction.
    weight_column:
      The weight column name.
    max_failed_trial_count:
      The number of failed trials that need to be seen
      before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides
      how many trials must fail before the whole job fails.
    study_spec_algorithm:
      The search algorithm specified for the study. One of
      "ALGORITHM_UNSPECIFIED", "GRID_SEARCH", or "RANDOM_SEARCH".
    study_spec_measurement_selection_type:
      Which measurement to use if/when the
      service automatically selects the final measurement from previously
      reported intermediate measurements. One of "BEST_MEASUREMENT" or
      "LAST_MEASUREMENT".
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override:
      The dictionary for overriding training and
        evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation:
      Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'transform_config':
          transform_config,
      'study_spec_metric_id':
          study_spec_metric_id,
      'study_spec_metric_goal':
          study_spec_metric_goal,
      'study_spec_parameters_override':
          study_spec_parameters_override,
      'max_trial_count':
          max_trial_count,
      'parallel_trial_count':
          parallel_trial_count,
      'enable_profiler':
          enable_profiler,
      'seed':
          seed,
      'eval_steps':
          eval_steps,
      'eval_frequency_secs':
          eval_frequency_secs,
      'weight_column':
          weight_column,
      'max_failed_trial_count':
          max_failed_trial_count,
      'study_spec_algorithm':
          study_spec_algorithm,
      'study_spec_measurement_selection_type':
          study_spec_measurement_selection_type,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'worker_pool_specs_override':
          worker_pool_specs_override,
      'run_evaluation':
          run_evaluation,
      'evaluation_batch_predict_machine_type':
          evaluation_batch_predict_machine_type,
      'evaluation_batch_predict_starting_replica_count':
          evaluation_batch_predict_starting_replica_count,
      'evaluation_batch_predict_max_replica_count':
          evaluation_batch_predict_max_replica_count,
      'evaluation_dataflow_machine_type':
          evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers':
          evaluation_dataflow_max_num_workers,
      'evaluation_dataflow_disk_size_gb':
          evaluation_dataflow_disk_size_gb,
      'dataflow_service_account':
          dataflow_service_account,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }
  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction
  }
  parameter_values.update({
      param: value
      for param, value in data_source_and_split_parameters.items()
      if value is not None
  })

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'wide_and_deep_hyperparameter_tuning_job_pipeline.json')

  return pipeline_definition_path, parameter_values


def get_tabnet_trainer_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    transform_config: str,
    learning_rate: float,
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
    seed: int = 1,
    eval_steps: int = 0,
    batch_size: int = 100,
    eval_frequency_secs: int = 600,
    data_source_csv_filenames: Optional[str] = None,
    data_source_bigquery_table_path: Optional[str] = None,
    predefined_split_key: Optional[str] = None,
    timestamp_split_key: Optional[str] = None,
    stratified_split_key: Optional[str] = None,
    training_fraction: Optional[float] = None,
    validation_fraction: Optional[float] = None,
    test_fraction: Optional[float] = None,
    weight_column: str = '',
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    stats_and_example_gen_dataflow_disk_size_gb: int = 40,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    transform_dataflow_disk_size_gb: int = 40,
    worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    run_evaluation: bool = True,
    evaluation_batch_predict_machine_type:
    str = _EVALUATION_BATCH_PREDICT_MACHINE_TYPE,
    evaluation_batch_predict_starting_replica_count:
    int = _EVALUATION_BATCH_PREDICT_STARTING_REPLICA_COUNT,
    evaluation_batch_predict_max_replica_count:
    int = _EVALUATION_BATCH_PREDICT_MAX_REPLICA_COUNT,
    evaluation_dataflow_machine_type: str = _EVALUATION_DATAFLOW_MACHINE_TYPE,
    evaluation_dataflow_max_num_workers:
    int = _EVALUATION_DATAFLOW_MAX_NUM_WORKERS,
    evaluation_dataflow_disk_size_gb: int = _EVALUATION_DATAFLOW_DISK_SIZE_GB,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the TabNet training pipeline.

  Args:
    project:
      The GCP project that runs the pipeline components.
    location:
      The GCP region that runs the pipeline components.
    root_dir:
      The root GCS directory for the pipeline components.
    target_column:
      The target column name.
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    transform_config:
      The path to a GCS file containing the transformations to
      apply.
    learning_rate:
      The learning rate used by the linear optimizer.
    max_steps:
      Number of steps to run the trainer for.
    max_train_secs:
      Amount of time in seconds to run the trainer for.
    large_category_dim:
      Embedding dimension for categorical feature with large
      number of categories.
    large_category_thresh:
      Threshold for number of categories to apply
      large_category_dim embedding dimension to.
    yeo_johnson_transform:
      Enables trainable Yeo-Johnson power transform.
    feature_dim:
      Dimensionality of the hidden representation in feature
      transformation block.
    feature_dim_ratio:
      The ratio of output dimension (dimensionality of the
      outputs of each decision step) to feature dimension.
    num_decision_steps:
      Number of sequential decision steps.
    relaxation_factor:
      Relaxation factor that promotes the reuse of each feature
      at different decision steps. When it is 1, a feature is enforced to be
      used only at one decision step and as it increases, more flexibility is
      provided to use a feature at multiple decision steps.
    decay_every:
      Number of iterations for periodically applying learning rate
      decaying.
    decay_rate:
      Learning rate decaying.
    gradient_thresh:
      Threshold for the norm of gradients for clipping.
    sparsity_loss_weight:
      Weight of the loss for sparsity regularization
      (increasing it will yield more sparse feature selection).
    batch_momentum:
      Momentum in ghost batch normalization.
    batch_size_ratio:
      The ratio of virtual batch size (size of the ghost batch
      normalization) to batch size.
    num_transformer_layers:
      The number of transformer layers for each decision
      step. used only at one decision step and as it increases, more flexibility
      is provided to use a feature at multiple decision steps.
    num_transformer_layers_ratio:
      The ratio of shared transformer layer to
      transformer layers.
    class_weight:
      The class weight is used to computes a weighted cross entropy
      which is helpful in classify imbalanced dataset. Only used for
      classification.
    loss_function_type:
      Loss function type. Loss function in classification
      [cross_entropy, weighted_cross_entropy, focal_loss], default is
      cross_entropy. Loss function in regression:
        [rmse, mae, mse], default is
      mse.
    alpha_focal_loss:
      Alpha value (balancing factor) in focal_loss function.
      Only used for classification.
    gamma_focal_loss:
      Gamma value (modulating factor) for focal loss for focal
      loss. Only used for classification.
    enable_profiler:
      Enables profiling and saves a trace during evaluation.
    seed:
      Seed to be used for this run.
    eval_steps:
      Number of steps to run evaluation for. If not specified or
      negative, it means run evaluation on the whole validation dataset. If set
      to 0, it means run evaluation for a fixed number of samples.
    batch_size:
      Batch size for training.
    eval_frequency_secs:
      Frequency at which evaluation and checkpointing will
      take place.
    data_source_csv_filenames:
      The CSV data source.
    data_source_bigquery_table_path:
      The BigQuery data source.
    predefined_split_key:
      The predefined_split column name.
    timestamp_split_key:
      The timestamp_split column name.
    stratified_split_key:
      The stratified_split column name.
    training_fraction:
      The training fraction.
    validation_fraction:
      The validation fraction.
    test_fraction:
      The test fraction.
    weight_column:
      The weight column name.
    stats_and_example_gen_dataflow_machine_type:
      The dataflow machine type for
      stats_and_example_gen component.
    stats_and_example_gen_dataflow_max_num_workers:
      The max number of Dataflow
      workers for stats_and_example_gen component.
    stats_and_example_gen_dataflow_disk_size_gb:
      Dataflow worker's disk size in
      GB for stats_and_example_gen component.
    transform_dataflow_machine_type:
      The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers:
      The max number of Dataflow workers for
      transform component.
    transform_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      transform component.
    worker_pool_specs_override:
      The dictionary for overriding training and
        evaluation worker pool specs. The dictionary should be of format
          https://github.com/googleapis/googleapis/blob/4e836c7c257e3e20b1de14d470993a2b1f4736a8/google/cloud/aiplatform/v1beta1/custom_job.proto#L172.
    run_evaluation:
      Whether to run evaluation steps during training.
    evaluation_batch_predict_machine_type:
      The prediction server machine type
      for batch predict components during evaluation.
    evaluation_batch_predict_starting_replica_count:
      The initial number of
      prediction server for batch predict components during evaluation.
    evaluation_batch_predict_max_replica_count:
      The max number of prediction
      server for batch predict components during evaluation.
    evaluation_dataflow_machine_type:
      The dataflow machine type for evaluation
      components.
    evaluation_dataflow_max_num_workers:
      The max number of Dataflow workers for
      evaluation components.
    evaluation_dataflow_disk_size_gb:
      Dataflow worker's disk size in GB for
      evaluation components.
    dataflow_service_account:
      Custom service account to run dataflow jobs.
    dataflow_subnetwork:
      Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips:
      Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:
      The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if not worker_pool_specs_override:
    worker_pool_specs_override = []

  parameter_values = {
      'project':
          project,
      'location':
          location,
      'root_dir':
          root_dir,
      'target_column':
          target_column,
      'prediction_type':
          prediction_type,
      'transform_config':
          transform_config,
      'learning_rate':
          learning_rate,
      'max_steps':
          max_steps,
      'max_train_secs':
          max_train_secs,
      'large_category_dim':
          large_category_dim,
      'large_category_thresh':
          large_category_thresh,
      'yeo_johnson_transform':
          yeo_johnson_transform,
      'feature_dim':
          feature_dim,
      'feature_dim_ratio':
          feature_dim_ratio,
      'num_decision_steps':
          num_decision_steps,
      'relaxation_factor':
          relaxation_factor,
      'decay_every':
          decay_every,
      'decay_rate':
          decay_rate,
      'gradient_thresh':
          gradient_thresh,
      'sparsity_loss_weight':
          sparsity_loss_weight,
      'batch_momentum':
          batch_momentum,
      'batch_size_ratio':
          batch_size_ratio,
      'num_transformer_layers':
          num_transformer_layers,
      'num_transformer_layers_ratio':
          num_transformer_layers_ratio,
      'class_weight':
          class_weight,
      'loss_function_type':
          loss_function_type,
      'alpha_focal_loss':
          alpha_focal_loss,
      'gamma_focal_loss':
          gamma_focal_loss,
      'enable_profiler':
          enable_profiler,
      'seed':
          seed,
      'eval_steps':
          eval_steps,
      'batch_size':
          batch_size,
      'eval_frequency_secs':
          eval_frequency_secs,
      'weight_column':
          weight_column,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'stats_and_example_gen_dataflow_disk_size_gb':
          stats_and_example_gen_dataflow_disk_size_gb,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'transform_dataflow_disk_size_gb':
          transform_dataflow_disk_size_gb,
      'worker_pool_specs_override':
          worker_pool_specs_override,
      'run_evaluation':
          run_evaluation,
      'evaluation_batch_predict_machine_type':
          evaluation_batch_predict_machine_type,
      'evaluation_batch_predict_starting_replica_count':
          evaluation_batch_predict_starting_replica_count,
      'evaluation_batch_predict_max_replica_count':
          evaluation_batch_predict_max_replica_count,
      'evaluation_dataflow_machine_type':
          evaluation_dataflow_machine_type,
      'evaluation_dataflow_max_num_workers':
          evaluation_dataflow_max_num_workers,
      'evaluation_dataflow_disk_size_gb':
          evaluation_dataflow_disk_size_gb,
      'dataflow_service_account':
          dataflow_service_account,
      'dataflow_subnetwork':
          dataflow_subnetwork,
      'dataflow_use_public_ips':
          dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }
  data_source_and_split_parameters = {
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'predefined_split_key': predefined_split_key,
      'timestamp_split_key': timestamp_split_key,
      'stratified_split_key': stratified_split_key,
      'training_fraction': training_fraction,
      'validation_fraction': validation_fraction,
      'test_fraction': test_fraction
  }
  parameter_values.update({
      param: value
      for param, value in data_source_and_split_parameters.items()
      if value is not None
  })
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'tabnet_trainer_pipeline.json')

  return pipeline_definition_path, parameter_values


def get_tabnet_study_spec_parameters_override(
    dataset_size_bucket: str, prediction_type: str,
    training_budget_bucket: str) -> List[Dict[str, Any]]:
  """Get study_spec_parameters_override for a TabNet hyperparameter tuning job.

  Args:
    dataset_size_bucket:
      Size of the dataset. One of "small" (< 1M rows),
      "medium" (1M - 100M rows), or "large" (> 100M rows).
    prediction_type:
      The type of prediction the model is to produce.
      "classification" or "regression".
    training_budget_bucket:
      Bucket of the estimated training budget. One of
      "small" (< $600), "medium" ($600 - $2400), or "large" (> $2400). This
      parameter is only used as a hint for the hyperparameter search space,
      unrelated to the real cost.

  Returns:
    List of study_spec_parameters_override.
  """

  if dataset_size_bucket not in ['small', 'medium', 'large']:
    raise ValueError('Invalid dataset_size_bucket provided. Supported values '
                     ' are "small", "medium" or "large".')
  if training_budget_bucket not in ['small', 'medium', 'large']:
    raise ValueError(
        'Invalid training_budget_bucket provided. Supported values '
        'are "small", "medium" or "large".')

  param_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      f'configs/tabnet_params_{dataset_size_bucket}_data_{training_budget_bucket}_search_space.json'
  )
  with open(param_path, 'r') as f:
    param_content = f.read()
    params = json.loads(param_content)

  if prediction_type == 'regression':
    return _format_tabnet_regression_study_spec_parameters_override(
        params, training_budget_bucket)
  return params


def _format_tabnet_regression_study_spec_parameters_override(
    params: List[Dict[str, Any]],
    training_budget_bucket: str) -> List[Dict[str, Any]]:
  """Get regression study_spec_parameters_override for a TabNet hyperparameter tuning job.

  Args:
    params:
      List of dictionaries representing parameters to optimize. The
      dictionary key is the parameter_id, which is passed to training job as a
      command line argument, and the dictionary value is the parameter
      specification of the metric.
    training_budget_bucket:
      Bucket of the estimated training budget. One of
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
        'alpha_focal_loss', 'gamma_focal_loss', 'class_weight'
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
      'configs/wide_and_deep_params.json')
  with open(param_path, 'r') as f:
    param_content = f.read()
    params = json.loads(param_content)

  return params


def get_model_comparison_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    problem_type: str,
    training_jobs: Dict[str, Dict[str, Any]],
    data_source_csv_filenames: str = '-',
    data_source_bigquery_table_path: str = '-',
    experiment: str = '-',
) -> Tuple[str, Dict[str, Any]]:
  """Returns a compiled model comparison pipeline and formatted parameters.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components
    problem_type: The type of problem being solved. Can be one of: regression,
      binary_classification, multiclass_classification, or forecasting
    training_jobs: A dict mapping name to a dict of training job inputs.
    data_source_csv_filenames: Comman-separated paths to CSVs stored in GCS to
      use as the dataset for all training pipelines. This should be None if
      `data_source_bigquery_table_path` is not None.
    data_source_bigquery_table_path: Path to BigQuery Table to use as the
      dataset for all training pipelines. This should be None if
      `data_source_csv_filenames` is not None.
    experiment: Vertex Experiment to add training pipeline runs to. A new
      Experiment will be created if none is provided.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  parameter_values = {
      'project': project,
      'location': location,
      'root_dir': root_dir,
      'problem_type': problem_type,
      'training_jobs': training_jobs,
      'data_source_csv_filenames': data_source_csv_filenames,
      'data_source_bigquery_table_path': data_source_bigquery_table_path,
      'experiment': experiment,
  }
  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'model_comparison_pipeline.json')
  return pipeline_definition_path, parameter_values
