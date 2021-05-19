# Copyright 2021 The Kubeflow Authors
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
from typing import List, Optional, Union

from kfp.dsl import io_types
from kfp.dsl.io_types import Input, InputAnnotation, Output, Model, OutputAnnotation


class IOTypesTest(unittest.TestCase):

  def test_complex_metrics(self):
    metrics = io_types.ClassificationMetrics()
    metrics.log_roc_data_point(threshold=0.1, tpr=98.2, fpr=96.2)
    metrics.log_roc_data_point(threshold=24.3, tpr=24.5, fpr=98.4)
    metrics.set_confusion_matrix_categories(['dog', 'cat', 'horses'])
    metrics.log_confusion_matrix_row('dog', [2, 6, 0])
    metrics.log_confusion_matrix_cell('cat', 'dog', 3)
    metrics.log_confusion_matrix_cell('horses', 'horses', 3)
    metrics.metadata['test'] = 1.0
    with open(
        os.path.join(
            os.path.dirname(__file__), 'test_data',
            'expected_io_types_classification_metrics.json')) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, metrics.metadata)

  def test_complex_metrics_bulk_loading(self):
    metrics = io_types.ClassificationMetrics()
    metrics.log_roc_curve(
        fpr=[85.1, 85.1, 85.1],
        tpr=[52.6, 52.6, 52.6],
        threshold=[53.6, 53.6, 53.6])
    metrics.log_confusion_matrix(['dog', 'cat', 'horses'],
                                 [[2, 6, 0], [3, 5, 6], [5, 7, 8]])
    with open(
        os.path.join(
            os.path.dirname(__file__), 'test_data',
            'expected_io_types_bulk_load_classification_metrics.json')
    ) as json_file:
      expected_json = json.load(json_file)
      self.assertEqual(expected_json, metrics.metadata)

  def test_is_artifact_annotation(self):
    self.assertTrue(io_types.is_artifact_annotation(Input[Model]))
    self.assertTrue(io_types.is_artifact_annotation(Output[Model]))
    self.assertTrue(io_types.is_artifact_annotation(Output['MyArtifact']))

    self.assertFalse(io_types.is_artifact_annotation(Model))
    self.assertFalse(io_types.is_artifact_annotation(int))
    self.assertFalse(io_types.is_artifact_annotation('Dataset'))
    self.assertFalse(io_types.is_artifact_annotation(List[str]))
    self.assertFalse(io_types.is_artifact_annotation(Optional[str]))

  def test_is_input_artifact(self):
    self.assertTrue(io_types.is_input_artifact(Input[Model]))
    self.assertTrue(io_types.is_input_artifact(Input))

    self.assertFalse(io_types.is_input_artifact(Output[Model]))
    self.assertFalse(io_types.is_input_artifact(Output))

  def test_is_output_artifact(self):
    self.assertTrue(io_types.is_output_artifact(Output[Model]))
    self.assertTrue(io_types.is_output_artifact(Output))

    self.assertFalse(io_types.is_output_artifact(Input[Model]))
    self.assertFalse(io_types.is_output_artifact(Input))

  def test_get_io_artifact_class(self):
    self.assertEqual(io_types.get_io_artifact_class(Output[Model]), Model)

    self.assertEqual(io_types.get_io_artifact_class(Input), None)
    self.assertEqual(io_types.get_io_artifact_class(Output), None)
    self.assertEqual(io_types.get_io_artifact_class(Model), None)
    self.assertEqual(io_types.get_io_artifact_class(str), None)

  def test_get_io_artifact_annotation(self):
    self.assertEqual(
        io_types.get_io_artifact_annotation(Output[Model]), OutputAnnotation)
    self.assertEqual(
        io_types.get_io_artifact_annotation(Input[Model]), InputAnnotation)
    self.assertEqual(
        io_types.get_io_artifact_annotation(Input), InputAnnotation)
    self.assertEqual(
        io_types.get_io_artifact_annotation(Output), OutputAnnotation)

    self.assertEqual(io_types.get_io_artifact_annotation(Model), None)
    self.assertEqual(io_types.get_io_artifact_annotation(str), None)


if __name__ == '__main__':
  unittest.main()
