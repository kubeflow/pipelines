
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
from kfp_component.google.ml_engine import create_job

CREATE_JOB_MODULE = 'kfp_component.google.ml_engine._create_job'
COMMON_OPS_MODEL = 'kfp_component.google.ml_engine._common_ops'

@mock.patch(COMMON_OPS_MODEL + '.display.display')
@mock.patch(COMMON_OPS_MODEL + '.gcp_common.dump_file')
@mock.patch(CREATE_JOB_MODULE + '.KfpExecutionContext')
@mock.patch(CREATE_JOB_MODULE + '.MLEngineClient')
class TestCreateJob(unittest.TestCase):

    def test_create_job_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'job_ctx1',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)

        result = create_job('mock_project', job)

        self.assertEqual(returned_job, result)
        mock_mlengine_client().create_job.assert_called_with(
            project_id = 'mock_project',
            job = {
                'jobId': 'job_ctx1'
            }
        )

    def test_create_job_with_job_id_prefix_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'mock_job_ctx1',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)

        result = create_job('mock_project', job, job_id_prefix='mock_job_')

        self.assertEqual(returned_job, result)
        mock_mlengine_client().create_job.assert_called_with(
            project_id = 'mock_project',
            job = {
                'jobId': 'mock_job_ctx1'
            }
        )

    def test_create_job_with_job_id_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'mock_job',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)

        result = create_job('mock_project', job, job_id='mock_job')

        self.assertEqual(returned_job, result)
        mock_mlengine_client().create_job.assert_called_with(
            project_id = 'mock_project',
            job = {
                'jobId': 'mock_job'
            }
        )
        
    def test_execute_retry_job_success(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'job_ctx1',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().create_job.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        mock_mlengine_client().get_job.return_value = returned_job
        
        result = create_job('mock_project', job)

        self.assertEqual(returned_job, result)

    def test_create_job_use_context_id_as_name(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        context_id = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'job_ctx1',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)
        mock_kfp_context().__enter__().context_id.return_value = context_id

        create_job('mock_project', job)

        mock_mlengine_client().create_job.assert_called_with(
            project_id = 'mock_project',
            job = {
                'jobId': 'job_ctx1'
            }
        )

    def test_execute_conflict_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'job_ctx1',
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

        with self.assertRaises(errors.HttpError) as context:
            create_job('mock_project', job)

        self.assertEqual(409, context.exception.resp.status)

    def test_execute_create_job_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        mock_mlengine_client().create_job.side_effect = errors.HttpError(
            resp = mock.Mock(status=400),
            content = b'bad request'
        )

        with self.assertRaises(errors.HttpError) as context:
            create_job('mock_project', job)

        self.assertEqual(400, context.exception.resp.status)
    
    def test_execute_job_status_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'mock_job',
            'trainingInput': {
                'modelDir': 'test'
            },
            'state': 'FAILED'
        }
        mock_mlengine_client().get_job.return_value = returned_job

        with self.assertRaises(RuntimeError):
            create_job('mock_project', job)

    def test_cancel_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'jobId': 'job_ctx1',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)
        create_job('mock_project', job)
        cancel_func = mock_kfp_context.call_args[1]['on_cancel']
        
        cancel_func()

        mock_mlengine_client().cancel_job.assert_called_with(
            'mock_project', 'job_ctx1'
        )
