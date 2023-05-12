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

import os

from google_cloud_pipeline_components.experimental.notebooks import NotebooksExecutorOp
from google_cloud_pipeline_components.tests.v1 import utils
import kfp

import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._execution_id = 'notebook_executor_test'
    self._input_notebook_file = 'test_notebook_file'
    self._output_notebook_folder = 'test_output_folder'
    self._location = 'us-central1'
    self._master_type = 'n1-standard-4'
    self._container_image_uri = 'gcr.io/deeplearning-platform-release/base-cpu'

  def tearDown(self):
    super(ComponentsCompileTest, self).tearDown()

  def test_notebooks_executor_op_compile(self):
    @kfp.dsl.pipeline(name='notebooks-executor-test')
    def pipeline():
      NotebooksExecutorOp(
          project=self._project,
          location=self._location,
          execution_id=self._execution_id,
          input_notebook_file=self._input_notebook_file,
          output_notebook_folder=self._output_notebook_folder,
          master_type=self._master_type,
          container_image_uri=self._container_image_uri,
          parameters=(
              f'PROJECT_ID={self._project},EXECUTION_ID={self._execution_id}'
          ),
      )
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            '../testdata/notebooks_executor_component_pipeline.json',
        ),
    )
