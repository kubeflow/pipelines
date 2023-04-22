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
# ============================================================================
"""Tests for convert_classification_export_for_batch_predict."""

import json
import os

from absl.testing import parameterized
from etils import epath
from google_cloud_pipeline_components.experimental.natural_language import ConvertDatasetExportForBatchPredictOp
import tensorflow as tf
import unittest

TESTS_ROOT = os.path.abspath(os.path.dirname(__file__))
TESTDATA_ROOT = os.path.abspath(os.path.join(TESTS_ROOT, 'testdata'))

SINGLE_LABEL_DATA_FILES = ['single_label.jsonl']
SINGLE_LABEL_SHARDED_DATA_FILES = [
    'single_label_sharded-00001-of-00002.jsonl',
    'single_label_sharded-00002-of-00002.jsonl',
]
MULTI_LABEL_DATA_FILES = ['multi_label.jsonl']
MULTI_LABEL_SHARDED_DATA_FILES = [
    'multi_label_sharded-00001-of-00002.jsonl',
    'multi_label_sharded-00002-of-00002.jsonl',
]
NON_EXISTENT_DATA_FILES = ['DOES_NOT_EXIST.jsonl']
INVALID_DATA_FILES = ['/asdf/']
EXPECTED_SINGLE_LABEL_DATA = [
    {'text': ['Test first sentence.'], 'labels': 'FirstClass'},
    {'text': ['Test second sentence.'], 'labels': 'SecondClass'},
    {'text': ['Sentence, the third one.'], 'labels': 'FirstClass'},
    {'text': ['Here is a fourth sentence.'], 'labels': 'SecondClass'},
    {'text': ['Five sentences and counting.'], 'labels': 'FirstClass'},
    {'text': ['Number six example sentence.'], 'labels': 'SecondClass'},
]
EXPECTED_MULTI_LABEL_DATA = [
    {'text': ['Test first sentence.'], 'labels': ['FirstClass']},
    {
        'text': ['Test second sentence.'],
        'labels': ['FirstClass', 'SecondClass'],
    },
    {
        'text': ['Sentence, the third one.'],
        'labels': ['FirstClass', 'SecondClass'],
    },
    {'text': ['Here is a fourth sentence.'], 'labels': ['SecondClass']},
    {'text': ['Five sentences and counting.'], 'labels': ['FirstClass']},
    {
        'text': ['Number six example sentence.'],
        'labels': ['FirstClass', 'SecondClass'],
    },
]


class ConvertClassificationExportForBatchPredictTest(parameterized.TestCase):
  """Unit tests for convert_classification_export_for_batch_predict."""

  def get_json_lines(self, output_files_tuple):
    output_data = []
    for output_file in output_files_tuple.output_files:
      json_str = epath.Path(output_file).read_text()
      for json_line in json_str.splitlines():
        output_data.append(json.loads(json_line))
    return output_data

  @parameterized.named_parameters(
      {
          'testcase_name': 'single_label',
          'files': SINGLE_LABEL_DATA_FILES,
          'classification_type': 'multiclass',
          'expected': EXPECTED_SINGLE_LABEL_DATA,
      },
      {
          'testcase_name': 'single_label_sharded',
          'files': SINGLE_LABEL_SHARDED_DATA_FILES,
          'classification_type': 'multiclass',
          'expected': EXPECTED_SINGLE_LABEL_DATA,
      },
      {
          'testcase_name': 'multi_label',
          'files': MULTI_LABEL_DATA_FILES,
          'classification_type': 'multilabel',
          'expected': EXPECTED_MULTI_LABEL_DATA,
      },
      {
          'testcase_name': 'multi_label_sharded',
          'files': MULTI_LABEL_SHARDED_DATA_FILES,
          'classification_type': 'multilabel',
          'expected': EXPECTED_MULTI_LABEL_DATA,
      },
  )
  def test_datagen(self, files, classification_type, expected):
    output_dir = epath.Path(self.create_tempdir().full_path)
    file_paths = [os.path.join(TESTDATA_ROOT, f) for f in files]
    output_files_tuple = ConvertDatasetExportForBatchPredictOp.python_func(
        file_paths=file_paths,
        classification_type=classification_type,
        output_dir=output_dir,
    )
    output_data = self.get_json_lines(output_files_tuple)
    self.assertCountEqual(output_data, expected)

  def test_load_dataset_non_existent(self):
    output_dir = epath.Path(self.create_tempdir().full_path)
    file_paths = [
        os.path.join(TESTDATA_ROOT, f) for f in NON_EXISTENT_DATA_FILES
    ]
    with self.assertRaises(tf.errors.NotFoundError):
      _ = ConvertDatasetExportForBatchPredictOp.python_func(
          file_paths=file_paths,
          classification_type='multiclass',
          output_dir=output_dir,
      )

  def test_datagen_invalid_data_paths(self):
    output_dir = epath.Path(self.create_tempdir().full_path)
    with self.assertRaises(tf.errors.NotFoundError):
      _ = ConvertDatasetExportForBatchPredictOp.python_func(
          file_paths=INVALID_DATA_FILES,
          classification_type='multiclass',
          output_dir=output_dir,
      )


if __name__ == '__main__':
  unittest.main()
