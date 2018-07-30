# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import mlp


@mlp.pipeline(
  name='Exit Handler',
  description='Download a message and print it out. Exit Handler will run at the end.'
)
def download_and_print(url: mlp.PipelineParam):
  """A very simple two-step pipeline."""

  exit_op = mlp.ContainerOp(
      name='finally',
      image='library/bash',
      command=['echo', 'exit!'])

  with mlp.ExitHandler(exit_op):

    op1 = mlp.ContainerOp(
       name='download',
       image='google/cloud-sdk',
       command=['sh', '-c'],
       arguments=['gsutil cat %s | tee /tmp/results.txt' % url],
       argument_inputs=[url],
       file_outputs={'downloaded': '/tmp/results.txt'})

    op2 = mlp.ContainerOp(
       name='echo',
       image='library/bash',
       command=['sh', '-c'],
       arguments=['echo %s' % op1.output],
       argument_inputs=[op1.output])
