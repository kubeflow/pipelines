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
from pprint import pprint
import kfp
import kfp_server_api
from google.cloud import storage

from .tensorboard_minio import my_pipeline
from ...test.util import run_pipeline_func, TestCase, KfpMlmdClient


def verify(
    run: kfp_server_api.ApiRun, mlmd_connection_config, argo_workflow_name: str,
    **kwargs
):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')

    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(argo_workflow_name=argo_workflow_name)
    # uncomment to debug
    # pprint(tasks)
    vis = tasks['create-tensorboard-visualization']
    pprint(vis)
    mlpipeline_ui_metadata = vis.outputs.artifacts[0]
    t.assertEqual(mlpipeline_ui_metadata.name, 'mlpipeline-ui-metadata')

    # download artifact content
    storage_client = storage.Client()
    blob = storage.Blob.from_string(mlpipeline_ui_metadata.uri, storage_client)
    data = blob.download_as_text(storage_client)
    print('=== artifact content begin ===')
    print(data)
    print('=== artifact content end ===')
    # TODO(#5830): fix the JSON encoding issue, then update the following to assert parsed JSON data.
    # https://github.com/kubeflow/pipelines/issues/5830
    t.assertTrue('"type": "tensorboard"' in data)


run_pipeline_func([
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        verify_func=verify,
    ),
    TestCase(
        pipeline_func=my_pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
    ),
])
