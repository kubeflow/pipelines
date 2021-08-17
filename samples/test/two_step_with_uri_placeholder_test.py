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
from .util import run_pipeline_func, TestCase, KfpMlmdClient, KfpTask
from ml_metadata.proto import Execution


def verify_tasks(t: unittest.TestCase, tasks: Dict[str, KfpTask]):
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['read-from-gcs', 'write-to-gcs'], 'task names')

    write_task = tasks['write-to-gcs']
    read_task = tasks['read-from-gcs']

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
                'metadata': {
                    'display_name': 'artifact'
                },
                'name': 'artifact',
                'type': 'system.Artifact'
            }],
            'parameters': {}
        },
        'type': 'system.ContainerExecution',
        'state': Execution.State.COMPLETE,
    }, write_task.get_dict())
    t.assertEqual({
        'name': 'read-from-gcs',
        'inputs': {
            'artifacts': [{
                'metadata': {
                    'display_name': 'artifact'
                },
                'name': 'artifact',
                'type': 'system.Artifact',
            }],
            'parameters': {}
        },
        'outputs': {
            'artifacts': [],
            'parameters': {}
        },
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
            mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        ),
        # TODO(v2): support dynamic pipeline root, error log:
        # gsutil cp - minio://mlpipeline/v2/artifacts/two-step-with-uri-placeholders/f4366dd7-2bfa-45f6-8b19-586ed061bbaf/write-to-gcs/artifact
        # InvalidUrlError: Unrecognized scheme "minio".
        # F0816 07:49:41.552936       1 main.go:41] failed to execute component: exit status 1
        # TestCase(
        #     pipeline_func=two_step_with_uri_placeholder,
        #     verify_func=verify,
        #     mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        # ),
    ])
