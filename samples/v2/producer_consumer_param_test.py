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

from .producer_consumer_param import producer_consumer_param_pipeline
from kfp.samples.test.utils import KfpTask, TaskInputs, TaskOutputs, run_pipeline_func, TestCase, KfpMlmdClient
from ml_metadata.proto import Execution


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    t.assertEqual(
        {
            'consumer':
                KfpTask(
                    name='consumer',
                    type='system.ContainerExecution',
                    state=Execution.State.COMPLETE,
                    inputs=TaskInputs(
                        parameters={
                            'input_value':
                                'Hello world, this is an output parameter\n'
                        },
                        artifacts=[]),
                    outputs=TaskOutputs(parameters={}, artifacts=[])),
            'producer':
                KfpTask(
                    name='producer',
                    type='system.ContainerExecution',
                    state=Execution.State.COMPLETE,
                    inputs=TaskInputs(
                        parameters={'input_text': 'Hello world'}, artifacts=[]),
                    outputs=TaskOutputs(
                        parameters={
                            'output_value':
                                'Hello world, this is an output parameter\n'
                        },
                        artifacts=[]))
        }, tasks)


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=producer_consumer_param_pipeline,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
