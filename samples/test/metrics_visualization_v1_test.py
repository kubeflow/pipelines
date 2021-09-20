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
from .metrics_visualization_v1 import metrics_visualization_v1_pipeline
from .util import KfpMlmdClient, run_pipeline_func, TestCase

import kfp
import kfp_server_api


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

    table_visualization = tasks['table-visualization']
    markdown_visualization = tasks['markdown-visualization']
    roc_visualization = tasks['roc-visualization']
    html_visualization = tasks['html-visualization']
    confusion_visualization = tasks['confusion-visualization']

    t.assertEqual(table_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')
    t.assertEqual(markdown_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')
    t.assertEqual(roc_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')
    t.assertEqual(html_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')
    t.assertEqual(confusion_visualization.get_dict()['outputs']['artifacts'][0]['name'],
                  'mlpipeline_ui_metadata')


run_pipeline_func([TestCase(pipeline_func=metrics_visualization_v1_pipeline,
                            mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
                            )])
