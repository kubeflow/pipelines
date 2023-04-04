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
"""Test natural language components."""

import os

from google_cloud_pipeline_components.experimental import natural_language
from kfp import compiler
from kfp import dsl

import unittest


class NaturalLanguageComponentsImportTest(unittest.TestCase):
  """Tests for importing natural language components."""

  def setUp(self):
    super().setUp()
    self._project = 'test_project'
    self._display_name = 'test_display_name'
    self._package_path = self.create_tempfile('pipeline.json').full_path
    self._location = 'us-central1'
    self._input_paths = ['gs://test_project/data-00000-of-00001.jsonl']
    self._natural_language_task_type = 'CLASSIFICATION'

  def tearDown(self):
    super().tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_convert_dataset_export_compile(self):
    """Checks that convert dataset component compiles."""

    @dsl.pipeline(name='test-convert-dataset-export-for-batch-predict')
    def pipeline():
      _ = natural_language.ConvertDatasetExportForBatchPredictOp(
          file_paths=self._input_paths, classification_type='multiclass')

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

  def test_train_text_classification_compile(self):
    """Checks that trainer component compiles."""

    @dsl.pipeline(name='test-train-text-classification')
    def pipeline():
      _ = natural_language.TrainTextClassificationOp(
          project=self._project,
          location=self._location,
          input_data_path='gs://test_project/data-00000-of-00001.jsonl',
          input_format='jsonl',
          natural_language_task_type=self._natural_language_task_type,
      )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)


if __name__ == '__main__':
  unittest.main()
