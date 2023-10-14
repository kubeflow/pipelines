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

"""AutoML XGBoost Hyperparameter Tuning component spec."""

from typing import Optional

from kfp import dsl


@dsl.container_component
def xgboost_hyperparameter_tuning_job(
    project: str,
    location: str,
    study_spec_metric_id: str,
    study_spec_metric_goal: str,
    study_spec_parameters_override: list,
    max_trial_count: int,
    parallel_trial_count: int,
    worker_pool_specs: list,
    gcp_resources: dsl.OutputPath(str),
    max_failed_trial_count: Optional[int] = 0,
    study_spec_algorithm: Optional[str] = 'ALGORITHM_UNSPECIFIED',
    study_spec_measurement_selection_type: Optional[str] = 'BEST_MEASUREMENT',
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Tunes XGBoost hyperparameters using Vertex HyperparameterTuningJob API.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region that runs the pipeline components.
      study_spec_metric_id: Metric to optimize. For options, please look under 'eval_metric' at https://xgboost.readthedocs.io/en/stable/parameter.html#learning-task-parameters.
      study_spec_metric_goal: Optimization goal of the metric, possible values: "MAXIMIZE", "MINIMIZE".
      study_spec_parameters_override: List of dictionaries representing parameters to optimize. The dictionary key is the parameter_id, which is passed to training job as a command line argument, and the dictionary value is the parameter specification of the metric.
      max_trial_count: The desired total number of trials.
      parallel_trial_count: The desired number of trials to run in parallel.
      max_failed_trial_count: The number of failed trials that need to be seen before failing the HyperparameterTuningJob. If set to 0, Vertex AI decides how many trials must fail before the whole job fails.
      study_spec_algorithm: The search algorithm specified for the study. One of 'ALGORITHM_UNSPECIFIED', 'GRID_SEARCH', or 'RANDOM_SEARCH'.
      study_spec_measurement_selection_type: Which measurement to use if/when the service automatically selects the final measurement from previously reported intermediate measurements. One of "BEST_MEASUREMENT" or "LAST_MEASUREMENT".
      worker_pool_specs: The worker pool specs.
      encryption_spec_key_name: The KMS key name.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the custom training job.
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
          'HyperparameterTuningJob',
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
                      f' "xgboost-hyperparameter-tuning-job-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
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
                  ', "trial_job_spec": {"worker_pool_specs": ',
                  worker_pool_specs,
                  '}}',
              ]
          ),
      ],
  )
