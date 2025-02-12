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
"""Starry Net Set Eval Args Component."""

from typing import List, NamedTuple

from kfp import dsl


@dsl.component
def set_eval_args(
    big_query_source: dsl.Input[dsl.Artifact], quantiles: List[float]
) -> NamedTuple(
    'EvalArgs',
    big_query_source=str,
    forecasting_type=str,
    quantiles=List[float],
    prediction_score_column=str,
):
  # fmt: off
  """Creates Evaluation args.

  Args:
    big_query_source: The BQ Table containing the test set.
    quantiles: The quantiles the model was trained to output.

  Returns:
    A NamedTuple containing big_query_source as a string, forecasting_type
    used for evaluation step, quantiles in the format expected by the evaluation
    job, and the prediction_score_column used to evaluate.
  """
  outputs = NamedTuple(
      'EvalArgs',
      big_query_source=str,
      forecasting_type=str,
      quantiles=List[float],
      prediction_score_column=str)

  def set_forecasting_type_for_eval(quantiles: List[float]) -> str:
    if quantiles and quantiles[-1] != 0.5:
      return 'quantile'
    return 'point'

  def set_quantiles_for_eval(quantiles: List[float]) -> List[float]:
    updated_q = [q for q in quantiles if q != 0.5]
    if updated_q:
      updated_q = [0.5] + updated_q
    return updated_q

  def set_prediction_score_column(
      quantiles: List[float]) -> str:
    updated_q = [q for q in quantiles if q != 0.5]
    if updated_q:
      return 'predicted_x.quantile_predictions'
    return 'predicted_x.value'

  project_id = big_query_source.metadata['projectId']
  dataset_id = big_query_source.metadata['datasetId']
  table_id = big_query_source.metadata['tableId']
  return outputs(
      f'bq://{project_id}.{dataset_id}.{table_id}',  # pylint: disable=too-many-function-args
      set_forecasting_type_for_eval(quantiles),  # pylint: disable=too-many-function-args
      set_quantiles_for_eval(quantiles),  # pylint: disable=too-many-function-args
      set_prediction_score_column(quantiles),  # pylint: disable=too-many-function-args
  )
