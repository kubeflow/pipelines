# Copyright 2018 Google LLC
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

from kfp_component.google.dataproc import submit_pyspark_job

MODULE = 'kfp_component.google.dataproc._submit_pyspark_job'

@mock.patch(MODULE + '.submit_job')
class TestSubmitPySparkJob(unittest.TestCase):

    def test_submit_pyspark_job_with_expected_payload(self, mock_submit_job):
        submit_pyspark_job('mock-project', 'mock-region', 'mock-cluster', 
            job_id_output_path='/tmp/kfp/output/dataproc/job_id.txt',
            main_python_file_uri='gs://mock/python/file.py', args=['arg1', 'arg2'], 
            pyspark_job={ 'pythonFileUris': ['gs://other/python/file.py'] },
            job={ 'labels': {'key1': 'value1'}})

        mock_submit_job.assert_called_with('mock-project', 'mock-region', 'mock-cluster',
            {
                'pysparkJob': {
                    'mainPythonFileUri': 'gs://mock/python/file.py',
                    'args': ['arg1', 'arg2'],
                    'pythonFileUris': ['gs://other/python/file.py']
                },
                'labels': {
                    'key1': 'value1'
                }
            }, 30, job_id_output_path='/tmp/kfp/output/dataproc/job_id.txt')