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
"""Hello world v2 engine pipeline."""

from __future__ import annotations

import unittest
from pprint import pprint

import kfp
import kfp_server_api
from ml_metadata.proto import Execution

from .pipeline_with_importer import pipeline_with_importer
from ..test.util import KfpTask, KfpArtifact, TaskInputs, TaskOutputs, run_pipeline_func, TestCase, KfpMlmdClient


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['importer', 'train'], 'task names')
    importer = tasks['importer']
    train = tasks['train']

    pprint('======= importer task with artifact uri =======')
    pprint(importer.get_dict())
    pprint('======= train task =======')
    pprint(train.get_dict())
    pprint('==============')
    pprint(tasks)
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt', tasks['importer'].outputs.artifacts['artifact']['uri'],
        'output artifact uri of importer should be "gs://ml-pipeline-playground/shakespeare1.txt"')
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt', tasks['train'].inputs.artifacts['artifact']['uri'],
        'input artifact uri of train should be "gs://ml-pipeline-playground/shakespeare1.txt"')
    t.assertEqual({
        'name': 'importer',
        'inputs': {
            'artifacts': [],
            'parameters': {}
        },
        'outputs': {
            'artifacts': [{
                'metadata': {},
                'name': 'artifact',
                'type': 'system.Dataset',
            }],
            'parameters': {}
        },
        'type': 'system.ContainerExecution',
        'state': Execution.State.COMPLETE,
    }, importer.get_dict())

    t.assertEqual({
        'name': 'train',
        'inputs': {
            'artifacts': [{
                'metadata': {},
                'name': 'artifact',
                'type': 'system.Dataset'
            }],
            'parameters': {}
        },
        'outputs': {
            'artifacts': [{
                'metadata': {'display_name': 'model'},
                'name': 'model',
                'type': 'system.Model'
            }],
            'parameters': {{'scalar': '123'}}
        },
        'type': 'system.ContainerExecution',
        'state': Execution.State.COMPLETE,
    }, train.get_dict())



if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_with_importer,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
