# Copyright 2018 Google LLC
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


import mlp


@mlp.pipeline(
  name='Default Value',
  description='A pipeline with parameter and default value.'
)
def immediate_value_pipeline(
    url=mlp.PipelineParam(name='url', value='gs://ml-pipeline/shakespeare1.txt')):

  # "url" is a pipeline parameter, meaning users can provide values when running the
  # pipeline using UI, CLI, or API to override the default value.
  op1 = mlp.ContainerOp(
     name='download',
     image='google/cloud-sdk:216.0.0',
     command=['sh', '-c'],
     arguments=['gsutil cat %s | tee /tmp/results.txt' % url],
     file_outputs={'downloaded': '/tmp/results.txt'})
  op2 = mlp.ContainerOp(
     name='echo',
     image='library/bash:4.4.23',
     command=['sh', '-c'],
     arguments=['echo %s' % op1.output])
