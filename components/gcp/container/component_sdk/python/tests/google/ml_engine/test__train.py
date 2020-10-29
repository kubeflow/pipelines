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

from kfp_component.google.ml_engine import train

CREATE_JOB_MODULE = 'kfp_component.google.ml_engine._train'


@mock.patch(CREATE_JOB_MODULE + '.create_job')
class TestCreateTrainingJob(unittest.TestCase):

    def test_train_succeed(self, mock_create_job):
        train(
            project_id='proj-1',
            python_module='mock.module',
            package_uris=['gs://test/package'],
            region='region-1',
            args=['arg-1', 'arg-2'],
            job_dir='gs://test/job/dir',
            training_input={
                'runtimeVersion': '1.10',
                'pythonVersion': '2.7'
            },
            job_id_prefix='job-',
            job_id='job-1',
            master_image_uri='tensorflow:latest',
            worker_image_uri='debian:latest',
            job_id_output_path='/tmp/kfp/output/ml_engine/job_id.txt',
            job_dir_output_path='/tmp/kfp/output/ml_engine/job_dir.txt')

        mock_create_job.assert_called_with(
            project_id='proj-1',
            job={
                'trainingInput': {
                    'pythonModule': 'mock.module',
                    'packageUris': ['gs://test/package'],
                    'region': 'region-1',
                    'args': ['arg-1', 'arg-2'],
                    'jobDir': 'gs://test/job/dir',
                    'runtimeVersion': '1.10',
                    'pythonVersion': '2.7',
                    'masterConfig': {
                        'imageUri': 'tensorflow:latest'
                    },
                    'workerConfig': {
                        'imageUri': 'debian:latest'
                    }
                }
            },
            job_id_prefix='job-',
            job_id='job-1',
            wait_interval=30,
            job_id_output_path='/tmp/kfp/output/ml_engine/job_id.txt',
            job_dir_output_path='/tmp/kfp/output/ml_engine/job_dir.txt')
