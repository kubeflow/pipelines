# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import mock
import unittest

from ml_engine.ml_engine import MLEngine

class TestMLEngine(unittest.TestCase):

    @mock.patch('ml_engine.ml_engine.CreateJobOp')
    def test_train_succeed(self, mock_create_job_op):
        MLEngine(wait_interval=15).train('project-1', 'job-1', ['gs://package/uri'], 
            'module-1', 'region-1', args=['--arg1 0', '--arg2 value2'],
            scale_tier='BASIC', runtime_version='1.10', python_version='2.7',
            job_dir='gs://job/dir', master_type='standard', 
            worker_type='large_model', parameter_server_type='cloud_tpu',
            worker_count='2', parameter_server_count='1', hyperparameters={
                'goal': 'MINIMIZE',
                'params': [{
                    'parameterName': 'arg1',
                    'type': 'DOUBLE',
                    'minValue': 0,
                    'maxValue': 999
                }]
            }, labels={'key': 'value'})
        
        expected_job = {
            'jobId': 'job-1',
            'labels': {'key': 'value'},
            'trainingInput': {
                'scaleTier': 'BASIC',
                'masterType': 'standard',
                'workerType': 'large_model',
                'parameterServerType': 'cloud_tpu',
                'workerCount': '2',
                'parameterServerCount': '1',
                'packageUris': ['gs://package/uri'],
                'pythonModule': 'module-1',
                'args': ['--arg1 0', '--arg2 value2'],
                'hyperparameters': {
                    'goal': 'MINIMIZE',
                    'params': [{
                        'parameterName': 'arg1',
                        'type': 'DOUBLE',
                        'minValue': 0,
                        'maxValue': 999
                    }]
                },
                'region': 'region-1',
                'jobDir': 'gs://job/dir',
                'runtimeVersion': '1.10',
                'pythonVersion': '2.7',
            }
        }
        mock_create_job_op.assert_called_once_with(
            'project-1',
            expected_job,
            15
        )

    @mock.patch('ml_engine.ml_engine.CreateJobOp')
    def test_train_missing_required(self, mock_create_job_op):
        self._assert_value_error(
            lambda: MLEngine().train('', 'job-1', ['gs://package/uri'], 
                'module-1', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().train('project-1', '', ['gs://package/uri'], 
                'module-1', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().train('project-1', 'job-1', '', 
                'module-1', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().train('project-1', 'job-1', ['gs://package/uri'], 
                '', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().train('project-1', 'job-1', ['gs://package/uri'], 
                'module-1', ''))

    @mock.patch('ml_engine.ml_engine.CreateJobOp')
    def test_batch_predict_succeed(self, mock_create_job_op):
        MLEngine(wait_interval=15).batch_predict('project-1', 'job-1', ['gs://input/path'], 
            'TEXT', 'gs://output/path', 'region-1', 
            model_name='projects/YOUR_PROJECT/models/YOUR_MODEL', 
            output_data_format='JSON', max_worker_count='2', runtime_version='1.10',
            batch_size='4', signature_name='serving_default',
            accelerator={
                'count': '2',
                'type': 'NVIDIA_TESLA_K80'
            }, labels={'key': 'value'})
        
        expected_job = {
            'jobId': 'job-1',
            'labels': {'key': 'value'},
            'predictionInput': {
                'dataFormat': 'TEXT',
                'outputDataFormat': 'JSON',
                'inputPaths': ['gs://input/path'],
                'outputPath': 'gs://output/path',
                'maxWorkerCount': '2',
                'region': 'region-1',
                'runtimeVersion': '1.10',
                'batchSize': '4',
                'signatureName': 'serving_default',
                'accelerator': {
                    'count': '2',
                    'type': 'NVIDIA_TESLA_K80'
                },
                'modelName': 'projects/YOUR_PROJECT/models/YOUR_MODEL',
            }
        }
        mock_create_job_op.assert_called_once_with(
            'project-1',
            expected_job,
            15
        )

    @mock.patch('ml_engine.ml_engine.CreateJobOp')
    def test_batch_predict_missing_required(self, mock_create_job_op):
        self._assert_value_error(
            lambda: MLEngine().batch_predict('', 'job-1', ['gs://input/path'], 
            'TEXT', 'gs://output/path', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().batch_predict('project-1', '', ['gs://input/path'], 
            'TEXT', 'gs://output/path', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().batch_predict('project-1', 'job-1', '', 
            'TEXT', 'gs://output/path', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().batch_predict('project-1', 'job-1', ['gs://input/path'], 
            '', 'gs://output/path', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().batch_predict('project-1', 'job-1', ['gs://input/path'], 
            'TEXT', '', 'region-1'))
        self._assert_value_error(
            lambda: MLEngine().batch_predict('project-1', 'job-1', ['gs://input/path'], 
            'TEXT', 'gs://output/path', ''))

    @mock.patch('ml_engine.ml_engine.CreateModelOp')
    def test_create_model_succeed(self, mock_create_model_op):
        MLEngine().create_model('project-1', 'model-1', 
            description='model 1', regions=['region-1'],
            online_prediction_logging=True, 
            labels={'key': 'value'})
        
        expected_model = {
            'name': 'model-1',
            'labels': {'key': 'value'},
            'description': 'model 1',
            'regions': ['region-1'],
            'onlinePredictionLogging': True,
        }
        mock_create_model_op.assert_called_once_with(
            'project-1',
            expected_model
        )
    @mock.patch('ml_engine.ml_engine.CreateModelOp')
    def test_create_model_missing_required(self, mock_create_model_op):
        self._assert_value_error(
            lambda: MLEngine().create_model('', 'model-1'))
        self._assert_value_error(
            lambda: MLEngine().create_model('project-1', ''))

    @mock.patch('ml_engine.ml_engine.DeleteVersionOp')
    def test_delete_version_succeed(self, mock_delete_version_op):
        MLEngine(wait_interval=15).delete_version(
            'project-1', 'model-1', 'version-1')

        mock_delete_version_op.assert_called_once_with(
            'project-1',
            'model-1',
            'version-1',
            15)

    @mock.patch('ml_engine.ml_engine.DeleteVersionOp')
    def test_delete_version_missing_required(self, mock_delete_version_op):
        self._assert_value_error(
            lambda: MLEngine().delete_version('', 'model-1', 'version-1'))
        self._assert_value_error(
            lambda: MLEngine().delete_version('project-1', '',  'version-1'))
        self._assert_value_error(
            lambda: MLEngine().delete_version('project-1', 'model-1',  ''))

    @mock.patch('ml_engine.ml_engine.CreateVersionOp')
    def test_create_version_succeed(self, mock_create_version_op):
        MLEngine(wait_interval=15).create_version('project-1', 'model-1', 
            'version-1', 'gs://deployment/uri',
            description='version 1', runtime_version='1.10',
            machine_type='mls1-highmem-1', framework='TENSORFLOW',
            python_version='3.5', manual_scaling={
                'nodes': 2
            }, replace_existing=True, labels={'key': 'value'})
        
        expected_version = {
            'name': 'version-1',
            'description': 'version 1',
            'deploymentUri': 'gs://deployment/uri',
            'runtimeVersion': '1.10',
            'machineType': 'mls1-highmem-1',
            'labels': {'key': 'value'},
            'framework': 'TENSORFLOW',
            'pythonVersion': '3.5',
            'manualScaling': {
                'nodes': 2
            }
        }
        mock_create_version_op.assert_called_once_with(
            'project-1',
            'model-1',
            expected_version,
            replace_existing=True,
            wait_interval=15
        )

    @mock.patch('ml_engine.ml_engine.CreateVersionOp')
    def test_create_version_missing_required(self, mock_create_version_op):
        self._assert_value_error(
            lambda: MLEngine().create_version('', 'model-1', 'version-1', 
                'gs://deployment/uri'))
        self._assert_value_error(
            lambda: MLEngine().create_version('project-1', '', 'version-1', 
                'gs://deployment/uri'))
        self._assert_value_error(
            lambda: MLEngine().create_version('project-1', 'model-1', '', 
                'gs://deployment/uri'))
        self._assert_value_error(
            lambda: MLEngine().create_version('project-1', 'model-1', 'version-1', 
                ''))

    def _assert_value_error(self, func):
        with self.assertRaises(ValueError):
            func()