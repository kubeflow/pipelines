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
from google_cloud_pipeline_components._implementation.llm import arbiter_preprocess
from google_cloud_pipeline_components._implementation.llm import autosxs_arbiter
from google_cloud_pipeline_components._implementation.llm import autosxs_metrics_computer
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import task_preprocess
from google_cloud_pipeline_components.types import artifact_types
from google_cloud_pipeline_components.v1 import batch_predict_job
from kfp import dsl


# pylint: disable=no-value-for-parameter
@dsl.pipeline(
    name='predictions-pipeline',
    description='Runs the prediction pipeline for one of the two SxS models.',
)
def _get_predictions(
    name: str,
    project: str,
    location: str,
    model: str,
    model_parameters: Dict[str, str],
    prediction_inputs: List[str],
    is_model_inference: bool,
) -> str:
  """Makes predictions for a given model."""
  with dsl.If(is_model_inference == True, name='Inference Required'):  # pylint: disable=singleton-comparison
    get_vertex_model_task = dsl.importer(
        artifact_uri=(
            f'https://{location}-aiplatform.googleapis.com/v1/{model}'
        ),
        artifact_class=artifact_types.VertexModel,
        metadata={'resourceName': model},
    ).set_display_name('Import Vertex Model Artifact')

    batch_predict_task = batch_predict_job.ModelBatchPredictOp(
        project=project,
        location=location,
        model=get_vertex_model_task.outputs['artifact'],
        job_display_name=(
            f'autosxs-{name}-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}'
        ),
        gcs_source_uris=prediction_inputs,
        instances_format='jsonl',
        predictions_format='jsonl',
        gcs_destination_output_uri_prefix=(
            f'{dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
            f'/{name}_predictions'
        ),
        model_parameters=model_parameters,
    )
    prediction_uris_from_inference = function_based.get_uri(
        artifact=batch_predict_task.outputs['gcs_output_directory'],
        is_dir=True,
    )

  with dsl.Else(name='Responses Provided'):  # pylint: disable=singleton-comparison
    prediction_uris_inference_provided = function_based.get_empty_string()

  prediction_uris = dsl.OneOf(
      prediction_uris_from_inference.output,
      prediction_uris_inference_provided.output,
  )

  # We can't directly output dsl.OneOf, so we need to use identity.
  return function_based.identity(x=prediction_uris).output


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
    project: Project used to run custom jobs. Default is the same project used to run the pipeline.
    location: Location used to run custom jobs. Default is the same location used to run the pipeline.
    judgments_format: The format to write judgments to. Can be either `[json, bigquery]`.
    bigquery_destination_prefix: BigQuery table to write judgments to if the specified format is 'bigquery'.
    experimental_args: Experimentally released arguments. Subject to change.
  """
  # fmt: on
  prediction_inputs_a = task_preprocess.task_preprocess(
      evaluation_dataset=evaluation_dataset,
      task=task,
      model_prompt_parameters=model_a_prompt_parameters,
      response_column=response_column_a,
      human_preference_column=human_preference_column,
      id_columns=id_columns,
  ).set_display_name('Preprocess Model A Inputs')

  prediction_inputs_b = task_preprocess.task_preprocess(
      evaluation_dataset=evaluation_dataset,
      task=task,
      model_prompt_parameters=model_b_prompt_parameters,
      response_column=response_column_b,
      human_preference_column=human_preference_column,
      id_columns=id_columns,
  ).set_display_name('Preprocess Model B Inputs')

  is_model_a_inference = function_based.get_usage_metric(
      metadata=prediction_inputs_a.outputs['metadata'],
      key='is_model_inference',
  ).set_display_name('Read is_model_a_inference')

  is_model_b_inference = function_based.get_usage_metric(
      metadata=prediction_inputs_b.outputs['metadata'],
      key='is_model_inference',
  ).set_display_name('Read is_model_b_inference')

  inferrer_a = _get_predictions(
      name='A',
      project=project,
      location=location,
      model=model_a,
      model_parameters=model_a_parameters,
      prediction_inputs=prediction_inputs_a.outputs['prediction_inputs'],
      is_model_inference=is_model_a_inference.output,
  ).set_display_name('Model A Responses')

  inferrer_b = _get_predictions(
      name='B',
      project=project,
      location=location,
      model=model_b,
      model_parameters=model_b_parameters,
      prediction_inputs=prediction_inputs_b.outputs['prediction_inputs'],
      is_model_inference=is_model_b_inference.output,
  ).set_display_name('Model B Responses')

  arbiter_input_preprocess = arbiter_preprocess.arbiter_preprocess(
      autorater_prompt_parameters=autorater_prompt_parameters,
      evaluation_dataset=evaluation_dataset,
      id_columns=id_columns,
      prediction_uris_b=inferrer_b.output,
      prediction_uris_a=inferrer_a.output,
      model_a_prompt_parameters=model_a_prompt_parameters,
      model_b_prompt_parameters=model_b_prompt_parameters,
      task=task,
      response_column_a=response_column_a,
      response_column_b=response_column_b,
      human_preference_column=human_preference_column,
      is_bp_output_a=is_model_a_inference.output,
      is_bp_output_b=is_model_b_inference.output,
  ).set_display_name('Preprocess Predictions')

  autosxs_arbiter_task = autosxs_arbiter.autosxs_arbiter(
      inference_output_uri=arbiter_input_preprocess.outputs[
          'preprocessed_evaluation_dataset_uri'
      ],
      id_columns=id_columns,
      human_preference_column=human_preference_column,
      task=task,
      judgments_format=judgments_format,
      bigquery_destination_prefix=bigquery_destination_prefix,
      experimental_args=experimental_args,
  ).set_display_name('AutoSxS Arbiter')

  has_human_preference = function_based.get_usage_metric(
      metadata=prediction_inputs_a.outputs['metadata'],
      key='has_human_preference_column',
  ).set_display_name('Read has_human_preference_column')

  autosxs_metrics_computer.autosxs_metrics_computer(
      judgments_dir=autosxs_arbiter_task.outputs['judgments_uri'],
      has_human_preference=has_human_preference.output,
  ).set_display_name('AutoSxS Metrics')
