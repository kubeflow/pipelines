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
"""Starry Net component to set TFRecord args if training with TF Records."""

from typing import List, NamedTuple

from kfp import dsl


@dsl.component
def maybe_set_tfrecord_args(
    dataprep_previous_run_dir: str,
    static_covariates: List[str],
) -> NamedTuple(
    'TfrecordArgs',
    static_covariates_vocab_path=str,
    train_tf_record_patterns=str,
    val_tf_record_patterns=str,
    test_tf_record_patterns=str,
):
  # fmt: off
  """Creates Trainer TFRecord args if training with TF Records.

  Args:
    dataprep_previous_run_dir: The dataprep dir from a previous run. Use this
      to save time if you've already created TFRecords from your BigQuery
      dataset with the same dataprep parameters as this run.
    static_covariates: The static covariates to train the model with.

  Returns:
    A NamedTuple containing the path to the static covariates covabulary, and
    the tf record patterns for the train, validation, and test sets.
  """

  outputs = NamedTuple(
      'TfrecordArgs',
      static_covariates_vocab_path=str,
      train_tf_record_patterns=str,
      val_tf_record_patterns=str,
      test_tf_record_patterns=str,
  )

  if static_covariates and dataprep_previous_run_dir:
    static_covariates_vocab_path = (
        f'{dataprep_previous_run_dir}/static_covariate_vocab.json'
    )
  else:
    static_covariates_vocab_path = ''
  if dataprep_previous_run_dir:
    train_tf_record_patterns = (
        f"('{dataprep_previous_run_dir}/tf_records/train*',)"
    )
    val_tf_record_patterns = f"('{dataprep_previous_run_dir}/tf_records/val*',)"
    test_tf_record_patterns = (
        f"('{dataprep_previous_run_dir}/tf_records/test_path_for_plot*',)"
    )
  else:
    train_tf_record_patterns = '()'
    val_tf_record_patterns = '()'
    test_tf_record_patterns = '()'
  return outputs(
      static_covariates_vocab_path,  # pylint: disable=too-many-function-args
      train_tf_record_patterns,  # pylint: disable=too-many-function-args
      val_tf_record_patterns,  # pylint: disable=too-many-function-args
      test_tf_record_patterns,  # pylint: disable=too-many-function-args
  )
