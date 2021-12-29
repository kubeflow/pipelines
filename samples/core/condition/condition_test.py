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
from .condition import condition
from .condition_v2 import condition as condition_v2
from kfp.samples.test.utils import KfpTask, debug_verify, run_pipeline_func, TestCase


def verify_heads(t: unittest.TestCase, run: kfp_server_api.ApiRun,
                 tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertCountEqual(['print-msg', 'condition-1', 'flip-coin'], tasks.keys())
    t.assertCountEqual(['print-msg-2', 'print-msg-3', 'flip-coin-2'],
                       tasks['condition-1'].children.keys())


def verify_tails(t: unittest.TestCase, run: kfp_server_api.ApiRun,
                 tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertCountEqual(['print-msg', 'condition-1', 'flip-coin'], tasks.keys())
    t.assertIsNone(tasks['condition-1'].children)
    # MLMD canceled state means NotTriggered state for KFP.
    t.assertEqual(Execution.State.CANCELED, tasks['condition-1'].state)


run_pipeline_func([
    TestCase(
        pipeline_func=condition_v2,
        mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        arguments={"force_flip_result": "heads"},
        verify_func=verify_heads,
    ),
    TestCase(
        pipeline_func=condition_v2,
        mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        arguments={"force_flip_result": "tails"},
        verify_func=verify_tails,
    ),
    TestCase(
        pipeline_func=condition,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
        arguments={"force_flip_result": "heads"},
    ),
    TestCase(
        pipeline_func=condition,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
        arguments={"force_flip_result": "tails"},
    ),
])
