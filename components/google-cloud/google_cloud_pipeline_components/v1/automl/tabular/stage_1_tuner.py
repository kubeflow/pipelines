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

"""AutoML Tabular Stage 1 Tuner component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def automl_tabular_stage_1_tuner(
    project: str,
    location: str,
    root_dir: str,
    num_selected_trials: int,
    deadline_hours: float,
    num_parallel_trials: int,
    single_run_max_secs: int,
    metadata: Input[Artifact],
    transform_output: Input[Artifact],
    materialized_train_split: Input[Artifact],
    materialized_eval_split: Input[Artifact],
    gcp_resources: dsl.OutputPath(str),
    tuning_result_output: Output[Artifact],
    execution_metrics: dsl.OutputPath(dict),
    study_spec_parameters_override: Optional[list] = [],
    worker_pool_specs_override_json: Optional[list] = [],
    reduce_search_space_mode: Optional[str] = 'regular',
    num_selected_features: Optional[int] = 0,
    disable_early_stopping: Optional[bool] = False,
    feature_ranking: Optional[Input[Artifact]] = None,
    tune_feature_selection_rate: Optional[bool] = False,
    encryption_spec_key_name: Optional[str] = '',
    run_distillation: Optional[bool] = False,
):
  # fmt: off
  """Searches AutoML Tabular architectures and selects the top trials.

  Args:
      project: Project to run Cross-validation trainer.
      location: Location for running the Cross-validation trainer.
      root_dir: The Cloud Storage location to store the output.
      study_spec_parameters_override: JSON study spec. E.g., [{"parameter_id": "model_type","categorical_value_spec": {"values": ["nn"]}}]
      worker_pool_specs_override_json: JSON worker pool specs. E.g., [{"machine_spec": {"machine_type": "n1-standard-16"}},{},{},{"machine_spec": {"machine_type": "n1-standard-16"}}]
      reduce_search_space_mode: The reduce search space mode. Possible values: "regular" (default), "minimal", "full".
      num_selected_trials: Number of selected trials. The number of weak learners in the final model is 5 * num_selected_trials.
      num_selected_features: Number of selected features. The number of features to learn in the NN models.
      deadline_hours: Number of hours the cross-validation trainer should run.
      disable_early_stopping: True if disable early stopping. Default value is false.
      num_parallel_trials: Number of parallel training trials.
      single_run_max_secs: Max number of seconds each training trial runs.
      metadata: The tabular example gen metadata.
      transform_output: The transform output artifact.
      materialized_train_split: The materialized train split.
      materialized_eval_split: The materialized eval split.
      encryption_spec_key_name: Customer-managed encryption key.
      run_distillation: True if in distillation mode. The default value is false.

  Returns:
      gcp_resources: GCP resources created by this component. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
      tuning_result_output: The trained model and architectures.
      execution_metrics: Core metrics in dictionary of component execution.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.44',
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
          '--gcp_resources',
          gcp_resources,
          '--payload',
          dsl.ConcatPlaceholder(
              items=[
                  (
                      '{"display_name":'
                      f' "automl-tabular-stage-1-tuner-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20250620_0525',
                  '", "args": ["l2l_stage_1_tuner", "--transform_output_path=',
                  transform_output.uri,
                  '", "--training_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20250620_0525',
                  '", "--feature_selection_result_path=',
                  feature_ranking.uri,
                  '", "--disable_early_stopping=',
                  disable_early_stopping,
                  '", "--tune_feature_selection_rate=',
                  tune_feature_selection_rate,
                  '", "--reduce_search_space_mode=',
                  reduce_search_space_mode,
                  (
                      f'", "--component_id={dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "--training_base_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/train",'
                      ' "--num_parallel_trial='
                  ),
                  num_parallel_trials,
                  '", "--single_run_max_secs=',
                  single_run_max_secs,
                  '", "--deadline_hours=',
                  deadline_hours,
                  '", "--num_selected_trials=',
                  num_selected_trials,
                  '", "--num_selected_features=',
                  num_selected_features,
                  '", "--lro_job_info=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/lro",'
                      ' "--error_file_path='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.pb",'
                      ' "--metadata_path='
                  ),
                  metadata.uri,
                  '", "--materialized_train_split=',
                  materialized_train_split.uri,
                  '", "--materialized_eval_split=',
                  materialized_eval_split.uri,
                  '", "--is_distill=',
                  run_distillation,
                  '", "--tuning_result_output_path=',
                  tuning_result_output.uri,
                  '", "--kms_key_name=',
                  encryption_spec_key_name,
                  '", "--gcp_resources_path=',
                  gcp_resources,
                  '", "--execution_metrics_path=',
                  execution_metrics,
                  (
                      '", "--use_json=true", "--log_level=ERROR",'
                      ' "--executor_input={{$.json_escape[1]}}"]}}]}}'
                  ),
              ]
          ),
      ],
  )
