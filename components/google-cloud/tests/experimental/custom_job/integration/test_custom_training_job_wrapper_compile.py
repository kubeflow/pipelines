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

from google_cloud_pipeline_components.experimental.custom_job import utils
import kfp
from kfp import components
from kfp import compiler

import unittest


class CustomTrainingJobWrapperCompileTest(unittest.TestCase):

  def setUp(self):
    super(CustomTrainingJobWrapperCompileTest, self).setUp()
    self._project = "test_project"
    self._location = "us-central1"
    self._test_input_string = "test_input_string"
    self._package_path = "pipeline.json"
    self._container_component = components.load_component_from_text(
        "name: Producer\n"
        "inputs:\n"
        "- {name: input_text, type: String, description: 'Represents an input parameter.'}\n"
        "outputs:\n"
        "- {name: output_value, type: String, description: 'Represents an output paramter.'}\n"
        "implementation:\n"
        "  container:\n"
        "    image: google/cloud-sdk:latest\n"
        "    command:\n"
        "    - sh\n"
        "    - -c\n"
        "    - |\n"
        "      set -e -x\n"
        "      echo '$0, this is an output parameter' | gsutil cp - '$1'\n"
        "    - {inputValue: input_text}\n"
        "    - {outputPath: output_value}\n")
    self._python_componeont = self._create_a_pytnon_based_component()

  def tearDown(self):
    super(CustomTrainingJobWrapperCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

  def _create_a_pytnon_based_component(self):
    """Creates a test python based component factory."""

    @kfp.dsl.component
    def sum_numbers(a: int, b: int) -> int:
      return a + b

    return sum_numbers

  def test_container_based_custom_job_op_compile(self):

    custom_job_op = utils.create_custom_training_job_op_from_component(
        self._container_component)

    @kfp.dsl.pipeline(name="training-test")
    def pipeline():
      custom_job_task = custom_job_op(  # pylint: disable=unused-variable
          self._test_input_string,
          project=self._project,
          location=self._location)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)

    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open(
        os.path.join(
            os.path.dirname(__file__),
            "../testdata/custom_training_job_wrapper_pipeline.json")) as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparision
    del executor_output_json["pipelineSpec"]["sdkVersion"]
    del executor_output_json["pipelineSpec"]["schemaVersion"]
    self.assertEqual(executor_output_json, expected_executor_output_json)
