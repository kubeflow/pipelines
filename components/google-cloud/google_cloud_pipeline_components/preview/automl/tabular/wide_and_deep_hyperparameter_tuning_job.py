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

"""AutoML Wide and Deep Hyperparameter Tuning component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input


@dsl.container_component
def wide_and_deep_hyperparameter_tuning_job(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: list,
    max_trial_count: int,
    parallel_trial_count: int,
    instance_baseline: Input[Artifact],
    metadata: Input[Artifact],
    materialized_train_split: Input[Artifact],
    materialized_eval_split: Input[Artifact],
    transform_output: Input[Artifact],
    training_schema_uri: Input[Artifact],
    gcp_resources: dsl.OutputPath(str),
    instance_schema_uri: dsl.OutputPath(str),
    prediction_schema_uri: dsl.OutputPath(str),
    trials: dsl.OutputPath(str),
    prediction_docker_uri_output: dsl.OutputPath(str),
    execution_metrics: dsl.OutputPath(dict),
    weight_column: Optional[str] = '',
    enable_profiler: Optional[bool] = False,
    cache_data: Optional[str] = 'auto',
    seed: Optional[int] = 1,
    eval_steps: Optional[int] = 0,
    eval_frequency_secs: Optional[int] = 600,
    max_failed_trial_count: Optional[int] = 0,
    study_spec_algorithm: Optional[str] = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: Optional[str] = 'BEST_MEASUREMENT',
    training_machine_spec: Optional[dict] = {'machine_type': 'c2-standard-16'},
    training_disk_spec: Optional[dict] = {
        'boot_disk_type': 'pd-ssd',
        'boot_disk_size_gb': 100,
    },
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Tunes Wide & Deep hyperparameters using Vertex HyperparameterTuningJob API.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region that runs the pipeline components.
      root_dir: The root GCS directory for the pipeline components.
      target_column: The target column name.
      prediction_type: The type of prediction the model is to produce. "classification" or "regression".
      weight_column: The weight column name.
      enable_profiler: Enables profiling and saves a trace during evaluation.
      cache_data: Whether to cache data or not. If set to 'auto', caching is determined based on the dataset size.
      seed: Seed to be used for this run.
      eval_steps: Number of steps to run evaluation for. If not specified or negative, it means run evaluation on the whole validation dataset. If set to 0, it means run evaluation for a fixed number of samples.
      eval_frequency_secs: Frequency at which evaluation and checkpointing will take place.
      study_spec_metric_id: Metric to optimize, possible values: [ 'loss', 'average_loss', 'rmse', 'mae', 'mql', 'accuracy', 'auc', 'precision', 'recall'].
      study_spec_metric_goal: Optimization goal of the metric, possible values: "MAXIMIZE", "MINIMIZE".
      study_spec_parameters_override: List of dictionaries representing parameters to optimize. The dictionary key is the parameter_id, which is passed to training job as a command line argument, and the dictionary value is the parameter specification of the metric.
      max_trial_count: The desired total number of trials.
      parallel_trial_count: The desired number of trials to run in parallel.
      max_failed_trial_count: The number of failed trials that need to be seen before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides how many trials must fail before the whole job fails.
      study_spec_algorithm: The search algorithm specified for the study. One of 'ALGORITHM_UNSPECIFIED', 'GRID_SEARCH', or 'RANDOM_SEARCH'.
      study_spec_measurement_selection_type: Which measurement to use if/when the service automatically selects the final measurement from previously reported intermediate measurements. One of "BEST_MEASUREMENT" or "LAST_MEASUREMENT".
      training_machine_spec: The training machine spec. See https://cloud.google.com/compute/docs/machine-types for options.
      training_disk_spec: The training disk spec.
      instance_baseline: The path to a JSON file for baseline values.
      metadata: Amount of time in seconds to run the trainer for.
      materialized_train_split: The path to the materialized train split.
      materialized_eval_split: The path to the materialized validation split.
      transform_output: The path to transform output.
      training_schema_uri: The path to the training schema.
      encryption_spec_key_name: The KMS key name.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the custom training job.
      instance_schema_uri: The path to the instance schema.
      prediction_schema_uri: The path to the prediction schema.
      trials: The path to the hyperparameter tuning trials
      prediction_docker_uri_output: The URI of the prediction container.
      execution_metrics: Core metrics in dictionary of hyperparameter tuning job execution.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.44',
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.hyperparameter_tuning_job.launcher',
      ],
      args=[
          '--type',
          'HyperparameterTuningJobWithMetrics',
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--execution_metrics',
          execution_metrics,
          '--payload',
          dsl.ConcatPlaceholder(
              items=[
                  (
                      '{"display_name":'
                      f' "wide-and-deep-hyperparameter-tuning-job-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  '"}, "study_spec": {"metrics": [{"metric_id": "',
                  study_spec_metric_id,
                  '", "goal": "',
                  study_spec_metric_goal,
                  '"}], "parameters": ',
                  study_spec_parameters_override,
                  ', "algorithm": "',
                  study_spec_algorithm,
                  '", "measurement_selection_type": "',
                  study_spec_measurement_selection_type,
                  '"}, "max_trial_count": ',
                  max_trial_count,
                  ', "parallel_trial_count": ',
                  parallel_trial_count,
                  ', "max_failed_trial_count": ',
                  max_failed_trial_count,
                  (
                      ', "trial_job_spec": {"worker_pool_specs":'
                      ' [{"replica_count":"'
                  ),
                  '1',
                  '", "machine_spec": ',
                  training_machine_spec,
                  ', "disk_spec": ',
                  training_disk_spec,
                  ', "container_spec": {"image_uri":"',
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/wide-and-deep-training:20240710_0625',
                  '", "args": ["--target_column=',
                  target_column,
                  '", "--weight_column=',
                  weight_column,
                  '", "--model_type=',
                  prediction_type,
                  '", "--prediction_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:20240710_0625',
                  '", "--prediction_docker_uri_artifact_path=',
                  prediction_docker_uri_output,
                  '", "--baseline_path=',
                  instance_baseline.uri,
                  '", "--metadata_path=',
                  metadata.uri,
                  '", "--transform_output_path=',
                  transform_output.uri,
                  '", "--training_schema_path=',
                  training_schema_uri.uri,
                  '", "--instance_schema_path=',
                  instance_schema_uri,
                  '", "--prediction_schema_path=',
                  prediction_schema_uri,
                  '", "--trials_path=',
                  trials,
                  '", "--job_dir=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/train",'
                      ' "--training_data_path='
                  ),
                  materialized_train_split.uri,
                  '", "--validation_data_path=',
                  materialized_eval_split.uri,
                  '", "--enable_profiler=',
                  enable_profiler,
                  '", "--cache_data=',
                  cache_data,
                  '", "--measurement_selection_type=',
                  study_spec_measurement_selection_type,
                  '", "--metric_goal=',
                  study_spec_metric_goal,
                  '", "--seed=',
                  seed,
                  '", "--eval_steps=',
                  eval_steps,
                  '", "--eval_frequency_secs=',
                  eval_frequency_secs,
                  '"]}}]}}',
              ]
          ),
      ],
  )
