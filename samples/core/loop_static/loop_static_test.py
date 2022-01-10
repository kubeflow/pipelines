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
from .loop_static import my_pipeline
from .loop_static_v2 import my_pipeline as my_pipeline_v2
from kfp.samples.test.utils import KfpTask, debug_verify, run_pipeline_func, TestCase


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    # assert DAG structure
    t.assertCountEqual(['print-op', 'for-loop-1'], tasks.keys())
    # assert all iteration parameters
    t.assertCountEqual(
        [{
            'a': '1',
            'b': '2'
        }, {
            'a': '10',
            'b': '20'
        }],
        [
            x.inputs
            .parameters['pipelinechannel--static_loop_arguments-loop-item']
            for x in tasks['for-loop-1'].children.values()
        ],
    )
    # assert all iteration outputs
    t.assertCountEqual(['12', '1020'], [
        x.children['print-op-2'].outputs.parameters['Output']
        for x in tasks['for-loop-1'].children.values()
    ])


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
