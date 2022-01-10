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
from kfp.v2 import dsl


@dsl.component(
    packages_to_install=[
        'google-cloud-aiplatform', 'google-cloud-pipeline-components',
        'protobuf'
    ],
    base_image='python:3.7')
def GetTrialsOp(gcp_resources: str) -> list:
  """Retrieves the best trial from the trials.

  Args:
      gcp_resources (str): Proto tracking the hyperparameter tuning job.

  Returns:
      List of strings representing the intermediate JSON representation of the
      trials from the hyperparameter tuning job.
  """
  from google.cloud import aiplatform
  from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
  from google.protobuf.json_format import Parse
  from google.cloud.aiplatform_v1.types import study

  api_endpoint_suffix = '-aiplatform.googleapis.com'
  gcp_resources_proto = Parse(gcp_resources, GcpResources())
  gcp_resources_split = gcp_resources_proto.resources[0].resource_uri.partition(
      'projects')
  resource_name = gcp_resources_split[1] + gcp_resources_split[2]
  prefix_str = gcp_resources_split[0]
  prefix_str = prefix_str[:prefix_str.find(api_endpoint_suffix)]
  api_endpoint = prefix_str[(prefix_str.rfind('//') + 2):] + api_endpoint_suffix

  client_options = {'api_endpoint': api_endpoint}
  job_client = aiplatform.gapic.JobServiceClient(client_options=client_options)
  response = job_client.get_hyperparameter_tuning_job(name=resource_name)

  return [study.Trial.to_json(trial) for trial in response.trials]


@dsl.component(
    packages_to_install=['google-cloud-aiplatform'], base_image='python:3.7')
def GetBestTrialOp(trials: list, study_spec_metrics: list) -> str:
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


@dsl.component(
    packages_to_install=['google-cloud-aiplatform'], base_image='python:3.7')
def GetBestHyperparametersOp(trials: list, study_spec_metrics: list) -> list:
  """Retrieves the best hyperparameters based on the trials.

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
      List representing the intermediate JSON representation of the best
      hyperparameters from the best trial in the list of trials.

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

  return [
      study.Trial.Parameter.to_json(param) for param in best_trial.parameters
  ]


@dsl.component(
    packages_to_install=['google-cloud-aiplatform'], base_image='python:3.7')
def GetHyperparametersOp(trial: str) -> list:
  """Retrieves the hyperparameters from the given trial.

  Args:
      trial (str): Required. The intermediate JSON representation of a
        hyperparameter tuning job trial.

  Returns:
      List representing the intermediate JSON representation of the
      hyperparameters from the trial.
  """
  from google.cloud.aiplatform_v1.types import study

  trial_proto = study.Trial.from_json(trial)

  return [
      study.Trial.Parameter.to_json(param) for param in trial_proto.parameters
  ]


@dsl.component(
    packages_to_install=['google-cloud-aiplatform'], base_image='python:3.7')
def IsMetricBeyondThresholdOp(trial: str, study_spec_metrics: list,
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


@dsl.component(
    packages_to_install=['google-cloud-aiplatform'], base_image='python:3.7')
def GetWorkerPoolSpecsOp(best_hyperparameters: list,
                         worker_pool_specs: list) -> list:
  """Constructs worker_pool_specs based on the best hyperparameters.

  Args:
      best_hyperparameters (list): Required. List representing the intermediate
        JSON representation of the best hyperparameters from the
        hyperparameter tuning job.
      worker_pool_specs (list): Required. The spec of the worker pools
        including machine type and Docker image. All worker pools except the
        first one are optional and can be skipped by providing an empty value.

  Returns:
      List containing an intermediate JSON representation of the
      worker_pool_specs updated with the best hyperparameters as arguments
      in the container_spec.

  """
  from google.cloud.aiplatform_v1.types import study

  for worker_pool_spec in worker_pool_specs:
    if 'args' not in worker_pool_spec['container_spec']:
      worker_pool_spec['container_spec']['args'] = []
    for param in best_hyperparameters:
      p = study.Trial.Parameter.from_json(param)
      worker_pool_spec['container_spec']['args'].append(
          f'--{p.parameter_id}={p.value}')

  return worker_pool_specs


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
