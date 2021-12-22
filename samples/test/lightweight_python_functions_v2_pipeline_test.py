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

import unittest
from pprint import pprint
import kfp_server_api
import kfp.dsl as dsl

from .lightweight_python_functions_v2_pipeline import pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient
from ml_metadata.proto import Execution


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)

    task_names = [*tasks.keys()]
    t.assertCountEqual(task_names, ['preprocess', 'train'], 'task names')
    pprint(tasks)

    preprocess = tasks['preprocess']
    train = tasks['train']
    pprint(preprocess.get_dict())
    t.assertEqual(
        {
            'inputs': {
                'parameters': {
                    'message': 'message',
                }
            },
            'name': 'preprocess',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'output_dataset_one'
                    },
                    'name': 'output_dataset_one',
                    'type': 'system.Dataset'
                }, {
                    'metadata': {
                        'display_name': 'output_dataset_two_path'
                    },
                    'name': 'output_dataset_two_path',
                    'type': 'system.Dataset'
                }],
                'parameters': {
                    'output_bool_parameter_path': True,
                    'output_dict_parameter_path': {
                        "A": 1,
                        "B": 2
                    },
                    'output_list_parameter_path': ["a", "b", "c"],
                    'output_parameter_path': 'message'
                }
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        },
        preprocess.get_dict(),
    )
    t.assertEqual(
        {
            'inputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'output_dataset_one'
                    },
                    'name': 'dataset_one_path',
                    'type': 'system.Dataset'
                }, {
                    'metadata': {
                        'display_name': 'output_dataset_two_path'
                    },
                    'name': 'dataset_two',
                    'type': 'system.Dataset'
                }],
                'parameters': {
                    'input_bool': True,
                    'input_dict': {
                        "A": 1.0,
                        "B": 2.0,
                    },
                    'input_list': ["a", "b", "c"],
                    'message': 'message'
                }
            },
            'name': 'train',
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'model',
                        'accuracy': 0.9,
                    },
                    'name': 'model',
                    'type': 'system.Model'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        },
        train.get_dict(),
    )


run_pipeline_func([
    TestCase(
        pipeline_func=pipeline,
        verify_func=verify,
        mode=dsl.PipelineExecutionMode.V2_ENGINE),
])
