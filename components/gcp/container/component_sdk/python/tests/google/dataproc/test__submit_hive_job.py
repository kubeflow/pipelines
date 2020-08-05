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

from kfp_component.google.dataproc import submit_hive_job

MODULE = 'kfp_component.google.dataproc._submit_hive_job'

@mock.patch(MODULE + '.submit_job')
class TestSubmitHiveJob(unittest.TestCase):

    def test_submit_hive_job_with_expected_payload(self, mock_submit_job):
        submit_hive_job('mock-project', 'mock-region', 'mock-cluster', 
            job_id_output_path='/tmp/kfp/output/dataproc/job_id.txt',
            queries=['select * from mock_table'], 
            script_variables={'var-1': 'value1'}, 
            hive_job={ 'continueOnFailure': True },
            job={ 'labels': {'key1': 'value1'}})

        mock_submit_job.assert_called_with('mock-project', 'mock-region', 'mock-cluster',
            {
                'hiveJob': {
                    'queryList': { 'queries': [
                        'select * from mock_table'
                    ]},
                    'scriptVariables': {'var-1': 'value1'},
                    'continueOnFailure': True
                },
                'labels': {
                    'key1': 'value1'
                }
            }, 30, job_id_output_path='/tmp/kfp/output/dataproc/job_id.txt')