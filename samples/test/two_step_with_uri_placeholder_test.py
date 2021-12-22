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
"""Two step v2-compatible pipeline with URI placeholders."""

import unittest
from pprint import pprint
from typing import Dict

import kfp
import kfp_server_api

from .two_step_with_uri_placeholder import two_step_with_uri_placeholder
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient, KfpTask
from ml_metadata.proto import Execution


def verify_tasks(t: unittest.TestCase, tasks: Dict[str, KfpTask]):
    t.assertCountEqual(tasks.keys(), ['read-from-gcs', 'write-to-gcs'],
                       'task names')

    write_task = tasks['write-to-gcs']
    read_task = tasks['read-from-gcs']

    t.assertEqual(
        {
            'name': 'write-to-gcs',
            'inputs': {
                'parameters': {
                    'msg': 'Hello world!',
                }
            },
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'artifact'
                    },
                    'name': 'artifact',
                    'type': 'system.Artifact'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, write_task.get_dict())
    t.assertEqual(
        {
            'name': 'read-from-gcs',
            'inputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'artifact'
                    },
                    'name': 'artifact',
                    'type': 'system.Artifact',
                }],
            },
            'outputs': {},
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, read_task.get_dict())


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    verify_tasks(t, tasks)


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_with_uri_placeholder,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
