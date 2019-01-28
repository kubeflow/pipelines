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

from googleapiclient import errors
from ml_engine.create_job_op import CreateJobOp
from ml_engine.client import MLEngineClient
import ml_engine

@mock.patch('ml_engine.create_job_op.gcp_common.dump_file')
@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
@mock.patch('ml_engine.create_job_op.MLEngineClient')
class TestCreateJobOp(unittest.TestCase):

    def test_init(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        op = CreateJobOp('mock_project', job)

        self.assertEqual('mock_project', op._project_id)
        self.assertEqual(job['jobId'], op._job_id)
        self.assertEqual(job, op._job)

    def test_execute_submit_job_success(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        returned_job = {
            'jobId': 'mock_job',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = returned_job
        op = CreateJobOp('mock_project', job)

        result = op.execute()

        self.assertEqual(returned_job, result)

    def test_execute_retry_job_success(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        returned_job = {
            'jobId': 'mock_job',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().create_job.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        mock_mlengine_client().get_job.return_value = returned_job
        op = CreateJobOp('mock_project', job)

        result = op.execute()

        self.assertEqual(returned_job, result)

    def test_execute_conflict_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        returned_job = {
            'jobId': 'mock_job',
            'trainingInput': {
                'modelDir': 'test'
            },
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().create_job.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        mock_mlengine_client().get_job.return_value = returned_job
        op = CreateJobOp('mock_project', job)

        with self.assertRaises(errors.HttpError) as context:
            op.execute()

        self.assertEqual(409, context.exception.resp.status)

    def test_execute_create_job_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        mock_mlengine_client().create_job.side_effect = errors.HttpError(
            resp = mock.Mock(status=400),
            content = b'bad request'
        )
        op = CreateJobOp('mock_project', job)

        with self.assertRaises(errors.HttpError) as context:
            op.execute()

        self.assertEqual(400, context.exception.resp.status)
    
    def test_execute_job_status_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        job = {
            'jobId': 'mock_job'
        }
        returned_job = {
            'jobId': 'mock_job',
            'trainingInput': {
                'modelDir': 'test'
            },
            'state': 'FAILED'
        }
        mock_mlengine_client().get_job.return_value = returned_job
        op = CreateJobOp('mock_project', job)

        with self.assertRaises(RuntimeError):
            op.execute()