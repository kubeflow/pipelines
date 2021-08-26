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
    pprint(tasks)
    t.assertEqual(
        {
            'importer':
                KfpTask(name='importer',
                        type='system.ContainerExecution',
                        state=Execution.State.COMPLETE,
                        inputs=TaskInputs(parameters={},
                                          artifacts=[]),
                        outputs=TaskOutputs(parameters={},
                                            artifacts=[
                                                KfpArtifact(name='artifact',
                                                            uri='gs://ml-pipeline-playground/shakespeare1.txt',
                                                            type='system.Dataset',
                                                            metadata={})]))
        },
        tasks,
    )


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_with_importer,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
