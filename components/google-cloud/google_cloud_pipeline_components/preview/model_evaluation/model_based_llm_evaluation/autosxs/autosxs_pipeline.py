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
"""Optimization AI Inference and AutoSxS pipeline function."""

from typing import Any, Dict, List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import batch_prediction_pairwise
from google_cloud_pipeline_components._implementation.llm import model_evaluation_text_generation_pairwise
from google_cloud_pipeline_components._implementation.llm import online_evaluation_pairwise
from kfp import dsl


# pylint: disable=dangerous-default-value,g-bare-generic,unused-argument
@dsl.pipeline(
    name='autosxs-template',
    description='Determines the SxS winrate between two models.',
)
def autosxs_pipeline(
    evaluation_dataset: str,
    task: str,
    id_columns: List[str],
    model_a: str = '',
    model_b: str = '',
    autorater_prompt_parameters: Dict[str, Dict[str, str]] = {},
    model_a_prompt_parameters: Dict[str, Dict[str, str]] = {},
    model_b_prompt_parameters: Dict[str, Dict[str, str]] = {},
    response_column_a: str = '',
    response_column_b: str = '',
    model_a_parameters: Dict[str, str] = {},
    model_b_parameters: Dict[str, str] = {},
    human_preference_column: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    judgments_format: str = 'jsonl',
    bigquery_destination_prefix: str = '',
    experimental_args: Dict[str, Any] = {},
):
  # fmt: off
  """Evaluates two models side-by-side using an arbiter model.

  Args:
    evaluation_dataset: A BigQuery table or comma-separated list of GCS paths to a JSONL dataset containing evaluation examples.
    task: Evaluation task in the form `{task}@{version}`. task can be one of `[summarization, question_answering]`. Version is an integer with 3 digits or "latest". Ex: `summarization@001` or `question_answering@latest`.
    id_columns: The columns which distinguish unique evaluation examples.
    model_a: A fully-qualified model resource name (`projects/{project}/locations/{location}/models/{model}@{version}`) or publisher model resource name (`publishers/{publisher}/models/{model}`).  This parameter is optional if Model A responses are specified.
    model_b: A fully-qualified model resource name (`projects/{project}/locations/{location}/models/{model}@{version}`) or publisher model resource name (`publishers/{publisher}/models/{model}`).  This parameter is optional if Model B responses are specified.
    autorater_prompt_parameters: Map of autorater prompt parameters to columns or templates. The expected parameters are: `inference_instruction` (details on how to perform a task) and `inference_context` (content to reference to perform the task). As an example, `{'inference_context': {'column': 'my_prompt'}}` uses the evaluation dataset's `my_prompt` column for the AutoRater's context.
    model_a_prompt_parameters: Map of Model A prompt template parameters to columns or templates. This parameter is optional if Model A predictions are predefined. Example - `{'prompt': {'column': 'my_prompt'}}` uses the evaluation dataset's `my_prompt` column for the prompt parameter named `prompt`.
    model_b_prompt_parameters: Map of Model B prompt template parameters to columns or templates. This parameter is optional if Model B predictions are predefined. Example - `{'prompt': {'column': 'my_prompt'}}` uses the evaluation dataset's `my_prompt` column for the prompt parameter named `prompt`.
    response_column_a: Either the name of a column in the evaluation dataset containing predefined predictions, or the name of the column in the Model A output containing predictions. If no value is provided, the correct model output column name will attempt to be inferred.
    response_column_b: Either the name of a column in the evaluation dataset containing predefined predictions, or the name of the column in the Model B output containing predictions. If no value is provided, the correct model output column name will attempt to be inferred.
    model_a_parameters: The parameters that govern the predictions from model A, such as temperature or maximum output tokens.
    model_b_parameters: The parameters that govern the predictions from model B, such as temperature or maximum output tokens.
    human_preference_column: The column containing ground truth winners for each example. Providing this parameter adds additional metrics for checking the AutoRater alignment with human preferences.
    project: Project used to run custom jobs. This should be the same project used to run the pipeline.
    location: Location used to run custom jobs. This should be the same location used to run the pipeline.
    judgments_format: The format to write judgments to. Can be either `[json, bigquery]`.
    bigquery_destination_prefix: BigQuery table to write judgments to if the specified format is 'bigquery'.
    experimental_args: Experimentally released arguments. Subject to change.
  """
  # fmt: on
  responses = batch_prediction_pairwise.batch_prediction_pairwise(
      display_name='autosxs-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      evaluation_dataset=evaluation_dataset,
      id_columns=id_columns,
      task=task,
      autorater_prompt_parameters=autorater_prompt_parameters,
      response_column_a=response_column_a,
      response_column_b=response_column_b,
      model_a=model_a,
      model_b=model_b,
      model_a_prompt_parameters=model_a_prompt_parameters,
      model_b_prompt_parameters=model_b_prompt_parameters,
      model_a_parameters=model_a_parameters,
      model_b_parameters=model_b_parameters,
      human_preference_column=human_preference_column,
      experimental_args=experimental_args,
  ).set_display_name('AutoSxS Batch Prediction')

  winners = online_evaluation_pairwise.online_evaluation_pairwise(
      inference_output_uri=responses.outputs[
          'preprocessed_evaluation_dataset_uri'
      ],
      id_columns=id_columns,
      human_preference_column=human_preference_column,
      task=task,
      judgments_format=judgments_format,
      bigquery_destination_prefix=bigquery_destination_prefix,
      experimental_args=experimental_args,
  ).set_display_name('AutoSxS Autorater')

  model_evaluation_text_generation_pairwise.model_evaluation_text_generation_pairwise(
      judgments_dir=winners.outputs['judgments_uri'],
      human_preference_column=human_preference_column,
  ).set_display_name(
      'AutoSxS Metrics'
  )
