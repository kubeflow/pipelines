# Copyright 2023 The Kubeflow Authors
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

import kfp.deprecated as kfp
import kfp_server_api
from ml_metadata.proto import Execution

from kfp.samples.test.utils import KfpTask, TaskInputs, TaskOutputs, TestCase, run_pipeline_func
from .pipeline_with_env import pipeline_with_env


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertEqual(
        {
            'print-env-op':
                KfpTask(
                    name='print-env-op',
                    type='system.ContainerExecution',
                    state=Execution.State.COMPLETE,
                    inputs=TaskInputs(
                        parameters={}, artifacts=[]),
                    outputs=TaskOutputs(
                        parameters={}, artifacts=[])),
            'check-env':
                KfpTask(
                    name='check-env',
                    type='system.ContainerExecution',
                    state=Execution.State.COMPLETE,
                    inputs=TaskInputs(
                        parameters={}, artifacts=[]),
                    outputs=TaskOutputs(
                        parameters={}, artifacts=[])),

        },
        tasks,
    )


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_with_env,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])