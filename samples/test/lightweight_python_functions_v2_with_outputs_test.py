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

import os
from pprint import pprint
import unittest

from kfp.samples.test.utils import KfpMlmdClient
from kfp.samples.test.utils import run_pipeline_func
from kfp.samples.test.utils import TestCase
import kfp_server_api
import boto3
from botocore.exceptions import ClientError

from .lightweight_python_functions_v2_with_outputs import pipeline


def verify(run: kfp_server_api.ApiRun, mlmd_connection_config, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff
    t.assertEqual(run.status, 'Succeeded')
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    pprint(tasks)

    output_artifact = tasks['output-artifact']
    output = [
        a for a in output_artifact.outputs.artifacts if a.name == 'Output'
    ][0]
    pprint(output)

    host = os.environ['MINIO_SERVICE_SERVICE_HOST']
    port = os.environ['MINIO_SERVICE_SERVICE_PORT']
    
    # Create boto3 S3 client for SeaweedFS
    s3_client = boto3.client(
        's3',
        aws_access_key_id='minio',
        aws_secret_access_key='minio123',
        endpoint_url=f'http://{host}:{port}',
        region_name='us-east-1'  # Required but not used by SeaweedFS
    )
    
    # Handle both minio:// and s3:// URI schemes
    if output.uri.startswith('minio://'):
        bucket, key = output.uri[len('minio://'):].split('/', 1)
    elif output.uri.startswith('s3://'):
        bucket, key = output.uri[len('s3://'):].split('/', 1) 
    else:
        raise ValueError(f"Unsupported URI scheme in {output.uri}")
        
    print(f'bucket={bucket} key={key}')
    
    try:
        response = s3_client.get_object(Bucket=bucket, Key=key)
        data = response['Body'].read().decode('UTF-8')
        t.assertEqual(data, 'firstsecond\nfirstsecond\nfirstsecond')
    except ClientError as e:
        raise AssertionError(f"Failed to get object from SeaweedFS: {e}")


run_pipeline_func([
    TestCase(pipeline_func=pipeline,),
])
