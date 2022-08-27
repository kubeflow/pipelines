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
from unittest import mock

from google_cloud_pipeline_components.container.v1.gcp_launcher import launcher

import unittest


class LauncherJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherJobUtilsTests, self).setUp()
    self._project = 'test_project'
    self._location = 'test_region'
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')

  def test_launcher_on_custom_job_type(self):
    job_type = 'CustomJob'
    payload = ('{"display_name": "ContainerComponent", "job_spec": '
               '{"worker_pool_specs": [{"machine_spec": {"machine_type": '
               '"n1-standard-4"}, "replica_count": 1, "container_spec": '
               '{"image_uri": "google/cloud-sdk:latest", "command": ["sh", '
               '"-c", "set -e -x\\necho \\"$0, this is an output '
               'parameter\\"\\n", "{{$.inputs.parameters[\'input_text\']}}", '
               '"{{$.outputs.parameters[\'output_value\'].output_file}}"]}}]}}')
    input_args = [
        '--type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        self._gcp_resources, '--extra_arg', 'extra_arg_value'
    ]
    mock_create_custom_job = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {job_type: mock_create_custom_job}):
      launcher.main(input_args)
      mock_create_custom_job.assert_called_once_with(
          type=job_type,
          project=self._project,
          location=self._location,
          payload=payload,
          gcp_resources=self._gcp_resources)

  def test_launcher_on_batch_prediction_job_type(self):
    job_type = 'BatchPredictionJob'
    payload = ('{"batchPredictionJob": {"displayName": '
               '"BatchPredictionComponentName", "model": '
               '"projects/test/locations/test/models/test-model","inputConfig":'
               ' {"instancesFormat": "CSV","gcsSource": {"uris": '
               '["test_gcs_source"]}}, "outputConfig": {"predictionsFormat": '
               '"CSV", "gcsDestination": {"outputUriPrefix": '
               '"test_gcs_destination"}}}}')
    input_args = [
        '--type', job_type, '--project', self._project, '--location',
        self._location, '--payload', payload, '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]
    mock_create_batch_prediction_job = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {job_type: mock_create_batch_prediction_job}):
      launcher.main(input_args)
      mock_create_batch_prediction_job.assert_called_once_with(
          type=job_type,
          project=self._project,
          location=self._location,
          payload=payload,
          gcp_resources=self._gcp_resources,
          executor_input='executor_input')


class LauncherUploadModelUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherUploadModelUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'UploadModel', '--project', 'test_project', '--location',
        'us_central1', '--payload', 'test_payload', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  def test_launcher_on_upload_model_type(self):
    mock_upload_model = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {'UploadModel': mock_upload_model}):
      launcher.main(self._input_args)
      mock_upload_model.assert_called_once_with(
          type='UploadModel',
          project='test_project',
          location='us_central1',
          payload='test_payload',
          gcp_resources=self._gcp_resources,
          executor_input='executor_input')


class LauncherDeleteModelUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDeleteModelUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DeleteModel', '--project', '', '--location', '', '--payload',
        'test_payload', '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_delete_model_type(self):
    mock_delete_model = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {'DeleteModel': mock_delete_model}):
      launcher.main(self._input_args)
      mock_delete_model.assert_called_once_with(
          type='DeleteModel',
          project='',
          location='',
          payload='test_payload',
          gcp_resources=self._gcp_resources)


class LauncherCreateEndpointUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherCreateEndpointUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'CreateEndpoint', '--project', 'test_project', '--location',
        'us_central1', '--payload', 'test_payload', '--gcp_resources',
        self._gcp_resources, '--executor_input', 'executor_input'
    ]

  def test_launcher_on_create_endpoint_type(self):
    mock_create_endpoint = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {'CreateEndpoint': mock_create_endpoint}):
      launcher.main(self._input_args)
      mock_create_endpoint.assert_called_once_with(
          type='CreateEndpoint',
          project='test_project',
          location='us_central1',
          payload='test_payload',
          gcp_resources=self._gcp_resources,
          executor_input='executor_input')


class LauncherDeleteEndpointUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDeleteEndpointUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DeleteEndpoint', '--project', '', '--location', '',
        '--payload', 'test_payload', '--gcp_resources', self._gcp_resources
    ]

  def test_launcher_on_delete_endpoint_type(self):
    mock_delete_endpoint = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {'DeleteEndpoint': mock_delete_endpoint}):
      launcher.main(self._input_args)
      mock_delete_endpoint.assert_called_once_with(
          type='DeleteEndpoint',
          project='',
          location='',
          payload='test_payload',
          gcp_resources=self._gcp_resources)


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

  def test_launcher_on_bigquery_query_job_type(self):
    mock_bigquery_query_job = mock.Mock()
    with mock.patch.dict(launcher._JOB_TYPE_TO_ACTION_MAP,
                         {'BigqueryQueryJob': mock_bigquery_query_job}):
      launcher.main(self._input_args)
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

  def test_launcher_on_bigquery_create_model_job_type(self):
    mock_bigquery_create_model_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'BigqueryCreateModelJob': mock_bigquery_create_model_job}):
      launcher.main(self._input_args)
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

  def test_launcher_on_bigquery_predict_model_job_type(self):
    mock_bigquery_predict_model_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'BigqueryPredictModelJob': mock_bigquery_predict_model_job}):
      launcher.main(self._input_args)
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

  def test_launcher_on_bigquery_export_model_job_type(self):
    mock_bigquery_export_model_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'BigqueryExportModelJob': mock_bigquery_export_model_job}):
      launcher.main(self._input_args)
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

  def test_launcher_on_bigquery_evaluate_model_job_type(self):
    mock_bigquery_evaluate_model_job = mock.Mock()
    with mock.patch.dict(
        launcher._JOB_TYPE_TO_ACTION_MAP,
        {'BigqueryEvaluateModelJob': mock_bigquery_evaluate_model_job}):
      launcher.main(self._input_args)
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


class LauncherDataprocPySparkBatchJobUtilsTests(unittest.TestCase):

  def setUp(self):
    super(LauncherDataprocPySparkBatchJobUtilsTests, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'),
        'test_file_path/test_file.txt')
    self._input_args = [
        '--type', 'DataprocPySparkBatch', '--project', 'test_project',
        '--location', 'us_central1', '--batch_id', 'test_batch_id', '--payload',
        'test_payload', '--gcp_resources', self._gcp_resources
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
        '--location', 'us_central1', '--batch_id', 'test_batch_id', '--payload',
        'test_payload', '--gcp_resources', self._gcp_resources
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
        '--location', 'us_central1', '--batch_id', 'test_batch_id', '--payload',
        'test_payload', '--gcp_resources', self._gcp_resources
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
        '--location', 'us_central1', '--batch_id', 'test_batch_id', '--payload',
        'test_payload', '--gcp_resources', self._gcp_resources
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


if __name__ == '__main__':
  unittest.main()
