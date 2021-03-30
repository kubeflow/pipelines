# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for kfp.dsl.io_types."""

import unittest
import json
import os

from kfp.dsl import io_types


class IOTypesTest(unittest.TestCase):

  def test_complex_metrics(self):
    metrics = io_types.ClassificationMetrics()
    metrics.log_roc_reading(0.1, 98.2, 96.2)
    metrics.log_roc_reading(24.3, 24.5, 98.4)
    metrics.set_confusion_matrix_categories(['dog', 'cat', 'horses'])
    metrics.log_confusion_matrix_row('dog', [2, 6, 0])
    metrics.log_confusion_matrix_cell('cat', 'dog', 3)
    metrics.metadata["test"] = 1.0
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_io_types_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, metrics.metadata)
  
  def test_complex_metrics_bulk_loading(self):
    metrics = io_types.ClassificationMetrics()
    metrics.load_roc_readings([
      [53.6, 52.6, 85.1],
      [53.6, 52.6, 85.1],
      [53.6, 52.6, 85.1]])
    metrics.load_confusion_matrix(['dog', 'cat', 'horses'], [[2, 6, 0], [3, 5,6], [5,7,8]])
    with open(os.path.join(os.path.dirname(__file__),
        'test_data', 'expected_io_types_bulk_load_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, metrics.metadata)

if __name__ == '__main__':
  unittest.main()