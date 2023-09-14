# Copyright 2023 The Kubeflow Authors
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
"""Module for supporting Google Vertex AI Hyperparameter Tuning Job Op."""

from typing import Any, Dict, List

from google.cloud.aiplatform import hyperparameter_tuning
from google.cloud.aiplatform_v1.types import study


def serialize_parameters(
    parameters: Dict[str, hyperparameter_tuning._ParameterSpec]
) -> List[Dict[str, Any]]:
  # fmt: off
  """Utility for converting a hyperparameter tuning [ParameterSpec](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/StudySpec#ParameterSpec) into a list of dictionaries.

  Args:
    parameters (Dict[str, hyperparameter_tuning._ParameterSpec]): Dictionary of parameter ids to subclasses of [_ParameterSpec](https://github.com/googleapis/python-aiplatform/blob/1fda4172baaf200414d95e7217bfef0e500cc16a/google/cloud/aiplatform/hyperparameter_tuning.py#L51). Supported subclasses include: `DoubleParameterSpec`, `IntegerParameterSpec`, `CategoricalParameterSpace`, `DiscreteParameterSpec`. Example:

      from google.cloud.aiplatform import hyperparameter_tuning as hpt
      parameters = { 'decay': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear'), 'learning_rate': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear'), 'batch_size': hpt.DiscreteParamterSpec( values=[4, 8, 16, 32, 64, 128], scale='linear') }

  Returns:
    List of `ParameterSpec` dictionaries.
  """
  # fmt: on
  # the to_dict function is used here instead of the to_json function for compatibility with GAPIC
  return [
      study.StudySpec.ParameterSpec.to_dict(
          parameter._to_parameter_spec(parameter_id=parameter_id)
      )
      for parameter_id, parameter in parameters.items()
  ]


def serialize_metrics(metric_spec: Dict[str, str]) -> List[Dict[str, Any]]:
  # fmt: off
  """Utility for converting a hyperparameter tuning [MetricSpec](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/StudySpec#metricspec) into a list of dictionaries.

  Args:
    metric_spec (Dict[str, str]): Dictionary representing metrics to optimize. The dictionary key is the metric_id, which is reported by your training job, and the dictionary value is the optimization goal of the metric (`'minimize'`  or `'maximize'`). Example:

      metrics = {'loss': 'minimize', 'accuracy': 'maximize'}

  Returns: List of `MetricSpec` dictionaries.
  """
  # fmt: on
  return [
      study.StudySpec.MetricSpec.to_dict(
          study.StudySpec.MetricSpec(
              {'metric_id': metric_id, 'goal': goal.upper()}
          )
      )
      for metric_id, goal in metric_spec.items()
  ]
