# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Test google-cloud-pipeline-Components to ensure they compile correctly."""

import json
import os
from google_cloud_pipeline_components.experimental.evaluation import ModelEvaluationOp
from google_cloud_pipeline_components.experimental.evaluation import EvaluationDataSamplerOp
from google_cloud_pipeline_components.experimental.evaluation import EvaluationDataSplitterOp
import kfp
from kfp.v2 import compiler
import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._pipeline_root = 'gs://test_pipeline_root'
    self._gcs_source = 'gs://test_gcs_source'
    self._problem_type = 'test-problem-type'
    self._package_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'pipeline.json')

  def tearDown(self):
    super(ComponentsCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_model_evaluation_op_compile(self):

    @kfp.dsl.pipeline(name='model-evaluation-test')
    def pipeline():
      ModelEvaluationOp(
          project=self._project,
          location=self._location,
          root_dir=self._pipeline_root,
          problem_type=self._problem_type,
          classification_type='multiclass',
          ground_truth_column='ground_truth',
          prediction_score_column='prediction.scores',
          prediction_label_column='prediction.classes',
          class_names=['1', '0'],
          dataflow_service_account='test@developer.gserviceaccount.com',
          predictions_format='jsonl',
          batch_prediction_job={'artifacts': [{
              'test_key': 'test_value'
          }]},
          dataflow_workers_num=15)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open('testdata/model_evaluation_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertDictEqual(executor_output_json, expected_executor_output_json)

  def test_evaluation_data_sampler_op_compile(self):

    @kfp.dsl.pipeline(name='evaluation-data-sampler-test')
    def pipeline():
      EvaluationDataSamplerOp(
          project=self._project,
          location=self._location,
          root_dir=self._pipeline_root,
          sample_size=10000,
          gcs_source_uris=[self._gcs_source],
          dataflow_service_account='test@developer.gserviceaccount.com',
          instances_format='jsonl')

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open('testdata/evaluation_data_sampler_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertDictEqual(executor_output_json, expected_executor_output_json)

  def test_evaluation_data_splitter_op_compile(self):

    @kfp.dsl.pipeline(name='evaluation-data-splitter-test')
    def pipeline():
      EvaluationDataSplitterOp(
          project=self._project,
          location=self._location,
          root_dir=self._pipeline_root,
          gcs_source_uris=[self._gcs_source],
          dataflow_service_account='test@developer.gserviceaccount.com',
          instances_format='jsonl',
          ground_truth_column='ground_truth')

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open(
        'testdata/evaluation_data_splitter_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertDictEqual(executor_output_json, expected_executor_output_json)
