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
"""Two step v2-compatible pipeline."""

# %%

from __future__ import annotations
import unittest
from pprint import pprint

import kfp
import kfp_server_api

from .two_step import two_step_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient, KfpTask
from ml_metadata.proto import Execution


def verify_tasks(t: unittest.TestCase, tasks: dict[str, KfpTask]):
    task_names = [*tasks.keys()]
    t.assertCountEqual(task_names, ['train-op', 'preprocess'], 'task names')

    preprocess = tasks['preprocess']
    train = tasks['train-op']

    pprint('======= preprocess task =======')
    pprint(preprocess.get_dict())
    pprint('======= train task =======')
    pprint(train.get_dict())
    pprint('==============')

    t.assertEqual(
        {
            'name': 'preprocess',
            'inputs': {
                'artifacts': [],
                'parameters': {
                    'some_int': 1234,
                    'uri': 'uri-to-import'
                }
            },
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'output_dataset_one',
                    },
                    'name': 'output_dataset_one',
                    'type': 'system.Dataset'
                }],
                'parameters': {
                    'output_parameter_one': 1234
                }
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, preprocess.get_dict())
    t.assertEqual(
        {
            'name': 'train-op',
            'inputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'output_dataset_one',
                    },
                    'name': 'dataset',
                    'type': 'system.Dataset',
                }],
                'parameters': {
                    'num_steps': 1234
                }
            },
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'model',
                    },
                    'name': 'model',
                    'type': 'system.Model',
                }],
                'parameters': {}
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, train.get_dict())


def verify_artifacts(t: unittest.TestCase, tasks: dict, artifact_uri_prefix):
    for task in tasks.values():
        for artifact in task.outputs.artifacts:
            t.assertTrue(artifact.uri.startswith(artifact_uri_prefix))


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    verify_tasks(t, tasks)


def verify_with_default_pipeline_root(run: kfp_server_api.ApiRun,
                                      mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    verify_tasks(t, tasks)
    verify_artifacts(t, tasks, 'minio://mlpipeline/v2/artifacts')


def verify_with_specific_pipeline_root(run: kfp_server_api.ApiRun,
                                       mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    verify_tasks(t, tasks)
    verify_artifacts(t, tasks, 'minio://mlpipeline/override/artifacts')


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline,
            mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY),
        # Cannot test V2_ENGINE and V1_LEGACY using the same code.
        # V2_ENGINE requires importing everything from v2 namespace.
        # TestCase(
        #     pipeline_func=two_step_pipeline,
        #     verify_func=verify,
        #     mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        # ),
    ])

# %%
