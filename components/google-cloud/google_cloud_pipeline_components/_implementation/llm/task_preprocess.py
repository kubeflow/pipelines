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
"""Component for preprocessing the evaluation dataset into prediction inputs."""

import os
from typing import Any, Dict, List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  """Determines the image URI to create a container from."""
  return (
      os.environ.get('AUTOSXS_IMAGE_OVERRIDE')
      or utils.get_default_image_uri('autosxs'))


# pylint: disable=dangerous-default-value,g-bare-generic,unused-argument
@dsl.container_component
def task_preprocess(
    evaluation_dataset: str,
    id_columns: List[str],
    task: str,
    model_prompt_parameters: Dict[str, Dict[str, str]],
    prediction_inputs: dsl.OutputPath(List[str]),  # pytype: disable=invalid-annotation
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata: dsl.OutputPath(Dict[str, Any]),  # pytype: disable=invalid-annotation
    response_column: str,
    human_preference_column: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Preprocesses evaluation dataset into prediction inputs.

  Args:
    evaluation_dataset: GCS or BigQuery URIs representing a dataset of prompts
      and responses.
    id_columns: The columns which distinguish unique evaluation examples.
    task: Evaluation task in the form {task}@{version}. task can be one of
      "summarization", "question_answer". Version is an integer with 3 digits or
      "latest". Ex: summarization@001 or question_answer@latest.
    model_prompt_parameters: Map of model prompt template parameters to columns
      or templates.
    response_column: Either an existing column containing predefined responses,
      or the name of the model output column containing responses.
    human_preference_column: The column containing ground truths. Only required
      when users want to check the autorater alignment against human preference.

  Returns:
    prediction_inputs_path: Path to write the path where preprocessed
      predictions are stored.
    gcp_resources: Tracker for GCP resources created by this component.
    metadata_path: Path to write the object that stores computed metrics
      metadata for the task preprocess component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='task_preprocess',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'task_preprocess',
              f'--evaluation_dataset={evaluation_dataset}',
              f'--staging_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}',
              f'--task={task}',
              f'--prediction_inputs_path={prediction_inputs}',
              (
                  '--id_columns='
                  "{{$.inputs.parameters['id_columns'].json_escape[0]}}"
              ),
              (
                  '--model_prompt_parameters='
                  "{{$.inputs.parameters['model_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
              f'--metadata_path={metadata}',
              f'--response_column={response_column}',
              f'--human_preference_column={human_preference_column}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
