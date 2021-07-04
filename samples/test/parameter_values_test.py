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

from pprint import pprint
import unittest
import json
import kfp
import kfp_server_api
from .parameter_values import param_values_pipeline
from .util import KfpMlmdClient, run_pipeline_func, TestCase


def verify(
    run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
    **kwargs
):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    pprint(tasks)
    test = tasks["test"]
    test2 = tasks["test-2"]
    test3 = tasks["test-3"]
    t.assertEqual({"a": "b"}, json.loads(test.inputs.parameters["json_param"]))
    t.assertEqual({"a": "c"}, json.loads(test2.inputs.parameters["json_param"]))
    t.assertEqual("", test3.inputs.parameters["json_param"])


run_pipeline_func([
    TestCase(
        pipeline_func=param_values_pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
    ),
    TestCase(
        pipeline_func=param_values_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])
