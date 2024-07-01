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
import os
from pathlib import Path
import sys
import tempfile
import unittest
import numpy as np
import pandas as pd

sys.path.append(str(Path(__file__).parent.parent.absolute()))
from fit_model import fit_model


class TestFitModel(unittest.TestCase):

  @classmethod
  def setUpClass(cls):
    cls.data = pd.DataFrame({
        'ds': pd.date_range(0, periods=500, freq='D'),
        'y': np.arange(0, 500),
        'feature': np.arange(0, 500)
    })

    _, cls.data_file_path = tempfile.mkstemp()
    cls.data.to_csv(cls.data_file_path)

  @classmethod
  def tearDownClass(cls):
    os.remove(cls.data_file_path)

  def test_fit_basic_model(self):
    model, _ = fit_model(self.data_file_path, {})
    self.assertIsNotNone(model.history, 'Model has not been fit')

  def test_fit_with_opt_args(self):
    prophet_constructor_args = {
        'n_changepoints': 1,
        'changepoint_range': 0.5,
        'yearly_seasonality': 'auto',
        'weekly_seasonality': False,
        'daily_seasonality': 1,
        'seasonality_mode': 'additive',
        'seasonality_prior_scale': 2,
        'holidays_prior_scale': 3,
        'changepoint_prior_scale': 4,
        'mcmc_samples': 10,
        'interval_width': 0.1,
        'uncertainty_samples': False,
        'stan_backend': None
    }
    model, _ = fit_model(self.data_file_path, prophet_constructor_args)
    self.assertIsNotNone(model.history, 'Model has not been fit')


if __name__ == '__main__':
  unittest.main()
