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
from kfp.samples.test.utils import KfpTask, run_pipeline_func, TestCase


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertCountEqual(['importer', 'train'], tasks.keys(), 'task names')
    importer = tasks['importer']
    train = tasks['train']
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt',
        importer.outputs.artifacts[0].uri,
        'output artifact uri of importer should be "gs://ml-pipeline-playground/shakespeare1.txt"'
    )
    t.assertEqual(
        'gs://ml-pipeline-playground/shakespeare1.txt',
        train.inputs.artifacts[0].uri,
        'input artifact uri of train should be "gs://ml-pipeline-playground/shakespeare1.txt"'
    )
    importer_dict = importer.get_dict()
    train_dict = train.get_dict()
    for artifact in importer_dict.get('outputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        if artifact.get('metadata') is not None:
            artifact.pop('metadata')
    for artifact in train_dict.get('inputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        if artifact.get('metadata') is not None:
            artifact.pop('metadata')

    t.assertEqual(
        {
            'name': 'importer',
            'inputs': {},
            'outputs': {
                'artifacts': [{
                    'name': 'artifact',
                    'type': 'system.Artifact',
                    # 'type': 'system.Dataset',
                }],
            },
            'type': 'system.ImporterExecution',
            'state': Execution.State.COMPLETE,
        },
        importer_dict)

    t.assertEqual(
        {
            'name': 'train',
            'inputs': {
                'artifacts': [{
                    'name': 'dataset',
                    # TODO(chesu): compiled pipeline spec incorrectly sets importer artifact type to system.Artifact, but in the pipeline, it should be system.Dataset.
                    'type': 'system.Artifact',
                    # 'type': 'system.Dataset'
                }],
            },
            'outputs': {
                'artifacts': [{
                    'metadata': {
                        'display_name': 'model'
                    },
                    'name': 'model',
                    'type': 'system.Model'
                }],
                'parameters': {
                    'scalar': '123'
                }
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        },
        train_dict)


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_with_importer,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
