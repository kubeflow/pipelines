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
"""Test Vertex AI Launcher Client module."""

import os

from google_cloud_pipeline_components.container.v1.dataproc import launcher

import unittest
from unittest import mock


class LauncherUnsupportedJobTypeTests(unittest.TestCase):

  def setUp(self):
    super(LauncherUnsupportedJobTypeTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'Unknown', '--project', 'test_project', '--location',
        'us_central1', '--payload', 'test_payload', '--model_name',
        'test_model_name', '--model_destination_path',
        'gs://testbucket/testpath', '--gcp_resources', self._gcp_resources,
        '--executor_input', 'executor_input'
    ]

  def test_launcher_unsupported_job_type(self):
    with self.assertRaises(ValueError) as context:
      launcher.main(self._input_args)
    self.assertEqual('Unsupported job type: Unknown', str(context.exception))


class LauncherDataprocPySparkBatchJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDataprocPySparkBatchJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DataprocPySparkBatch', '--project', 'test_project',
        '--location', 'us_central1', '--batch_id', 'test_batch_id',
        '--payload', 'test_payload',
        '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_dataproc_create_pysparkbatch_job_type(self):
    mock_dataproc_pysparkbatch_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'DataprocPySparkBatch': mock_dataproc_pysparkbatch_job}):
      launcher.main(self._input_args)
      mock_dataproc_pysparkbatch_job.assert_called_once_with(
          type='DataprocPySparkBatch',
          project='test_project',
          location='us_central1',
          batch_id='test_batch_id',
          payload='test_payload',
          gcp_resources=self._gcp_resources)


class LauncherDataprocSparkBatchJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDataprocSparkBatchJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DataprocSparkBatch', '--project', 'test_project',
        '--location', 'us_central1', '--batch_id', 'test_batch_id',
        '--payload', 'test_payload',
        '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_dataproc_create_pysparkbatch_job_type(self):
    mock_dataproc_pysparkbatch_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'DataprocSparkBatch': mock_dataproc_pysparkbatch_job}):
      launcher.main(self._input_args)
      mock_dataproc_pysparkbatch_job.assert_called_once_with(
          type='DataprocSparkBatch',
          project='test_project',
          location='us_central1',
          batch_id='test_batch_id',
          payload='test_payload',
          gcp_resources=self._gcp_resources)


class LauncherDataprocSparkRBatchJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDataprocSparkRBatchJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DataprocSparkRBatch', '--project', 'test_project',
        '--location', 'us_central1', '--batch_id', 'test_batch_id',
        '--payload', 'test_payload',
        '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_dataproc_create_pysparkbatch_job_type(self):
    mock_dataproc_pysparkbatch_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'DataprocSparkRBatch': mock_dataproc_pysparkbatch_job}):
      launcher.main(self._input_args)
      mock_dataproc_pysparkbatch_job.assert_called_once_with(
          type='DataprocSparkRBatch',
          project='test_project',
          location='us_central1',
          batch_id='test_batch_id',
          payload='test_payload',
          gcp_resources=self._gcp_resources)


class LauncherDataprocSparkSqlBatchJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDataprocSparkSqlBatchJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DataprocSparkSqlBatch', '--project', 'test_project',
        '--location', 'us_central1', '--batch_id', 'test_batch_id',
        '--payload', 'test_payload',
        '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_dataproc_create_pysparkbatch_job_type(self):
    mock_dataproc_pysparkbatch_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'DataprocSparkSqlBatch': mock_dataproc_pysparkbatch_job}):
      launcher.main(self._input_args)
      mock_dataproc_pysparkbatch_job.assert_called_once_with(
          type='DataprocSparkSqlBatch',
          project='test_project',
          location='us_central1',
          batch_id='test_batch_id',
          payload='test_payload',
          gcp_resources=self._gcp_resources)
