# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Component for running LLM Batch Prediction jobs side-by-side."""

import os
from typing import Any, Dict, List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  """Determines the image URI to create a container from."""
  return os.environ.get(
      'AUTOSXS_IMAGE_OVERRIDE'
  ) or utils.get_default_image_uri('autosxs')


# pylint: disable=unused-argument,dangerous-default-value
@dsl.container_component
def batch_prediction_pairwise(
    display_name: str,
    evaluation_dataset: str,
    id_columns: List[str],
    task: str,
    autorater_prompt_parameters: Dict[str, Dict[str, str]],
    response_column_a: str,
    response_column_b: str,
    preprocessed_evaluation_dataset: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    preprocessed_evaluation_dataset_uri: dsl.OutputPath(str),  # pylint: disable=unused-argument # pytype: disable=invalid-annotation
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata: dsl.OutputPath(Dict[str, Any]),  # pytype: disable=invalid-annotation
    model_a: str = '',
    model_b: str = '',
    model_a_prompt_parameters: Dict[str, Dict[str, str]] = {},
    model_b_prompt_parameters: Dict[str, Dict[str, str]] = {},
    model_a_parameters: Dict[str, str] = {},
    model_b_parameters: Dict[str, str] = {},
    human_preference_column: str = '',
    experimental_args: Dict[str, Any] = {},
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    encryption_spec_key_name: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Runs up to two LLM Batch Prediction jobs side-by-side.

  Args:
    display_name: Display name for the batch prediction job.
    evaluation_dataset: GCS or BigQuery URIs representing a dataset of prompts
      and responses.
    id_columns: The columns which distinguish unique evaluation examples.
    task: Task to evaluate.
    autorater_prompt_parameters: Map of autorater prompt template parameters to
      columns or templates.
    response_column_a: The column containing responses for model a.
    response_column_b: The column containing responses for model b.
    model_a: A fully-qualified model resource name
      (`projects/{project}/locations/{location}/models/{model}@{version}`) or
      publisher model resource name (`publishers/{publisher}/models/{model}`).
      This parameter is optional if Model A responses are specified.
    model_b: A fully-qualified model resource name
      (`projects/{project}/locations/{location}/models/{model}@{version}`) or
      publisher model resource name (`publishers/{publisher}/models/{model}`).
      This parameter is optional if Model B responses are specified.
    model_a_prompt_parameters: Map of model A prompt template parameters to
      columns or templates.
    model_b_prompt_parameters: Map of model B prompt template parameters to
      columns or templates.
    model_a_parameters: The parameters that govern the predictions from model A,
      such as temperature or maximum output tokens.
    model_b_parameters: The parameters that govern the predictions from model B,
      such as temperature or maximum output tokens.
    human_preference_column: The column containing ground truths. The default
      value is an empty string if not be provided by users.
    experimental_args: Experimentally released arguments. Subject to change.
    project: Project used to run batch prediction jobs.
    location: Location used to run batch prediction jobs.
    encryption_spec_key_name: Customer-managed encryption key options. If this
      is set, then all resources created by the component will be encrypted with
      the provided encryption key.

  Returns:
    preprocessed_evaluation_dataset: Dataset of the table containing the inputs
      expected by the Arbiter.
    preprocessed_evaluation_dataset_uri: URI of the table containing the inputs
      expected by the Arbiter.
    gcp_resources: Tracker for GCP resources created by this component.
    metadata_path: Path to write the object that stores computed metrics
      metadata for the task preprocess component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='batch_prediction_pairwise',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'batch_prediction_sxs',
              f'--display_name={display_name}',
              f'--evaluation_dataset={evaluation_dataset}',
              (
                  '--id_columns='
                  "{{$.inputs.parameters['id_columns'].json_escape[0]}}"
              ),
              f'--task={task}',
              f'--project={project}',
              f'--location={location}',
              f'--model_a={model_a}',
              f'--model_b={model_b}',
              (
                  '--model_a_prompt_parameters='
                  "{{$.inputs.parameters['model_a_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
              (
                  '--model_b_prompt_parameters='
                  "{{$.inputs.parameters['model_b_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
              (
                  '--autorater_prompt_parameters='
                  "{{$.inputs.parameters['autorater_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
              f'--response_column_a={response_column_a}',
              f'--response_column_b={response_column_b}',
              (
                  '--model_a_parameters='
                  "{{$.inputs.parameters['model_a_parameters'].json_escape[0]}}"
              ),
              (
                  '--model_b_parameters='
                  "{{$.inputs.parameters['model_b_parameters'].json_escape[0]}}"
              ),
              (
                  '--experimental_args='
                  "{{$.inputs.parameters['experimental_args'].json_escape[0]}}"
              ),
              f'--human_preference_column={human_preference_column}',
              f'--staging_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}',
              f'--preprocessed_evaluation_dataset_uri={preprocessed_evaluation_dataset_uri}',
              f'--metadata_path={metadata}',
              f'--kms_key_name={encryption_spec_key_name}',
              f'--gcp_resources_path={gcp_resources}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
