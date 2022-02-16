# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
'''Test Evaluation component to ensure the compile without error.'''

import json
import os

from google_cloud_pipeline_components.experimental.evaluation import ModelEvaluationOp
# from google_cloud_pipeline_components.v1.model import ModelUploadOp
from google_cloud_pipeline_components.types import artifact_types
import kfp
from kfp.v2 import compiler
from kfp.v2.components import importer_node

import unittest


class EvaluationCompileTest(unittest.TestCase):

  def setUp(self):
    super(EvaluationCompileTest, self).setUp()
    self._project = 'test-project'
    self._location = 'us-central1'
    self._root_dir = 'test-directory'
    self._problem_type = 'test-problem-type'
    self._predictions_format = 'test-predictions-format'
    self._classification_type = 'test-classification-type'
    self._class_names = ['test-class-1', 'test-class-2']
    self._ground_truth_column = 'test-ground-truth-column'
    self._batch_prediction_job_metadata = {
        'name': 'batch-prediction-name',
        'uri': 'batch-prediction-uri',
        'job_resource_name': 'batch-prediction-job-resource-name'
    }
    self._package_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'pipeline.json')

  def tearDown(self):
    super(EvaluationCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_evaluation_op_compile(self):

    @kfp.dsl.pipeline(name='evaluation-test')
    def pipeline():
      import_batch_prediction_op = importer_node.importer(
          artifact_uri='asd',
          artifact_class=artifact_types.VertexBatchPredictionJob,
          metadata=self._batch_prediction_job_metadata
      )

      model_eval_op = ModelEvaluationOp(
          project=self._project,
          location=self._location,
          root_dir=self._root_dir,
          problem_type=self._problem_type,
          predictions_format=self._predictions_format,
          classification_type=self._classification_type,
          batch_prediction_job=import_batch_prediction_op.output,
          class_names=self._class_names,
          ground_truth_column=self._ground_truth_column)

      # import_model_op = importer_node.importer(
      #     artifact_uri='asd',
      #     artifact_class=artifact_types.UnmanagedContainerModel,
      #     metadata={
      #       'containerSpec': {
      #           'imageUri':
      #               'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:prod'
      #     }
      # })
      # model_upload_op = ModelUploadOp(
      #     project='l',
      #     location='us-central1',
      #     display_name='name',
      #     unmanaged_container_model=import_model_op.output
      # )

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)
      print(executor_output_json)

    with open(
        os.path.join(
            os.path.dirname(__file__),
            '../testdata/evaluation_pipeline.json')) as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparision
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertEqual(executor_output_json, expected_executor_output_json)
