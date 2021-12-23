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

import kfp
import kfp_server_api
from ml_metadata.proto import Execution

from kfp.samples.test.utils import KfpTask, TaskInputs, TaskOutputs, TestCase, run_pipeline_func
from .hello_world import pipeline_hello_world


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertEqual(
        {
            'hello-world':
                KfpTask(
                    name='hello-world',
                    type='system.ContainerExecution',
                    state=Execution.State.COMPLETE,
                    inputs=TaskInputs(
                        parameters={'text': 'hi there'}, artifacts=[]),
                    outputs=TaskOutputs(
                        parameters={'Output': 'hi there'}, artifacts=[]))
        },
        tasks,
    )


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_hello_world,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
