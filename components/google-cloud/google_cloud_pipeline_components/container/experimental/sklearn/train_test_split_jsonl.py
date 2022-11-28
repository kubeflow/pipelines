# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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
"""Component for splitting a JSONL dataset into training and validation."""
import argparse
import pathlib
import random
import sys
from typing import Any, Dict

import pandas as pd
from sklearn.model_selection import train_test_split


def split_dataset_into_train_and_validation(
    training_data_path: str,
    validation_data_path: str,
    input_data_path: str,
    validation_split: float = 0.2,
    random_seed: int = 0
) -> None:
  """Split JSON(L) Data into training and validation data.

  Args:
    training_data_path: Output path for the training data
    validation_data_path: Output path for the validation data
    input_data_path: Data in JSON lines format.
    validation_split: Fraction of data that will make up validation dataset
    random_seed: Random seed
  """
  random.seed(random_seed)

  data = pd.read_json(input_data_path, lines=True, orient='records')

  train, test = train_test_split(data, test_size=validation_split,
                                 random_state=random_seed,
                                 shuffle=True)
  train.to_json(path_or_buf=training_data_path,
                lines=True, orient='records')
  test.to_json(path_or_buf=validation_data_path,
               lines=True, orient='records')


def _parse_args(args) -> Dict[str, Any]:
  """Parse command line arguments.

  Args:
    args: A list of arguments.
  Returns:
    A tuple containing an argparse.Namespace class instance holding parsed args,
    and a list containing all unknown args.
  """

  parser = argparse.ArgumentParser(
      prog='Text classification data processing', description='')
  parser.add_argument('--training-data-path',
                      type=str,
                      required=True,
                      default=argparse.SUPPRESS)
  parser.add_argument('--validation-data-path',
                      type=str,
                      required=True,
                      default=argparse.SUPPRESS)
  parser.add_argument('--input-data-path',
                      type=str,
                      required=True,
                      default=argparse.SUPPRESS)
  parser.add_argument('--validation-split',
                      type=float,
                      required=False,
                      default=0.2)
  parser.add_argument('--random-seed',
                      type=int,
                      required=False,
                      default=0)

  parsed_args, _ = parser.parse_known_args(args)

  # Creating the directory where the output file is created. The parent
  # directory does not exist when building container components.
  pathlib.Path(parsed_args.training_data_path).parent.mkdir(
      parents=True, exist_ok=True)
  pathlib.Path(parsed_args.validation_data_path).parent.mkdir(
      parents=True, exist_ok=True)

  return vars(parsed_args)


def main(argv):
  """Main entry for text classification preprocessing component.

  expected input args are as follows:
  training_data_path: Required. Output path for the training data
  validation_data_path: Required. Output path for the validation data
  input_data_path: Required. Data in JSON lines format.
  validation_split: Fraction of data that will make up validation dataset
  random_seed: Random seed.

  Args:
    argv: A list of system arguments.
  """
  parsed_args = _parse_args(argv)
  split_dataset_into_train_and_validation(**parsed_args)


if __name__ == '__main__':
  main(sys.argv[1:])
