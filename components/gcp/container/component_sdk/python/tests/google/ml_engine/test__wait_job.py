# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
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
from kfp_component.google.ml_engine import wait_job

CREATE_JOB_MODULE = 'kfp_component.google.ml_engine._wait_job'
COMMON_OPS_MODEL = 'kfp_component.google.ml_engine._common_ops'

@mock.patch(COMMON_OPS_MODEL + '.display.display')
@mock.patch(COMMON_OPS_MODEL + '.gcp_common.dump_file')
@mock.patch(CREATE_JOB_MODULE + '.KfpExecutionContext')
@mock.patch(CREATE_JOB_MODULE + '.MLEngineClient')
class TestWaitJob(unittest.TestCase):

    def test_wait_job_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        returned_job = {
            'jobId': 'mock_job',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)

        result = wait_job('mock_project', 'mock_job')

        self.assertEqual(returned_job, result)

    def test_wait_job_status_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        returned_job = {
            'jobId': 'mock_job',
            'trainingInput': {
                'modelDir': 'test'
            },
            'state': 'FAILED'
        }
        mock_mlengine_client().get_job.return_value = returned_job

        with self.assertRaises(RuntimeError):
            wait_job('mock_project', 'mock_job')

    def test_cancel_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        returned_job = {
            'jobId': 'mock_job',
            'state': 'SUCCEEDED'
        }
        mock_mlengine_client().get_job.return_value = (
            returned_job)
        wait_job('mock_project', 'mock_job')
        cancel_func = mock_kfp_context.call_args[1]['on_cancel']
        
        cancel_func()

        mock_mlengine_client().cancel_job.assert_called_with(
            'mock_project', 'mock_job'
        )