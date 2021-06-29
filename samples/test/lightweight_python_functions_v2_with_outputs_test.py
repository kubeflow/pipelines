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
import kfp
import kfp_server_api
import os
from minio import Minio

from .lightweight_python_functions_v2_with_outputs import pipeline
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

    output_artifact = tasks['output-artifact']
    output = [
        a for a in output_artifact.outputs.artifacts if a.name == 'Output'
    ][0]
    pprint(output)

    host = os.environ['MINIO_SERVICE_SERVICE_HOST']
    port = os.environ['MINIO_SERVICE_SERVICE_PORT']
    minio = Minio(
        f'{host}:{port}',
        access_key='minio',
        secret_key='minio123',
        secure=False
    )
    bucket, key = output.uri[len('minio://'):].split('/', 1)
    print(f'bucket={bucket} key={key}')
    response = minio.get_object(bucket, key)
    data = response.read().decode('UTF-8')
    t.assertEqual(data, 'firstsecond\nfirstsecond\nfirstsecond')


run_pipeline_func([
    # Verify overriding pipeline root to MinIO
    TestCase(
        pipeline_func=pipeline,
        verify_func=verify,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE,
        arguments={
            kfp.dsl.ROOT_PARAMETER_NAME: 'minio://mlpipeline/override/artifacts'
        },
    ),
    TestCase(
        pipeline_func=pipeline,
        mode=kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
    ),
])
