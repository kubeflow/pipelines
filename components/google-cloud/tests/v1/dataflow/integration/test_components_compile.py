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
"""Test google-cloud-pipeline-Components to ensure they compile correctly."""

import os

from google_cloud_pipeline_components.v1.dataflow import DataflowPythonJobOp
from google_cloud_pipeline_components.tests.v1 import utils
import kfp

import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._python_module_path = 'gs://python_module_path'
    self._requirements_file_path = 'gs://requirements_file_path'
    self._args = ['test_arg']
    self._gcs_source = 'gs://test_gcs_source'
    self._temp_location = 'gs://temp_location'
    self._pipeline_root = 'gs://test_pipeline_root'

  def tearDown(self):
    super(ComponentsCompileTest, self).tearDown()

  def test_dataflow_python_op_compile(self):
    @kfp.dsl.pipeline(name='dataflow-python-test')
    def pipeline():
      DataflowPythonJobOp(
          project=self._project,
          location=self._location,
          python_module_path=self._python_module_path,
          temp_location=self._temp_location,
          requirements_file_path=self._requirements_file_path,
          args=self._args,
      )
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            '../testdata/dataflow_python_job_component_pipeline.json',
        ),
    )
