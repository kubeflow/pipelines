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
import unittest
import kfp
from kfp import components
from kfp.v2 import compiler
from google_cloud_pipeline_components.experimental.custom_job import custom_training_job_op


class CustomJobCompileTest(unittest.TestCase):

    def setUp(self):
        super(CustomJobCompileTest, self).setUp()
        self._project = "test_project"
        self._location = "us-central1"
        self._test_input_string = "test_input_string"

        self._display_name = "test_display_name"
        self._model_display_name = "test_model_display_name"
        self._gcs_source = "gs://test_gcs_source"
        self._gcs_output_dir = "gs://test_gcs_output_dir"
        self._pipeline_root = "gs://test_pipeline_root"
        self._gcs_destination_prefix = "gs://test_gcs_output_dir/batch_prediction"
        self._serving_container_image_uri = "gcr.io/test_project/test_image:test_tag"
        self._artifact_uri = "project/test_artifact_uri"
        self._package_path = "pipeline.json"
        self._test_component = components.load_component_from_text(
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

    def tearDown(self):
        if os.path.exists(self._package_path):
            os.remove(self._package_path)

    def test_custom_job_op_compile(self):

        custom_job_op = custom_training_job_op(self._test_component)

        @kfp.dsl.pipeline(name="training-test")
        def pipeline():
            custom_job_task = custom_job_op(
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
                    '../testdata/custom_job_pipeline.json')) as ef:
            expected_executor_output_json = json.load(ef, strict=False)
        # Ignore the kfp SDK version during comparision
        del executor_output_json['pipelineSpec']['sdkVersion']
        self.assertEqual(executor_output_json, expected_executor_output_json)
