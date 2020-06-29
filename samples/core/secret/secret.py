#!/usr/bin/env python3
# Copyright 2019 Google LLC
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


import kfp
from kfp import dsl


def gcs_read_op(url):
  return dsl.ContainerOp(
      name='Access GCS using auth token',
      image='google/cloud-sdk:279.0.0',
      command=['sh', '-c'],
      arguments=[
        'gsutil ls "$0" && echo "$1"',
        url,
        'Auth token is located at /secret/gcp-credentials/user-gcp-sa.json'
        ]
      )


def use_gcp_api_op():
    return dsl.ContainerOp(
        name='Using Google Cloud APIs with Auth',
        image='google/cloud-sdk:279.0.0',
        command=[
            'sh', '-c',
            'pip3 install google-cloud-storage && "$0" "$*"',
            'python3', '-c', '''
from google.cloud import storage
storage_client = storage.Client()
buckets = storage_client.list_buckets()
print("List of buckets:")
for bucket in buckets:
    print(bucket.name)
    '''
        ])


@dsl.pipeline(
    name='Secret pipeline',
    description='A pipeline to demonstrate mounting and use of secretes.'
)
def secret_op_pipeline(
    url='gs://ml-pipeline/sample-data/shakespeare/shakespeare1.txt'):
  """A pipeline that uses secret to access cloud hosted resouces."""

  gcs_read_task = gcs_read_op(url)
  use_gcp_api_task = use_gcp_api_op()

if __name__ == '__main__':
  kfp.compiler.Compiler().compile(secret_op_pipeline, __file__ + '.yaml')
