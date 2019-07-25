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
from kfp.gcp import use_gcp_secret


def gcs_read_op(url):
  return dsl.ContainerOp(
      name='Access GCS using auth token',
      image='google/cloud-sdk:latest',
      command=['sh', '-c'],
      arguments=['gsutil ls '+str(url)]
      )


def use_secret_json_op():
  return dsl.ContainerOp(
      name='view auth token location',
      image='google/cloud-sdk:latest',
      command=['sh', '-c'],
      arguments=[
          'gcloud auth activate-service-account --key-file /secret/gcp-credentials/user-gcp-sa.json']
      )


@dsl.pipeline(
    name='Secret pipeline',
    description='A pipeline to demonstrate mounting and use of secretes.'
)


def secret_op_pipeline(url='gs://ml-pipeline-playground/shakespeare1.txt'):
  """A pipeline that requires secret to access cloud hosted resouces."""

  gcs_read_task = gcs_read_op(url).apply(
    use_gcp_secret('user-gcp-sa'))
  use_secret_json_task = use_secret_json_op().apply(
    use_gcp_secret('user-gcp-sa'))


if __name__ == '__main__':
  kfp.compiler.Compiler().compile(secret_op_pipeline, __file__ + '.zip')