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
"""KFP Container component that performs row-level pairwise evaluation."""

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


def _get_prediction_endpoint_overrides() -> str:
  """Used for integration tests to override the prediction endpoint."""
  return os.environ.get('PREDICTION_ENDPOINT_OVERRIDES', '')


# pylint: disable=unused-argument,dangerous-default-value
@dsl.container_component
def online_evaluation_pairwise(
    inference_output_uri: str,
    id_columns: List[str],
    task: str,
    judgments: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    judgments_uri: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    error_messages: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    gcp_resources: dsl.OutputPath(str),
    metadata: dsl.OutputPath(str),
    human_preference_column: str = '',
    judgments_format: str = 'jsonl',
    bigquery_destination_prefix: str = '',
    experimental_args: Dict[str, Any] = {},
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    encryption_spec_key_name: str = '',
    autorater_prompt_parameters: Dict[str, Dict[str, str]] = {},
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Evaluate two models using an autorater.

  Args:
    inference_output_uri: Directory of model A's inference output.
    id_columns: The columns which distinguish unique evaluation examples.
    human_preference_column: Human preference column included in our inference
      output.
    task: Evaluation task in the form {task}@{version}. task can be one of
      "summarization", "question_answering". Version is an integer with 3 digits
      or "latest". Ex: summarization@001 or question_answering@latest.
    judgments_format: The format to write judgments to. Can be either 'json' or
      'bigquery'.
    bigquery_destination_prefix: BigQuery table to write judgments to if the
      specified format is 'bigquery'.
    experimental_args: Experimentally released arguments. Subject to change.
    project: Project used to make autorater predictions.
    location: Location used to make autorater predictions.
    encryption_spec_key_name: Customer-managed encryption key options. If this
      is set, then all resources created by the component will be encrypted with
      the provided encryption key.
    autorater_prompt_parameters: Map of autorater prompt template parameters to
      columns or templates.

  Returns:
    judgments: Individual judgments used to calculate the win rates.
    judgments_uri: URI of the Judgments Artifact.
    error_messages: Error messages of failed samples of each evaluation example.
    gcp_resources: Tracker for GCP resources created by this component.
    metadata: Computed runtime metrics metadata from this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='online_evaluation_pairwise',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'arbiter',
              f'--inference_output_uri={inference_output_uri}',
              f'--human_preference_column={human_preference_column}',
              f'--task={task}',
              f'--project={project}',
              f'--location={location}',
              f'--prediction_endpoint_overrides={_get_prediction_endpoint_overrides()}',
              f'--output_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}',
              f'--judgments_uri={judgments_uri}',
              f'--judgments_format={judgments_format}',
              f'--bigquery_destination_prefix={bigquery_destination_prefix}',
              (
                  '--id_columns='
                  "{{$.inputs.parameters['id_columns'].json_escape[0]}}"
              ),
              (
                  '--experimental_args='
                  "{{$.inputs.parameters['experimental_args'].json_escape[0]}}"
              ),
              '--executor_input={{$.json_escape[1]}}',
              f'--kms_key_name={encryption_spec_key_name}',
              f'--metadata_path={metadata}',
              (
                  '--autorater_prompt_parameters='
                  "{{$.inputs.parameters['autorater_prompt_parameters']"
                  '.json_escape[0]}}'
              ),
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
