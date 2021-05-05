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

import unittest
from pprint import pprint

import kfp
import kfp_server_api

from .two_step import two_step_pipeline
from .util import run_pipeline_func, TestCase, KfpMlmdClient


def verify(
        run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
        **kwargs
):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff

    t.assertEqual(run.status, 'Succeeded')

    # Verify MLMD state
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)

    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['train-op', 'preprocess'], 'task names')

    preprocess: KfpTask = tasks.get('preprocess')
    train: KfpTask = tasks.get('train-op')

    pprint('======= preprocess task =======')
    pprint(preprocess.get_dict())
    pprint('======= train task =======')
    pprint(train.get_dict())
    pprint('==============')

    t.assertEqual(
        preprocess.get_dict(), {
            'name': 'preprocess',
            'inputs': {
                'artifacts': [],
                'parameters': {
                    'some_int': 12,
                    'uri': 'uri-to-import'
                }
            },
            'outputs': {
                'artifacts': [{'custom_properties': {'name': 'output_dataset_one'},
                               'name': 'output_dataset_one',
                               'type': 'system.Dataset'
                               }],
                'parameters': {
                    'output_parameter_one': 1234
                }
            },
            'type': 'kfp.ContainerExecution'
        }
    )
    t.assertEqual(
        train.get_dict(), {
            'name': 'train-op',
            'inputs': {
                'artifacts': [{'custom_properties': {'name': 'output_dataset_one'},
                               'name': 'output_dataset_one',
                               'type': 'system.Dataset',
                               }],
                'parameters': {
                    'num_steps': 1234
                }
            },
            'outputs': {
                'artifacts': [{'custom_properties': {'name': 'model'},
                               'name': 'model',
                               'type': 'system.Model',
                               }],
                'parameters': {}
            },
            'type': 'kfp.ContainerExecution'
        }
    )


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        ),
        TestCase(
            pipeline_func=two_step_pipeline,
            mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY
        ),
        TestCase(pipeline_func=two_step_pipeline,
                 verify_func=verify,
                 mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
                 arguments={
                     kfp.dsl.ROOT_PARAMETER_NAME: 'minio://mlpipeline/v2/artifacts'},
                 )
    ])

# %%
