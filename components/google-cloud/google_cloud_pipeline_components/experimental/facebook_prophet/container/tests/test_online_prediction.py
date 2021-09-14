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
"""HTTP server for making predictions using a trained Facebook Prophet model."""
import os
import pandas as pd
from pathlib import Path
import sys
import unittest
from unittest.mock import call, patch

# Environment variables used by the online prediction server
os.environ |= {
    'AIP_STORAGE_URI': 'gs://foo/bar/baz/',
    'AIP_HTTP_PORT': '8080',
    'AIP_HEALTH_ROUTE': '/health',
    'AIP_PREDICT_ROUTE': '/predict',
    'ONLINE_PREDICTION_UNITTEST': '1',
}

# Ensure that the online prediction module is on the python import path
sys.path.append(str(Path(__file__).parent.parent.absolute()))


class TestOnlinePrediction(unittest.TestCase):

  @patch('online_prediction.predict')
  @patch('online_prediction.get_model')
  def test_online_prediction(self, mock_get_model, mock_predict):
    # Mock loading model files from GCS
    # Prophet models are mocked as strings
    mock_get_model.side_effect = ['A', 'B', 'A', None]
    # Mock the result of prediction
    mock_predict.return_value = pd.DataFrame()

    from online_prediction import app
    app.testing = True
    with app.test_client() as c:
      self.assertEqual(c.get('/health').status_code, 200)
      self.assertEqual(
          c.post(
              '/predict',
              json={
                  'instances': [
                      {
                          'time_series': 'A',
                          'periods': 10
                      },
                      {
                          'time_series': 'B',
                          'future_data_source': 'gs://foo'
                      },
                  ]
              }).status_code, 200)
      mock_predict.assert_has_calls(
          [call('A', periods=10),
           call('B', future_data_source='gs://foo')])

      # Assert that error codes are sent when the input is invalid
      # Missing fields
      self.assertEqual(c.post('/predict').status_code, 400)
      self.assertEqual(c.post('/predict', json={}).status_code, 400)
      self.assertEqual(
          c.post('/predict', json={
              'instances': {}
          }).status_code, 400)
      self.assertEqual(
          c.post('/predict', json={
              'instances': []
          }).status_code, 400)
      self.assertEqual(
          c.post('/predict', json={
              'instances': [{}]
          }).status_code, 400)
      self.assertEqual(
          c.post('/predict', json={
              'instances': [{
                  'time_series': 'a'
              }]
          }).status_code, 400)
      # Try to predict for a model that cannot be found
      self.assertEqual(
          c.post(
              '/predict',
              json={
                  'instances': [{
                      'time_series': 'c',
                      'periods': 10
                  }]
              }).status_code, 400)


if __name__ == '__main__':
  unittest.main()
