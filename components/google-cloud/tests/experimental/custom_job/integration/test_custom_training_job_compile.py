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
"""Test Custom training job to ensure the compile without error."""

import json
import os

from google_cloud_pipeline_components.experimental.custom_job import CustomTrainingJobOp
import kfp
from kfp import compiler

import unittest


class CustomTrainingJobCompileTest(unittest.TestCase):

  def setUp(self):
    super(CustomTrainingJobCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._test_input_string = "test_input_string"
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
    self._service_account = "fake_sa"
    self._display_name = "fake_job"
    self._labels = {'key1':'val1'}

  def tearDown(self):
    super(CustomTrainingJobCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def test_custom_training_job_op_compile(self):

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      _ = CustomTrainingJobOp(
          project=self._project,
          location=self._location,
          display_name=self._display_name,
          worker_pool_specs=self._worker_pool_specs,
          reserved_ip_ranges=['1.0.0.0'],
          labels=self._labels,
          service_account=self._service_account)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open(
        os.path.join(
            os.path.dirname(__file__),
            "../testdata/custom_training_job_pipeline.json")) as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparision
    del executor_output_json["sdkVersion"]
    del executor_output_json["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
