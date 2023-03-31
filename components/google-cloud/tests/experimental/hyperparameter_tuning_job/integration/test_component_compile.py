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

import json
import os

from google.cloud.aiplatform import hyperparameter_tuning as hpt
from google_cloud_pipeline_components.experimental.hyperparameter_tuning_job import HyperparameterTuningJobRunOp, serialize_parameters, serialize_metrics
import kfp
from kfp import compiler

import unittest


class HPTuningJobCompileTest(unittest.TestCase):

  def setUp(self):
    super(HPTuningJobCompileTest, self).setUp()
    self._display_name = "test_display_name"
    self._project = "test_project"
    self._location = "us-central1"
    self._package_path = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"), "pipeline.json")
    self._worker_pool_specs = [{
        "machine_spec": {
            "machine_type": "n1-standard-4",
            "accelerator_type": "NVIDIA_TESLA_T4",
            "accelerator_count": 1
        },
        "replica_count": 1,
        "container_spec": {
            "image_uri": "gcr.io/project_id/test"
        }
    }]
    self._metrics_spec = serialize_metrics({"accuracy": "maximize"})
    self._parameter_spec = serialize_parameters({
        "learning_rate": hpt.DoubleParameterSpec(min=0.001, max=1, scale="log"),
    })
    self._base_output_directory = "gs://my-bucket/blob"

  def tearDown(self):
    super(HPTuningJobCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_hyperparameter_tuning_job_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      _ = HyperparameterTuningJobRunOp(
          display_name=self._display_name,
          project=self._project,
          location=self._location,
          worker_pool_specs=self._worker_pool_specs,
          study_spec_metrics=self._metrics_spec,
          study_spec_parameters=self._parameter_spec,
          max_trial_count=10,
          parallel_trial_count=3,
          base_output_directory=self._base_output_directory)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open(
        os.path.join(
            os.path.dirname(__file__),
            "../testdata/hyperparameter_tuning_job_pipeline.json")) as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
