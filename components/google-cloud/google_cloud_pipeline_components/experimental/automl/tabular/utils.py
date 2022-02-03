"""Util functions for AutoML Tabular pipeline."""

import json
import math
import os
import pathlib
from typing import Any, Dict, Tuple, Optional

_DEFAULT_NUM_PARALLEL_TRAILS = 35
_DEFAULT_STAGE_2_NUM_SELECTED_TRAILS = 5
_NUM_FOLDS = 5


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
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips evaluation.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the Model is to produce.
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
      the default
      subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
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
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(), 'skip_evaluation_pipeline.json')
  return pipeline_definition_path, parameter_values


def get_feature_selection_skip_evaluation_pipeline_and_parameters(
    project: str,
    location: str,
    root_dir: str,
    target_column_name: str,
    prediction_type: str,
    optimization_objective: str,
    transformations: Dict[str, Any],
    split_spec: Dict[str, Any],
    data_source: Dict[str, Any],
    max_selected_features: int,
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
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips evaluation.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the Model is to produce.
      "classification" or "regression".
    optimization_objective: For binary classification, "maximize-au-roc",
      "minimize-log-loss", "maximize-au-prc", "maximize-precision-at-recall", or
      "maximize-recall-at-precision". For multi class classification,
      "minimize-log-loss". For regression, "minimize-rmse", "minimize-mae", or
      "minimize-rmsle".
    transformations: The transformations to apply.
    split_spec: The split spec.
    data_source: The data source.
    max_selected_features: number of features to be selected.
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
      the default
      subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definition_path and parameter_values.
  """
  _, parameter_values = get_skip_evaluation_pipeline_and_parameters(
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
      dataflow_use_public_ips=dataflow_use_public_ips,
      dataflow_subnetwork=dataflow_subnetwork,
      encryption_spec_key_name=encryption_spec_key_name)

  parameter_values['max_selected_features'] = max_selected_features

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'feature_selection_skip_evaluation_pipeline.json')
  return pipeline_definition_path, parameter_values


def get_skip_architecture_search_pipeline_and_parameters(
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
    stage_1_tuning_result_artifact_uri: str,
    stage_2_num_parallel_trials: int = _DEFAULT_NUM_PARALLEL_TRAILS,
    stage_2_num_selected_trials: int = _DEFAULT_STAGE_2_NUM_SELECTED_TRAILS,
    weight_column_name: str = '',
    optimization_objective_recall_value: float = -1,
    optimization_objective_precision_value: float = -1,
    cv_trainer_worker_pool_specs_override: Optional[Dict[str, Any]] = None,
    export_additional_model_without_custom_ops: bool = False,
    stats_and_example_gen_dataflow_machine_type: str = 'n1-standard-16',
    stats_and_example_gen_dataflow_max_num_workers: int = 25,
    transform_dataflow_machine_type: str = 'n1-standard-16',
    transform_dataflow_max_num_workers: int = 25,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '') -> Tuple[str, Dict[str, Any]]:
  """Get the AutoML Tabular training pipeline that skips architecture search.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    root_dir: The root GCS directory for the pipeline components.
    target_column_name: The target column name.
    prediction_type: The type of prediction the Model is to produce.
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
    stage_1_tuning_result_artifact_uri: The stage 1 tuning result artifact GCS
      URI.
    stage_2_num_parallel_trials: Number of parallel trail for stage 2.
    stage_2_num_selected_trials: Number of selected trials for stage 2.
    weight_column_name: The weight column name.
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
    transform_dataflow_machine_type: The dataflow machine type for transform
      component.
    transform_dataflow_max_num_workers: The max number of Dataflow workers for
      transform component.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default
      subnetwork will be used. Example:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: The KMS key name.

  Returns:
    Tuple of pipeline_definiton_path and parameter_values.
  """
  if stage_2_num_parallel_trials <= 0:
    stage_2_num_parallel_trials = _DEFAULT_NUM_PARALLEL_TRAILS

  stage_2_deadline_hours = train_budget_milli_node_hours / 1000.0
  stage_2_single_run_max_secs = int(stage_2_deadline_hours * 3600.0 / 1.3)

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
      'stage_1_tuning_result_artifact_uri':
          stage_1_tuning_result_artifact_uri,
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
      'cv_trainer_worker_pool_specs_override':
          input_dictionary_to_parameter(cv_trainer_worker_pool_specs_override),
      'export_additional_model_without_custom_ops':
          export_additional_model_without_custom_ops,
      'stats_and_example_gen_dataflow_machine_type':
          stats_and_example_gen_dataflow_machine_type,
      'stats_and_example_gen_dataflow_max_num_workers':
          stats_and_example_gen_dataflow_max_num_workers,
      'transform_dataflow_machine_type':
          transform_dataflow_machine_type,
      'transform_dataflow_max_num_workers':
          transform_dataflow_max_num_workers,
      'dataflow_subnetwork': dataflow_subnetwork,
      'dataflow_use_public_ips': dataflow_use_public_ips,
      'encryption_spec_key_name':
          encryption_spec_key_name,
  }

  pipeline_definition_path = os.path.join(
      pathlib.Path(__file__).parent.resolve(),
      'skip_architecture_search_pipeline.json')
  return pipeline_definition_path, parameter_values
