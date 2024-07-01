#!/usr/bin/env python3
# Copyright 2021 Google LLC. All Rights Reserved.
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
"""Make predictions using a trained Facebook Prophet model in a pipeline.

For a detailed description of the input and output of this component see
prediction_component.yaml
"""
import json
import pathlib
import os

from prediction_args import parser
from prophet.serialize import model_from_json
from util import DataSource
import pandas as pd
from prophet import Prophet
from typing import Optional


def predict(m: Prophet,
            periods: Optional[int] = None,
            future_data_source: Optional[str] = None) -> pd.DataFrame:
  # Assert that exactly one of the two mutually exclusive arguments was provided
  assert bool(periods) != bool(future_data_source)
  future = DataSource(future_data_source).load(
  ) if future_data_source is not None else m.make_future_dataframe(
      periods=periods)

  if len(m.extra_regressors):
    if future_data_source is None:
      raise Exception(
          'The proivded model was trained with additional regressors. The future data source must be provided instead of periods.'
      )
    else:
      # Verify that user-provided future data frame contains the required
      # features
      missing_regressors = [r for r in m.extra_regressors if r not in future]
      if len(missing_regressors):
        raise Exception(
            'The following column(s) were missing from the provided future data source: '
            + ', '.join(missing_regressors))

  return m.predict(future)


if __name__ == '__main__':
  args = parser.parse_args()

  with open(args.model, 'r') as f:
    m = model_from_json(json.load(f))

  os.makedirs(
      pathlib.Path(args.prediction_output_file_path).parent.absolute(),
      exist_ok=True)

  predict(m, args.periods,
          args.future_data_source).to_csv(args.prediction_output_file_path)
