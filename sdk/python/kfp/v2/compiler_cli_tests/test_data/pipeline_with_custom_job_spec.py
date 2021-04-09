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
from kfp import dsl
from kfp.v2 import compiler


@components.create_component_from_func
def print_op(text: str):
  print(text)


@dsl.pipeline(name='pipeline-with-custom-job-spec')
def my_pipeline():

  # Normal container execution.
  print_op('container execution')

  # Full custom job spec execution.
  print_op('custom job execution - full custom job').set_custom_job_spec({
      'name': 'test-custom-job-full',
      'jobSpec': {
          'workerPoolSpecs': [{
              'containerSpec': {
                  'command': [
                      'sh', '-c', 'set -e -x\necho "$0"\n',
                      '{{$.inputs.parameters[\'text\']}}'
                  ],
                  'imageUri': 'alpine:latest',
              },
              'replicaCount': '1',
              'machineSpec': {
                  'machineType': 'n1-standard-4'
              }
          }]
      }
  })

  # Custom job spec with 'jobSpec' omitted - jobSpec will be auto-filled using
  # the container spec.
  print_op('custom job execution - partial custom job').set_custom_job_spec({
      'name': 'test-custom-job-partial',
  })


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      pipeline_root='dummy_root',
      package_path=__file__.replace('.py', '.json'))
