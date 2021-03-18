# Copyright 2021 Google LLC
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
import os

from kfp_component.google.dataflow import launch_python

MODULE = 'kfp_component.google.dataflow._launch_python'

@mock.patch(MODULE + '.storage')
@mock.patch('kfp_component.google.dataflow._common_ops.display')
@mock.patch(MODULE + '.stage_file')
@mock.patch(MODULE + '.KfpExecutionContext')
@mock.patch(MODULE + '.DataflowClient')
@mock.patch(MODULE + '.Process')
@mock.patch(MODULE + '.subprocess')
class LaunchPythonTest(unittest.TestCase):

    def test_launch_python_succeed(self, mock_subprocess, mock_process,
        mock_client, mock_context, mock_stage_file, mock_display, mock_storage):
        mock_context().__enter__().context_id.return_value = 'ctx-1'
        mock_storage.Client().bucket().blob().exists.return_value = False
        mock_process().read_lines.return_value = [
            b'https://console.cloud.google.com/dataflow/jobs/us-central1/job-1?project=project-1'
        ]
        expected_job = {
            'id': 'job-1',
            'currentState': 'JOB_STATE_DONE'
        }
        mock_client().get_job.return_value = expected_job

        result = launch_python('/tmp/test.py', 'project-1', 'us-central1', staging_dir='gs://staging/dir')

        self.assertEqual(expected_job, result)
        mock_storage.Client().bucket().blob().upload_from_string.assert_called_with(
            'job-1,us-central1'
        )

    def test_launch_python_retry_succeed(self, mock_subprocess, mock_process,
        mock_client, mock_context, mock_stage_file, mock_display, mock_storage):
        mock_context().__enter__().context_id.return_value = 'ctx-1'
        mock_storage.Client().bucket().blob().exists.return_value = True
        mock_storage.Client().bucket().blob().download_as_string.return_value = b'job-1,us-central1'
        expected_job = {
            'id': 'job-1',
            'currentState': 'JOB_STATE_DONE'
        }
        mock_client().get_job.return_value = expected_job

        result = launch_python('/tmp/test.py', 'project-1', 'us-central1', staging_dir='gs://staging/dir')

        self.assertEqual(expected_job, result)
        mock_process.assert_not_called()

    def test_launch_python_no_job_created(self, mock_subprocess, mock_process,
        mock_client, mock_context, mock_stage_file, mock_display, mock_storage):
        mock_context().__enter__().context_id.return_value = 'ctx-1'
        mock_process().read_lines.return_value = [
            b'no job id',
            b'no job id'
        ]

        result = launch_python('/tmp/test.py', 'project-1', 'us-central1')

        self.assertEqual(None, result)
