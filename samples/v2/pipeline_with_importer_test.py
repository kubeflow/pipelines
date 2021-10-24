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
    t.assertCountEqual(task_names, ['importer', 'train'], 'task names')
    importer = tasks['importer']
    train = tasks['train']

    pprint('==============')
    pprint(tasks)
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt', tasks['importer'].outputs.artifacts[0].uri,
        'output artifact uri of importer should be "gs://ml-pipeline-playground/shakespeare1.txt"')
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt', tasks['train'].inputs.artifacts[0].uri,
        'input artifact uri of train should be "gs://ml-pipeline-playground/shakespeare1.txt"')
    importer_dict = importer.get_dict()
    train_dict = train.get_dict()
    for artifact in importer_dict.get('outputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        artifact.pop('metadata')
    for artifact in train_dict.get('inputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        artifact.pop('metadata')

    pprint('======= importer task  =======')
    pprint(importer_dict)
    pprint('======= train task =======')
    pprint(train_dict)

    t.assertEqual({
        'name': 'importer',
        'inputs': {
            'artifacts': [],
            'parameters': {}
        },
        'outputs': {
            'artifacts': [{
                'name': 'artifact',
                'type': 'system.Dataset',
            }],
            'parameters': {}
        },
        'type': 'system.ImporterExecution',
        'state': Execution.State.COMPLETE,
    }, importer_dict)

    t.assertEqual({
        'name': 'train',
        'inputs': {
            'artifacts': [{
                'name': 'dataset',
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
            'parameters': {'scalar': '123'}
        },
        'type': 'system.ContainerExecution',
        'state': Execution.State.COMPLETE,
    }, train_dict)



if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_with_importer,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
