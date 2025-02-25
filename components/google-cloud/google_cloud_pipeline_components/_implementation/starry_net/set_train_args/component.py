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
"""Starry Net component to set training args."""

from typing import List, NamedTuple

from kfp import dsl


@dsl.component
def set_train_args(
    quantiles: List[float],
    model_blocks: List[str],
    static_covariates: List[str],
) -> NamedTuple(
    'TrainArgs',
    quantiles=str,
    use_static_covariates=bool,
    static_covariate_names=str,
    model_blocks=str,
    freeze_point_forecasts=bool,
):
  # fmt: off
  """Creates Trainer model args.

  Args:
    quantiles: The list of floats representing quantiles. Leave blank if
      only training to produce point forecasts.
    model_blocks: The list of model blocks to use in the order they will appear
      in the model. Possible values are `cleaning`, `change_point`, `trend`,
      `hour_of_week`, `day_of_week`, `day_of_year`, `week_of_year`,
      `month_of_year`, `residual`.
    static_covariates: The list of strings of static covariate names.

  Returns:
    A NamedTuple containing the quantiles formatted as expected by the train
    job, a bool indicating whether the job should train with static covariates,
    the model blocks formatted as expected by the train job, and a bool
    indicating whether or not to do two-pass training, fist training for point
    forecsats and then quantiles.
  """

  outputs = NamedTuple(
      'TrainArgs',
      quantiles=str,
      use_static_covariates=bool,
      static_covariate_names=str,
      model_blocks=str,
      freeze_point_forecasts=bool,
  )

  def set_quantiles(input_list: List[float]) -> str:
    if not input_list or input_list[0] != 0.5:
      input_list = [0.5] + input_list
    if len(input_list) == 1:
      return str(input_list).replace('[', '(').replace(']', ',)')
    return str(input_list).replace('[', '(').replace(']', ')')

  def maybe_update_model_blocks(
      quantiles: List[float], model_blocks: List[str]) -> List[str]:
    updated_q = [q for q in quantiles if q != 0.5]
    model_blocks = [b for b in model_blocks if b != 'quantile']
    if updated_q:
      model_blocks.append('quantile')
    return [f'{b}-hybrid' if '_of_' in b else b for b in model_blocks]

  def create_name_tuple_from_list(input_list: List[str]) -> str:
    if len(input_list) == 1:
      return str(input_list).replace('[', '(').replace(']', ',)')
    return str(input_list).replace('[', '(').replace(']', ')')

  return outputs(
      set_quantiles(quantiles),  # pylint: disable=too-many-function-args
      True if static_covariates else False,  # pylint: disable=too-many-function-args
      create_name_tuple_from_list(static_covariates),  # pylint: disable=too-many-function-args
      create_name_tuple_from_list(  # pylint: disable=too-many-function-args
          maybe_update_model_blocks(quantiles, model_blocks)),
      True if quantiles and quantiles[-1] != 0.5 else False,  # pylint: disable=too-many-function-args
  )
