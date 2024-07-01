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
import json
from os import path
from pathlib import Path
from prophet.serialize import model_from_json
import sys
import tempfile
import unittest

sys.path.append(str(Path(__file__).parent.parent.absolute()))
from prediction import predict


class TestPrediction(unittest.TestCase):

  @classmethod
  def setUpClass(cls):
    with open(
        path.join(path.dirname(__file__), 'data', 'example_model.json'),
        'r') as f:
      cls.model = model_from_json(json.load(f))

    with open(
        path.join(
            path.dirname(__file__), 'data',
            'example_model_with_features.json')) as f:
      cls.model_with_features = model_from_json(json.load(f))

  def test_basic_predict_periods(self):
    forecast = predict(self.model, periods=365)
    self.assertIn('ds', forecast)
    self.assertIn('yhat', forecast)

  def test_basic_predict_future_frame(self):
    with tempfile.NamedTemporaryFile() as f:
      self.model.make_future_dataframe(periods=100, freq='M').to_csv(f.name)
      forecast = predict(self.model, future_data_source=f.name)
      self.assertIn('ds', forecast)
      self.assertIn('yhat', forecast)

  def test_with_features_periods(self):
    with self.assertRaises(Exception):
      predict(self.model_with_features, periods=10)

  def test_with_features_future_frame(self):
    with tempfile.NamedTemporaryFile() as f:
      future = self.model_with_features.make_future_dataframe(
          periods=100, freq='M')
      future.to_csv(f.name)

      # The future dataframe is missing the additional regressor 'r'
      with self.assertRaises(Exception):
        forecast = predict(self.model_with_features, future_data_source=f.name)

      future['r'] = [0] * len(future)
      future.to_csv(f.name)
      forecast = predict(self.model_with_features, future_data_source=f.name)

      self.assertIn('ds', forecast)
      self.assertIn('yhat', forecast)


if __name__ == '__main__':
  unittest.main()
