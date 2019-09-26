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

from kfp_component.google.dataproc import submit_job

MODULE = 'kfp_component.google.dataproc._submit_job'

@mock.patch(MODULE + '.display.display')
@mock.patch(MODULE + '.gcp_common.dump_file')
@mock.patch(MODULE + '.KfpExecutionContext')
@mock.patch(MODULE + '.DataprocClient')
class TestSubmitJob(unittest.TestCase):

    def test_submit_job_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        expected_job = {
            'reference': {
                'projectId': 'mock-project'
            },
            'placement': {
                'clusterName': 'mock-cluster'
            }
        }
        returned_job = {
            'reference': {
                'projectId': 'mock-project',
                'jobId': 'mock-job'
            },
            'placement': {
                'clusterName': 'mock-cluster'
            },
            'status': {
                'state': 'DONE'
            }
        }
        mock_dataproc_client().submit_job.return_value = returned_job
        mock_dataproc_client().get_job.return_value = returned_job

        result = submit_job('mock-project', 'mock-region', 'mock-cluster', job)

        mock_dataproc_client().submit_job.assert_called_with('mock-project', 'mock-region',
            expected_job, request_id='ctx1')
        self.assertEqual(returned_job, result)

    def test_submit_job_failed_with_error(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'reference': {
                'projectId': 'mock-project',
                'jobId': 'mock-job'
            },
            'placement': {
                'clusterName': 'mock-cluster'
            },
            'status': {
                'state': 'ERROR',
                'details': 'mock error'
            }
        }
        mock_dataproc_client().submit_job.return_value = returned_job
        mock_dataproc_client().get_job.return_value = returned_job

        with self.assertRaises(RuntimeError):
            submit_job('mock-project', 'mock-region', 'mock-cluster', job)

    def test_cancel_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        job = {}
        returned_job = {
            'reference': {
                'projectId': 'mock-project',
                'jobId': 'mock-job'
            },
            'placement': {
                'clusterName': 'mock-cluster'
            },
            'status': {
                'state': 'DONE'
            }
        }
        mock_dataproc_client().submit_job.return_value = returned_job
        mock_dataproc_client().get_job.return_value = returned_job

        submit_job('mock-project', 'mock-region', 'mock-cluster', job)

        cancel_func = mock_kfp_context.call_args[1]['on_cancel']        
        cancel_func()

        mock_dataproc_client().cancel_job.assert_called_with(
            'mock-project',
            'mock-region',
            'mock-job'
        )