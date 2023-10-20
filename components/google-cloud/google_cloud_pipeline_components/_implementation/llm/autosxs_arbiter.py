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

"""KFP Container component that performs AutoSxS."""

import os

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  return (
      os.environ.get('AUTOSXS_IMAGE_OVERRIDE')
      or utils.get_default_image_uri('autosxs'))


def _get_prediction_endpoint_overrides() -> str:
  """Used for integration tests to override the prediction endpoint."""
  return os.environ.get('PREDICTION_ENDPOINT_OVERRIDES', '')


# TODO(b/290849010):  Align with how the Vertex API handles GCS inputs.
# pylint: disable=dangerous-default-value,g-bare-generic
@dsl.container_component
def autosxs_arbiter(
    model_a_prediction_dir: str,
    model_b_prediction_dir: str,
    instruction: str,
    task: str,
    win_rate_metrics: dsl.Output[dsl.Metrics],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    judgments: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    judgments_uri: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    gcp_resources: dsl.OutputPath(str),
    judgments_format: str = 'jsonl',
    bigquery_destination_prefix: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Compares two sets of predictions and returns the winrate for each.

  Args:
    model_a_prediction_dir: Directory of model A's inference output.
    model_b_prediction_dir: Directory of model B's inference output.
    instruction: Instruction used in model A/B inference.
    task: Task used for model A/B inference. Can be one of: 'summarization'.
    judgments_format: The format to write judgments to. Can be either 'json' or
      'bigquery'.
    bigquery_destination_prefix: BigQuery table to write judgments to if the
      specified format is 'bigquery'.

  Returns:
    win_rate_metrics: Win rates.
    judgments: Individual judgments used to calculate the win rates.
    judgments_uri: URI of the Judgments Artifact.
    gcp_resources: Tracker for GCP resources created by this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      # Hardcode location to us-central1 for text-bison availability.
      location='us-central1',
      custom_job_payload=utils.build_payload(
          display_name='autosxs_arbiter',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'arbiter',
              f'--model_a_prediction_dir={model_a_prediction_dir}',
              f'--model_b_prediction_dir={model_b_prediction_dir}',
              f'--instruction={instruction}',
              f'--task={task}',
              f'--prediction_endpoint_overrides={_get_prediction_endpoint_overrides()}',
              f'--output_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
              f'--judgments_uri={judgments_uri}',
              f'--judgments_format={judgments_format}',
              f'--bigquery_destination_prefix={bigquery_destination_prefix}',
              '--executor_input={{$.json_escape[1]}}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
