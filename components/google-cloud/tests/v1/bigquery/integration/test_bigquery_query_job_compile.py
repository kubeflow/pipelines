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

from google_cloud_pipeline_components.v1.bigquery import BigqueryQueryJobOp
from google_cloud_pipeline_components.tests.v1 import utils
import kfp

import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._query = 'SELECT * FROM foo_bar;'
    self._query_parameters = [{'name': 'foo'}, {'name': 'bar'}]
    self._job_configuration_query = {'priority': 'high'}
    self._labels = {'key1': 'val1'}
    self._encryption_spec_key_name = 'fake_encryption_key'

  def test_bigquery_query_job_op_compile(self):
    @kfp.dsl.pipeline(name='bigquery-test')
    def pipeline():
      BigqueryQueryJobOp(
          project=self._project,
          location=self._location,
          query=self._query,
          query_parameters=self._query_parameters,
          job_configuration_query=self._job_configuration_query,
          labels=self._labels,
          encryption_spec_key_name=self._encryption_spec_key_name,
      )
    utils.assert_pipeline_equals_golden(
        self,
        pipeline,
        os.path.join(
            os.path.dirname(__file__),
            '../testdata/bigquery_query_job_component_pipeline.json',
        ),
    )
