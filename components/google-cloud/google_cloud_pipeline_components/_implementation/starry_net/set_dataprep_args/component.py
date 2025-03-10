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
"""StarryNet Set Dataprep Args Component."""

from typing import List, NamedTuple

from kfp import dsl


@dsl.component
def set_dataprep_args(
    model_blocks: List[str],
    ts_identifier_columns: List[str],
    static_covariate_columns: List[str],
    csv_data_path: str,
    previous_run_dir: str,
    location: str,
) -> NamedTuple(
    'DataprepArgs',
    model_blocks=str,
    ts_identifier_columns=str,
    static_covariate_columns=str,
    create_tf_records=bool,
    docker_region=str,
):
  # fmt: off
  """Creates Dataprep args.

  Args:
    model_blocks: The list of model blocks to use in the order they will appear
      in the model. Possible values are `cleaning`, `change_point`, `trend`,
      `hour_of_week`, `day_of_week`, `day_of_year`, `week_of_year`,
      `month_of_year`, `residual`.
    ts_identifier_columns: The list of ts_identifier columns from the BigQuery
      data source.
    static_covariate_columns: The list of strings of static covariate names.
    csv_data_path: The path to the training data csv in the format
      gs://bucket_name/sub_dir/blob_name.csv.
    previous_run_dir: The dataprep dir from a previous run. Use this
      to save time if you've already created TFRecords from your BigQuery
      dataset with the same dataprep parameters as this run.
    location: The location where the pipeline is run.

  Returns:
    A NamedTuple containing the model blocks formatted as expected by the
    dataprep job, the ts_identifier_columns formatted as expected by the
    dataprep job, the static_covariate_columns formatted as expected by the
    dataprep job, a boolean indicating whether to create tf records, and the
    region of the dataprep docker image.
  """
  outputs = NamedTuple(
      'DataprepArgs',
      model_blocks=str,
      ts_identifier_columns=str,
      static_covariate_columns=str,
      create_tf_records=bool,
      docker_region=str,
  )

  def maybe_update_model_blocks(model_blocks: List[str]) -> List[str]:
    return [f'{b}-hybrid' if '_of_' in b else b for b in model_blocks]

  def create_name_tuple_from_list(input_list: List[str]) -> str:
    if len(input_list) == 1:
      return str(input_list).replace('[', '(').replace(']', ',)')
    return str(input_list).replace('[', '(').replace(']', ')')

  def set_docker_region(location: str) -> str:
    if location.startswith('africa') or location.startswith('europe'):
      return 'europe'
    elif (
        location.startswith('asia')
        or location.startswith('australia')
        or location.startswith('me')
    ):
      return 'asia'
    else:
      return 'us'

  return outputs(
      create_name_tuple_from_list(maybe_update_model_blocks(model_blocks)),  # pylint: disable=too-many-function-args
      ','.join(ts_identifier_columns),  # pylint: disable=too-many-function-args
      create_name_tuple_from_list(static_covariate_columns),  # pylint: disable=too-many-function-args
      False if csv_data_path or previous_run_dir else True,  # pylint: disable=too-many-function-args
      set_docker_region(location),  # pylint: disable=too-many-function-args
  )
