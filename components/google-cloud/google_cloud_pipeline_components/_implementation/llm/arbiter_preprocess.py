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
"""KFP Container component for preprocessing predictions for the Arbiter."""

import os
from typing import Dict, List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  """Determines the image URI to create a container from."""
  return (
      os.environ.get('AUTOSXS_IMAGE_OVERRIDE')
      or utils.get_default_image_uri('autosxs'))


# pylint: disable=unused-argument,dangerous-default-value
@dsl.container_component
def arbiter_preprocess(
    evaluation_dataset: str,
    id_columns: List[str],
    response_column_a: str,
    response_column_b: str,
    task: str,
    is_bp_output_a: bool,
    is_bp_output_b: bool,
    autorater_prompt_parameters: Dict[str, Dict[str, str]],
    preprocessed_evaluation_dataset: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    preprocessed_evaluation_dataset_uri: dsl.OutputPath(str),  # pylint: disable=unused-argument # pytype: disable=invalid-annotation
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    prediction_uris_a: str = '',
    prediction_uris_b: str = '',
    model_a_prompt_parameters: Dict[str, Dict[str, str]] = {},
    model_b_prompt_parameters: Dict[str, Dict[str, str]] = {},
    human_preference_column: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Preprocesses predictions tables for the AutoSxS Arbiter.

  Args:
    evaluation_dataset: GCS or BigQuery URIs representing a dataset of prompts
      and responses.
    id_columns: The columns which distinguish unique evaluation examples.
    response_column_a: The column containing responses for model a.
    response_column_b: The column containing responses for model a.
    task: Task to evaluate.
    output_path: Path to write the path where preprocessed predictions are
      stored.
    is_bp_output_a: If True, the prediction URIs will be parsed as if they came
      from Vertex Batch Prediction, where response_column_a represents a field
      in the model output containing the response. If False, the expected format
      will be a table containing all model_prompt_parameters and the
      response_column.
    is_bp_output_b: If True, the prediction URIs will be parsed as if they came
      from Vertex Batch Prediction, where response_column_b represents a field
      in the model output containing the response. If False, the expected format
      will be a table containing all model_prompt_parameters and the
      response_column.
    prediction_uris: A list of GCS or BigQuery URIs representing a dataset of
      prompts and responses for model a.
    prediction_uris: A list of GCS or BigQuery URIs representing a dataset of
      prompts and responses for model b.
    model_a_prompt_parameters: Map of model A prompt template parameters to
      columns or templates.
    model_b_prompt_parameters: Map of model B prompt template parameters to
      columns or templates.
    autorater_prompt_parameters: Map of autorater prompt template parameters to
      columns or templates.
    human_preference_column: The column containing ground truths. The default
      value is an empty string if not be provided by users.

  Returns:
    preprocessed_evaluation_dataset: Dataset of the table containing the inputs
    expected by the Arbiter.
    preprocessed_evaluation_dataset_uri: URI of the table containing the inputs
    expected by the Arbiter.
    gcp_resources: Tracker for GCP resources created by this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='arbiter_preprocess',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'arbiter_preprocess',
              f'--evaluation_dataset={evaluation_dataset}',
              f'--prediction_uris_a={prediction_uris_a}',
              f'--prediction_uris_b={prediction_uris_b}',
              (
                  '--id_columns='
                  "{{$.inputs.parameters['id_columns'].json_escape[0]}}"
              ),
              (
                  '--autorater_prompt_parameters='
                  "{{$.inputs.parameters['autorater_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
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
              f'--response_column_a={response_column_a}',
              f'--response_column_b={response_column_b}',
              f'--human_preference_column={human_preference_column}',
              f'--task={task}',
              f'--is_batch_prediction_output_a={is_bp_output_a}',
              f'--is_batch_prediction_output_b={is_bp_output_b}',
              f'--output_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}',
              f'--preprocessed_evaluation_dataset_uri={preprocessed_evaluation_dataset_uri}',
              '--executor_input={{$.json_escape[1]}}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
