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
"""Module for supporting Google Vertex AI Hyperparameter Tuning Job Op."""

from google.cloud.aiplatform_v1.types import study

def serialize_parameters(parameters: dict) -> list:
  """Serializes the hyperparameter tuning parameter spec to dictionary format.

  Args:
      parameters (Dict[str, hyperparameter_tuning._ParameterSpec]): Dictionary
        representing parameters to optimize. The dictionary key is the
        parameter_id, which is passed into your training job as a command line
        key word argument, and the dictionary value is the parameter
        specification of the metric. from google.cloud.aiplatform
        import hyperparameter_tuning as hpt
        parameters={
            'decay': hpt.DoubleParameterSpec(min=1e-7, max=1, scale='linear'),
            'learning_rate': hpt.DoubleParameterSpec(min=1e-7, max=1,
                scale='linear')
            'batch_size': hpt.DiscreteParamterSpec(values=[4, 8, 16, 32, 64,
                128], scale='linear') } Supported parameter specifications can
                be found in aiplatform.hyperparameter_tuning.
        These parameter specification are currently supported:
          DoubleParameterSpec, IntegerParameterSpec,
          CategoricalParameterSpace, DiscreteParameterSpec
        Note: The to_dict function is used here instead of the to_json
        function for compatibility with GAPIC.

  Returns:
      List containing an intermediate JSON representation of the parameter spec

  """
  return [
      study.StudySpec.ParameterSpec.to_dict(
          parameter._to_parameter_spec(parameter_id=parameter_id))
      for parameter_id, parameter in parameters.items()
  ]


def serialize_metrics(metric_spec: dict) -> list:
  """Serializes a metric spec to dictionary format.

  Args:
      metric_spec (Dict[str, str]): Required. Dictionary representing metrics
        to optimize. The dictionary key is the metric_id, which is reported by
        your training job, and the dictionary value is the optimization goal of
        the metric ('minimize' or 'maximize'). Example:
        metrics = {'loss': 'minimize', 'accuracy': 'maximize'}

  Returns:
      List containing an intermediate JSON representation of the metric spec

  """
  return [
      study.StudySpec.MetricSpec.to_dict(
          study.StudySpec.MetricSpec({
              'metric_id': metric_id,
              'goal': goal.upper()
          })) for metric_id, goal in metric_spec.items()
  ]
