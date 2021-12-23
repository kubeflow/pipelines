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

from __future__ import annotations

import unittest
import kfp
import kfp_server_api
from ml_metadata.proto import Execution
from .loop_output import my_pipeline
from .loop_output_v2 import my_pipeline as my_pipeline_v2
from kfp.samples.test.utils import KfpTask, run_pipeline_func, TestCase


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    # assert DAG structure
    t.assertCountEqual(tasks.keys(), ['args-generator-op', 'for-loop-1'])
    t.assertCountEqual(
        ['for-loop-1-#0', 'for-loop-1-#1', 'for-loop-1-#2'],
        tasks['for-loop-1'].children.keys(),
    )
    # assert all iteration parameters
    t.assertCountEqual(
        ['1.1', '1.2', '1.3'],
        [
            x.inputs
            .parameters['pipelinechannel--args-generator-op-Output-loop-item']
            for x in tasks['for-loop-1'].children.values()
        ],
    )
    # assert 1 iteration task
    t.assertEqual(
        {
            'name': 'for-loop-1-#0',
            'type': 'system.DAGExecution',
            'state':
                Execution.State.RUNNING,  # TODO(Bobgy): this should be COMPLETE
            'inputs': {
                'parameters': {
                    'pipelinechannel--args-generator-op-Output-loop-item': '1.1'
                }
            },
            'outputs': {},
        },
        tasks['for-loop-1'].children['for-loop-1-#0'].get_dict(),
    )
    t.assertEqual(
        {
            'name': 'print-op',
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
            'inputs': {
                'parameters': {
                    's': '1.1',
                }
            },
            'outputs': {},
        },
        tasks['for-loop-1'].children['for-loop-1-#0'].children['print-op']
        .get_dict(),
    )


run_pipeline_func([
    TestCase(
        pipeline_func=my_pipeline_v2,
        mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        verify_func=verify,
    ),
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])
