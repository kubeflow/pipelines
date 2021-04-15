# Copyright 2021 Google LLC
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

import sys
import logging
import unittest
from dataclasses import dataclass
from typing import Tuple

import kfp
import kfp_server_api

from .two_step import two_step_pipeline
from .util import run_pipeline_func, TestCase, KfpMlmdClient
# sys.path.append('.')
# from two_step import two_step_pipeline
# from util import run_pipeline_func, TestCase, KfpMlmdClient

from ml_metadata import metadata_store
from ml_metadata.proto import metadata_store_pb2

# %%
# argo_workflow_name = 'two-step-pipeline-8ddd7'
# mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
#     host='localhost',
#     port=8080,
# )


def verify(
    run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
    **kwargs
):
    t = unittest.TestCase()
    t.assertEqual(run.status, 'Succeeded')

    # Verify MLMD state
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)

    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['train-op', 'preprocess'], 'task names')

    preprocess: KfpTask = tasks.get('preprocess')
    train: KfpTask = tasks.get('train-op')
    t.assertEqual(
        train.inputs.parameters,
        {
            'num_steps': 1234,
        },
        'train task input parameters',
    )
    t.assertEqual(
        train.outputs.parameters,
        {},
        'train task output parameters',
    )
    t.assertEqual(
        preprocess.inputs.parameters, {
            'uri': 'uri-to-import',
            'some_int': 12,
        }, 'preprocess task input parameters'
    )
    t.assertEqual(
        preprocess.outputs.parameters, {
            'output_parameter_one': 1234,
        }, 'preprocess task output parameters'
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
        )
    ])

# %%
