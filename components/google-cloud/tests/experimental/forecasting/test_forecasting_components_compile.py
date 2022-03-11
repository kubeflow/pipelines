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
"""Test forecasting components to ensure they compile without error."""

import os

from google_cloud_pipeline_components import aiplatform
import kfp
from kfp.v2 import compiler

import unittest


class ForecastingComponetsCompileTest(unittest.TestCase):

    def setUp(self):
        super(ForecastingComponetsCompileTest, self).setUp()
        self._project = 'test_project'
        self._display_name = 'test_display_name'
        self._package_path = 'pipeline.json'
        self._location = 'us-central1'
        self._bq_source = 'bq://test_project.test_dataset.training_input'

    def tearDown(self):
        if os.path.exists(self._package_path):
            os.remove(self._package_path)

    def test_tabular_data_pipeline_component_ops_compile(self):
        # TODO(b/223823528): Add the other forecasting components.
        @kfp.dsl.pipeline(name='forecasting-training')
        def pipeline():
            aiplatform.TimeSeriesDatasetCreateOp(
                display_name=self._display_name,
                bq_source=self._bq_source,
                project=self._project,
                location=self._location,
            )

        compiler.Compiler().compile(
            pipeline_func=pipeline, package_path=self._package_path
        )
