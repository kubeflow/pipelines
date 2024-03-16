#!/usr/bin/env python3
# Copyright 2019 The Kubeflow Authors
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


from kfp import dsl, compiler
from kfp.components import load_component_from_text


# Accessing GCS using the Google Cloud SDK command-line programs
gcs_list_items_op = load_component_from_text(text='''
name: GCS - List items
inputs:
- {name: url, type: STRING}
implementation:
  container:
    image: 'google/cloud-sdk:279.0.0'
    command:
    - sh
    - -exc
    - |
      if [ -n "${GOOGLE_APPLICATION_CREDENTIALS}" ]; then
          echo "Using auth token from ${GOOGLE_APPLICATION_CREDENTIALS}"
          gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
      fi
      gcloud auth list
      gsutil ls "$0"
    - {inputValue: url}
''')


# Accessing GCS using the Google Cloud Python library
@dsl.component(
    base_image='python:3.7', 
    packages_to_install=['google-cloud-storage==1.31.2']
)
def gcs_list_buckets():
    from google.cloud import storage
    storage_client = storage.Client()
    buckets = storage_client.list_buckets()
    print("List of buckets:")
    for bucket in buckets:
        print(bucket.name)


@dsl.pipeline(
    name='secret-pipeline',
    description='A pipeline to demonstrate mounting and use of secretes.'
)
def secret_op_pipeline(
    url:str='gs://ml-pipeline/sample-data/shakespeare/shakespeare1.txt'):
  """A pipeline that uses secret to access cloud hosted resouces."""

  gcs_list_items_task = gcs_list_items_op(url=url)
  gcs_list_buckets_task = gcs_list_buckets()

if __name__ == '__main__':
  compiler.Compiler().compile(secret_op_pipeline, __file__ + '.yaml')
