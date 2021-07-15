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

import kfp_server_api
import unittest
from pprint import pprint
from .confusion_matrix import confusion_matrix_pipeline, confusion_visualization
from ...test.util import KfpMlmdClient, run_pipeline_func, TestCase

import kfp


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

    confusion_visualization = tasks['confusion-visualization']
    output = [
        a for a in confusion_visualization.outputs.artifacts if a.name == 'mlpipeline_ui_metadata'
    ][0]
    pprint(output)

    t.assertEqual(confusion_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')


run_pipeline_func([TestCase(pipeline_func=confusion_matrix_pipeline,
                            verify_func=verify,
                            mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
                            arguments={
                            }),
                   TestCase(pipeline_func=confusion_matrix_pipeline,
                            mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY
                            )])
