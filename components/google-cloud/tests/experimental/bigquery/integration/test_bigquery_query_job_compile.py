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

import json
import os
from google_cloud_pipeline_components.experimental.bigquery import BigqueryQueryJobOp
import kfp
from kfp import compiler
import unittest


class ComponentsCompileTest(unittest.TestCase):

  def setUp(self):
    super(ComponentsCompileTest, self).setUp()
    self._project = 'test_project'
    self._location = 'us-central1'
    self._query = 'SELECT * FROM foo_bar;'
    self._query_parameters = [{'name':'foo'},{'name':'bar'}]
    self._job_configuration_query = {'priority':'high'}
    self._labels = {'key1':'val1'}
    self._encryption_spec_key_name = 'fake_encryption_key'
    self._package_path = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'pipeline.json')

  def tearDown(self):
    super(ComponentsCompileTest, self).tearDown()
    if os.path.exists(self._package_path):
      os.remove(self._package_path)

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
          encryption_spec_key_name=self._encryption_spec_key_name)

    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=self._package_path)
    with open(self._package_path) as f:
      executor_output_json = json.load(f, strict=False)

    with open('testdata/bigquery_query_job_component_pipeline.json') as ef:
      expected_executor_output_json = json.load(ef, strict=False)

    # Ignore the kfp SDK & schema version during comparison
    del executor_output_json['sdkVersion']
    del executor_output_json['schemaVersion']
    self.assertDictEqual(executor_output_json, expected_executor_output_json)
