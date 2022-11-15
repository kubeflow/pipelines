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


def is_metric_beyond_threshold(trial: str, study_spec_metrics: list,
                              threshold: float) -> str:
  """Determines if the metric of the best trial beyond the threshold given.

  Args:
      trial (str): Required. The intermediate JSON representation of a
        hyperparameter tuning job trial.
      study_spec_metrics (list): Required. List serialized from dictionary
        representing the metrics to optimize.
        The dictionary key is the metric_id, which is reported by your training
        job, and the dictionary value is the optimization goal of the metric
        ('minimize' or 'maximize'). example:
        metrics = hyperparameter_tuning_job.serialize_metrics(
            {'loss': 'minimize', 'accuracy': 'maximize'})
      threshold (float): Required. Threshold to compare metric against.

  Returns:
      "true" if metric is beyond the threshold, otherwise "false"

  Raises:
      RuntimeError: If there are multiple metrics.
  """
  from google.cloud.aiplatform_v1.types import study

  if len(study_spec_metrics) > 1:
    raise RuntimeError('Unable to determine best parameters for multi-objective'
                       ' hyperparameter tuning.')
  trial_proto = study.Trial.from_json(trial)
  val = trial_proto.final_measurement.metrics[0].value
  goal = study_spec_metrics[0]['goal']

  is_beyond_threshold = False
  if goal == study.StudySpec.MetricSpec.GoalType.MAXIMIZE:
    is_beyond_threshold = val > threshold
  elif goal == study.StudySpec.MetricSpec.GoalType.MINIMIZE:
    is_beyond_threshold = val < threshold

  return 'true' if is_beyond_threshold else 'false'


if __name__ == "__main__":
  if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
    rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
    os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

  is_metric_beyond_threshold_op = components.create_component_from_func(
      is_metric_beyond_threshold,
      base_image="python:3.10",
      packages_to_install=["google-cloud-aiplatform==1.18.3"],
      output_component_file="component.yaml",
  )
