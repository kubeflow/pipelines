# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Component supporting Google Vertex AI Hyperparameter Tuning Job Op."""

import os
import re
from kfp import components


def get_best_trial(trials: list, study_spec_metrics: list) -> str:
  """Retrieves the best trial from the trials.

  Args:
      trials (list): Required. List representing the intermediate
        JSON representation of the trials from the hyperparameter tuning job.
      study_spec_metrics (list): Required. List serialized from dictionary
        representing the metrics to optimize.
        The dictionary key is the metric_id, which is reported by your training
        job, and the dictionary value is the optimization goal of the metric
        ('minimize' or 'maximize'). example:
        metrics = hyperparameter_tuning_job.serialize_metrics(
            {'loss': 'minimize', 'accuracy': 'maximize'})

  Returns:
      String representing the intermediate JSON representation of the best
      trial from the list of trials.

  Raises:
      RuntimeError: If there are multiple metrics.
  """
  from google.cloud.aiplatform_v1.types import study

  if len(study_spec_metrics) > 1:
    raise RuntimeError('Unable to determine best parameters for multi-objective'
                       ' hyperparameter tuning.')
  trials_list = [study.Trial.from_json(trial) for trial in trials]
  best_trial = None
  goal = study_spec_metrics[0]['goal']
  best_fn = None
  if goal == study.StudySpec.MetricSpec.GoalType.MAXIMIZE:
    best_fn = max
  elif goal == study.StudySpec.MetricSpec.GoalType.MINIMIZE:
    best_fn = min
  best_trial = best_fn(
      trials_list, key=lambda trial: trial.final_measurement.metrics[0].value)

  return study.Trial.to_json(best_trial)


if __name__ == "__main__":
  if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
    rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
    os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

  get_best_trial_op = components.create_component_from_func(
      get_best_trial,
      base_image="python:3.10",
      packages_to_install=["google-cloud-aiplatform==1.18.3"],
      output_component_file="component.yaml",
  )
