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

import unittest

import kfp
from .reused_component import my_pipeline
from .util import run_pipeline_func, TestCase, KfpMlmdClient


def verify(run, argo_workflow_name: str, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')

    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)

    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    t.assertEqual(
        tasks.get('add-3').outputs.parameters.get('sum'),
        17, 'add result should be 17')


run_pipeline_func([
    TestCase(pipeline_func=my_pipeline, verify_func=verify),
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])
