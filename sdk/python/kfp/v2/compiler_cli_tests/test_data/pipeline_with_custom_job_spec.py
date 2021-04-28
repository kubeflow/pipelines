# Copyright 2021 Google LLC
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

from kfp import components
from kfp.v2 import dsl
from kfp.v2 import compiler
from kfp.v2.dsl import Artifact, Output

MACHINE_TYPE = 'n1-standard-4'

@dsl.ai_platform_custom_job
def hello_world_op(input1: str, output1: Output[Artifact]):
  return {
      'workerPoolSpecs': [{
          'containerSpec': {
              'command': [
                  'sh',
                  '-exc',
                  'echo "$0" | gsutil cp - "$1"',
                  input1,
                  output1
              ],
              'imageUri': 'google/cloud-sdk:slim',
          },
          'replicaCount': '1',
          'machineSpec': {
              'machineType': MACHINE_TYPE
          }
      }]
  }


@dsl.pipeline(name='pipeline-with-custom-job-spec', pipeline_root='dummy_root')
def my_pipeline(text: str = 'hello world'):
  hello_world_op(input1=text)


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline, package_path=__file__.replace('.py', '.json'))
