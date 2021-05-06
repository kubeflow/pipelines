# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.v2.google.experimental.custom_job."""

import unittest

from kfp.dsl import _container_op
from kfp.v2.google.experimental import run_as_aiplatform_custom_job


class CustomJobTest(unittest.TestCase):

  def test_run_as_aiplatform_custom_job_with_master_settings(self):
    task = _container_op.ContainerOp(
        name='test-task',
        image='python:3.7',
        command=['python3', 'main.py'],
        arguments=['arg1', 'arg2'])
    run_as_aiplatform_custom_job(
        task,
        display_name='custom-job1',
        machine_type='n1-standard-8',
        accelerator_type='NVIDIA_TESLA_K80',
        accelerator_count=2,
        boot_disk_type='pd-ssd',
        boot_disk_size_gb=200,
        timeout='3600s',
        restart_job_on_worker_restart=True,
        service_account='test-sa',
        network='projects/123/global/networks/mypvc',
        output_uri_prefix='gs://bucket/')

    expected_custom_job_spec = {
        'displayName': 'custom-job1',
        'jobSpec': {
            'workerPoolSpecs': [{
                'machineSpec': {
                    'machineType': 'n1-standard-8',
                    'acceleratorType': 'NVIDIA_TESLA_K80',
                    'acceleratorCount': 2
                },
                'replicaCount': '1',
                'containerSpec': {
                    'imageUri': 'python:3.7',
                    'command': ['python3', 'main.py'],
                    'args': ['arg1', 'arg2']
                },
                'diskSpec': {
                    'bootDiskType': 'pd-ssd',
                    'bootDiskSizeGb': 200
                }
            }],
            'scheduling': {
                'timeout': '3600s',
                'restartJobOnWorkerRestart': True
            },
            'serviceAccount': 'test-sa',
            'network': 'projects/123/global/networks/mypvc',
            'baseOutputDirectory': {
                'outputUriPrefix': 'gs://bucket/'
            }
        }
    }
    self.maxDiff = None
    self.assertDictEqual(task.custom_job_spec, expected_custom_job_spec)

  def test_run_as_aiplatform_custom_job_with_additional_worker_pool_specs(self):
    task = _container_op.ContainerOp(
        name='test-task',
        image='python:3.7',
        command=['python3', 'main.py'],
        arguments=['arg1', 'arg2'])
    run_as_aiplatform_custom_job(
        task,
        display_name='custom-job1',
        additional_worker_pool_specs=[
            {
                'containerSpec': {
                    'imageUri': 'alpine',
                    'command': ['sh', '-c', 'echo 1'],
                },
                'replicaCount': '2',
                'machineSpec': {
                    'machineType': 'n1-standard-8',
                },
            },
        ])

    expected_custom_job_spec = {
        'displayName': 'custom-job1',
        'jobSpec': {
            'workerPoolSpecs': [{
                'machineSpec': {
                    'machineType': 'n1-standard-4'
                },
                'replicaCount': '1',
                'containerSpec': {
                    'imageUri': 'python:3.7',
                    'command': ['python3', 'main.py'],
                    'args': ['arg1', 'arg2']
                }
            }, {
                'containerSpec': {
                    'imageUri': 'alpine',
                    'command': ['sh', '-c', 'echo 1']
                },
                'replicaCount': '2',
                'machineSpec': {
                    'machineType': 'n1-standard-8'
                }
            }]
        }
    }

    print(task.custom_job_spec)
    self.maxDiff = None
    self.assertDictEqual(task.custom_job_spec, expected_custom_job_spec)


if __name__ == '__main__':
  unittest.main()
