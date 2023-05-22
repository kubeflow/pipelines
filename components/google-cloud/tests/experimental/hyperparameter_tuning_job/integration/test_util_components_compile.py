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
"""Test google-cloud-pipeline-Components to ensure the compile without error."""

import os

from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import GetBestHyperparametersOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import GetBestTrialOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import GetHyperparametersOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import GetTrialsOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import GetWorkerPoolSpecsOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import IsMetricBeyondThresholdOp
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import serialize_metrics
from google_cloud_pipeline_components.tests.v1 import utils
import kfp

import unittest


class UtilComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(UtilComponentsCompileTest, self).setUp()
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json"
    )
    self._gcp_resources = (
        '{ "resources": [ { "resourceType": '
        '"HyperparameterTuningJob", "resourceUri": '
        '"https://us-central1-aiplatform.googleapis.com/'
        "v1/projects/186556260430/locations/us-central1/"
        'hyperparameterTuningJobs/1234567890123456789" '
        "} ] }"
    )
    self._trial = (
        '{\n "id": "1",\n "state": 4,\n "parameters": [\n '
        '{\n "parameterId": "learning_rate",\n "value": '
        '0.03162277660168379\n },\n {\n "parameterId": '
        '"momentum",\n "value": 0.5\n },\n {\n "parameterId":'
        ' "num_neurons",\n "value": 128.0\n }\n ],\n '
        '"finalMeasurement": {\n "stepCount": "10",\n '
        '"metrics": [\n {\n "metricId": '
        '"accuracy",\n "value": 0.734375\n }\n ]\n },\n '
        '"startTime": "2021-12-10T00:41:57.675086142Z",\n '
        '"endTime": "2021-12-10T00:52:35Z",\n "name": "",\n '
        '"measurements": [],\n "clientId": "",\n '
        '"infeasibleReason": "",\n "customJob": ""\n}'
    )
    self._best_hp = [
        '{\n "parameterId": "learning_rate",\n "value": 0.028\n}',
        '{\n "parameterId": "momentum",\n "value": 0.49\n}',
        '{\n "parameterId": "num_neurons",\n "value": 512.0\n}',
    ]
    self._worker_pool_specs = [{
        "machine_spec": {
            "machine_type": "n1-standard-4",
            "accelerator_type": "NVIDIA_TESLA_T4",
            "accelerator_count": 1,
        },
        "replica_count": 1,
        "container_spec": {"image_uri": "gcr.io/project_id/test"},
    }]
    self._metrics_spec = serialize_metrics({"accuracy": "maximize"})

  def tearDown(self):
    super(UtilComponentsCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_get_trials_op_compile(self):
    @kfp.dsl.pipeline(name="get-trials-op-test")
    def pipeline():
      _ = GetTrialsOp(gcp_resources=self._gcp_resources)

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "get_trials_op_pipeline.json",
        ),
    )

  def test_get_best_trial_op_compile(self):
    print("TEST TEST TEST")
    print(GetBestTrialOp.pipeline_spec)
    @kfp.dsl.pipeline(name="get-best-trial-op-test")
    def pipeline():
      _ = GetBestTrialOp(
          trials=[self._trial], study_spec_metrics=self._metrics_spec
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "get_best_trial_op_pipeline.json",
        ),
    )

  def test_get_best_hyperparameters_op_compile(self):
    @kfp.dsl.pipeline(name="get-best-hyperparameters-op-test")
    def pipeline():
      _ = GetBestHyperparametersOp(
          trials=[self._trial], study_spec_metrics=self._metrics_spec
      )

    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "get_best_hyperparameters_op_pipeline.json",
        ),
    )

  def test_get_hyperparameters_op_compile(self):
    @kfp.dsl.pipeline(name="get-hyperparameters-op-test")
    def pipeline():
      _ = GetHyperparametersOp(trial=self._trial)
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "get_hyperparameters_op_pipeline.json",
        ),
    )

  def test_get_worker_pool_specs_op_compile(self):
    @kfp.dsl.pipeline(name="get-worker-pool-specs-op-test")
    def pipeline():
      _ = GetWorkerPoolSpecsOp(
          best_hyperparameters=self._best_hp,
          worker_pool_specs=self._worker_pool_specs,
      )
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "get_worker_pool_specs_op_pipeline.json",
        ),
    )

  def test_is_metric_beyond_threshold_op_compile(self):
    @kfp.dsl.pipeline(name="is-metric-beyond-threshold-op-test")
    def pipeline():
      _ = IsMetricBeyondThresholdOp(
          trial=self._trial,
          study_spec_metrics=self._metrics_spec,
          threshold=0.5,
      )
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(os.path.dirname(__file__)),
            "testdata",
            "is_metric_beyond_threshold_pipeline.json",
        ),
    )
