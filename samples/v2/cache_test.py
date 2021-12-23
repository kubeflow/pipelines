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
"""Two step v2 pipeline."""

# %%

from __future__ import annotations

import random
import string
import unittest
import functools

import kfp
import kfp_server_api

from ..test.two_step import two_step_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient, KfpTask
from ml_metadata.proto import Execution


def verify_tasks(t: unittest.TestCase, tasks: dict[str, KfpTask], task_state,
                 uri: str, some_int: int):
    task_names = [*tasks.keys()]
    t.assertCountEqual(task_names, ['train-op', 'preprocess'], 'task names')

    preprocess = tasks['preprocess']
    train = tasks['train-op']

    t.assertEqual(
        {
            'name': 'preprocess',
            'inputs': {
                'artifacts': [],
                'parameters': {
                    'some_int': some_int,
                    'uri': uri
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
                    'output_parameter_one': some_int
                }
            },
            'type': 'system.ContainerExecution',
            'state': task_state,
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
                    'num_steps': some_int
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
            'state': task_state,
        }, train.get_dict())


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, uri: str,
           some_int, state: int, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    verify_tasks(t, tasks, state, uri, some_int)


if __name__ == '__main__':
    letters = string.ascii_lowercase
    random_uri = 'http://v2' + ''.join(random.choice(letters) for i in range(5))
    random_int = random.randint(0, 10000)
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline,
            arguments={
                'uri': f'{random_uri}',
                'some_int': f'{random_int}'
            },
            verify_func=functools.partial(
                verify,
                uri=random_uri,
                some_int=random_int,
                state=Execution.State.COMPLETE,
            ),
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
            enable_caching=True),
    ]),
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline,
            arguments={
                'uri': f'{random_uri}',
                'some_int': f'{random_int}'
            },
            verify_func=functools.partial(
                verify,
                uri=random_uri,
                some_int=random_int,
                state=Execution.State.CACHED),
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
            enable_caching=True),
    ])

# %%
