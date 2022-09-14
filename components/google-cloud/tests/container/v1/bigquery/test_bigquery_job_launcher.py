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
"""Test Vertex AI Bigquery Launchers Client module."""

import os
from unittest import mock

from google_cloud_pipeline_components.container.v1.bigquery.query_job import launcher as query_job_launcher
from google_cloud_pipeline_components.container.v1.bigquery.create_model import launcher as create_model_launcher
from google_cloud_pipeline_components.container.v1.bigquery.predict_model import launcher as predict_model_launcher
from google_cloud_pipeline_components.container.v1.bigquery.export_model import launcher as export_model_launcher
from google_cloud_pipeline_components.container.v1.bigquery.evaluate_model import launcher as evaluate_model_launcher
from google_cloud_pipeline_components.container.v1.bigquery.query_job import remote_runner as query_job_remote_runner
from google_cloud_pipeline_components.container.v1.bigquery.create_model import remote_runner as create_model_remote_runner
from google_cloud_pipeline_components.container.v1.bigquery.predict_model import remote_runner as predict_model_remote_runner
from google_cloud_pipeline_components.container.v1.bigquery.export_model import remote_runner as export_model_remote_runner
from google_cloud_pipeline_components.container.v1.bigquery.evaluate_model import remote_runner as evaluate_model_remote_runner

import unittest


class LauncherBigqueryQueryJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherBigqueryQueryJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'BigqueryQueryJob', '--project', 'test_project', '--location',
        'us_central1', '--payload', 'test_payload',
        '--job_configuration_query_override', '{}', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      query_job_remote_runner, 'bigquery_query_job', autospec=True)
  def test_launcher_on_bigquery_query_job_type(self, mock_bigquery_query_job):
    query_job_launcher.main(self._input_args)
    mock_bigquery_query_job.assert_called_once_with(
        type='BigqueryQueryJob',
        project='test_project',
        location='us_central1',
        payload='test_payload',
        job_configuration_query_override='{}',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')


class LauncherBigqueryCreateModelJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherBigqueryCreateModelJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'BigqueryCreateModelJob', '--project', 'test_project',
        '--location', 'us_central1', '--payload', 'test_payload',
        '--job_configuration_query_override', '{}', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      create_model_remote_runner, 'bigquery_create_model_job', autospec=True)
  def test_launcher_on_bigquery_create_model_job_type(
      self, mock_bigquery_create_model_job):
    create_model_launcher.main(self._input_args)
    mock_bigquery_create_model_job.assert_called_once_with(
        type='BigqueryCreateModelJob',
        project='test_project',
        location='us_central1',
        payload='test_payload',
        job_configuration_query_override='{}',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')


class LauncherBigqueryPredictModelJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherBigqueryPredictModelJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'BigqueryPredictModelJob', '--project', 'test_project',
        '--location', 'us_central1', '--model_name', 'test_model',
        '--table_name', 'test_table', '--query_statement', '', '--threshold',
        '0.5', '--payload', 'test_payload',
        '--job_configuration_query_override', '{}', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      predict_model_remote_runner, 'bigquery_predict_model_job', autospec=True)
  def test_launcher_on_bigquery_predict_model_job_type(
      self, mock_bigquery_predict_model_job):
    predict_model_launcher.main(self._input_args)
    mock_bigquery_predict_model_job.assert_called_once_with(
        type='BigqueryPredictModelJob',
        project='test_project',
        location='us_central1',
        model_name='test_model',
        table_name='test_table',
        query_statement='',
        threshold=0.5,
        payload='test_payload',
        job_configuration_query_override='{}',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')


class LauncherBigqueryExportModelJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherBigqueryExportModelJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'BigqueryExportModelJob', '--project', 'test_project',
        '--location', 'us_central1', '--payload', 'test_payload',
        '--model_name', 'test_model_name', '--model_destination_path',
        'gs://testbucket/testpath', '--exported_model_path',
        'exported_model_path', '--gcp_resources', self._gcp_resources,
        '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      export_model_remote_runner, 'bigquery_export_model_job', autospec=True)
  def test_launcher_on_bigquery_export_model_job_type(
      self, mock_bigquery_export_model_job):
    export_model_launcher.main(self._input_args)
    mock_bigquery_export_model_job.assert_called_once_with(
        type='BigqueryExportModelJob',
        project='test_project',
        location='us_central1',
        payload='test_payload',
        model_name='test_model_name',
        model_destination_path='gs://testbucket/testpath',
        exported_model_path='exported_model_path',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')


class LauncherBigqueryEvaluateModelJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherBigqueryEvaluateModelJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'BigqueryEvaluateModelJob', '--project', 'test_project',
        '--location', 'us_central1', '--model_name', 'test_model',
        '--table_name', 'test_table', '--query_statement', '', '--threshold',
        '0.5', '--payload', 'test_payload',
        '--job_configuration_query_override', '{}', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  @mock.patch.object(
      evaluate_model_remote_runner,
      'bigquery_evaluate_model_job',
      autospec=True)
  def test_launcher_on_bigquery_evaluate_model_job_type(
      self, mock_bigquery_evaluate_model_job):
    evaluate_model_launcher.main(self._input_args)
    mock_bigquery_evaluate_model_job.assert_called_once_with(
        type='BigqueryEvaluateModelJob',
        project='test_project',
        location='us_central1',
        model_name='test_model',
        table_name='test_table',
        query_statement='',
        threshold=0.5,
        payload='test_payload',
        job_configuration_query_override='{}',
        gcp_resources=self._gcp_resources,
        executor_input='executor_input')
