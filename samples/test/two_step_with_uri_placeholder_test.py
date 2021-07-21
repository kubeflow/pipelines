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

import kfp
import kfp_server_api

from .two_step_with_uri_placeholder import two_step_pipeline
from .util import run_pipeline_func, TestCase, KfpMlmdClient, KfpTask


def get_tasks(mlmd_connection_config, argo_workflow_name: str):
    # Verify MLMD state
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    return tasks


def verify_tasks(t: unittest.TestCase, tasks: dict[str, KfpTask]):
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['read-from-gcs', 'write-to-gcs'], 'task names')

    write_task = tasks['wirte-to-gcs']
    read_task= tasks['read-from-gcs']

    pprint('======= preprocess task =======')
    pprint(write_task.get_dict())
    pprint('======= train task =======')
    pprint(read_task.get_dict())
    pprint('==============')

    t.assertEqual({
        'name': 'write-to-gcs',
        'inputs': {
            'artifacts': [],
            'parameters': {
                'msg': 'Hello world!',
            }
        },
        'outputs': {
            'artifacts': [{
                'metadata': {},
                'name': 'output_gcs_path',
                'type': 'system.Artifact'
            }],
            'parameters': {}
        },
        'type': 'kfp.ContainerExecution'
    }, write_task.get_dict())
    t.assertEqual({
        'name': 'read-from-gcs',
        'inputs': {
            'artifacts': [{
                'metadata': {},
                'name': 'input_gcs_path',
                'type': 'system.Artifact',
            }],
            'parameters': {}
        },
        'outputs': {
            'artifacts': [],
            'parameters': {}
        },
        'type': 'kfp.ContainerExecution'
    }, read_task.get_dict())


def verify(
    run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
    **kwargs
):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    tasks = get_tasks(mlmd_connection_config, argo_workflow_name)
    verify_tasks(t, tasks)

if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        ),
    ])
