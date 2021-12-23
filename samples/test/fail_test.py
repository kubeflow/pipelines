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
"""Fail pipeline."""

from __future__ import annotations
import unittest
import kfp
import kfp_server_api
from ml_metadata.proto import Execution
from .fail import fail_pipeline
from .fail_v2 import fail_pipeline as fail_v2_pipeline
from kfp.samples.test.utils import TaskInputs, TaskOutputs, run_pipeline_func, TestCase, KfpTask


def verify(run, **kwargs):
    assert run.status == 'Failed'


def verify_v2(t: unittest.TestCase, run: kfp_server_api.ApiRun,
              tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Failed')
    t.assertEqual(
        {
            'fail':
                KfpTask(
                    name='fail',
                    type='system.ContainerExecution',
                    # TODO(Bobgy): fix v2 engine to properly publish FAILED state.
                    state=Execution.State.RUNNING,
                    inputs=TaskInputs(parameters={}, artifacts=[]),
                    outputs=TaskOutputs(parameters={}, artifacts=[]),
                )
        },
        tasks,
    )


run_pipeline_func([
    TestCase(
        pipeline_func=fail_v2_pipeline,
        verify_func=verify_v2,
        mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE),
    TestCase(
        pipeline_func=fail_pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY),
])
