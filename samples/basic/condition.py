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
  name='pipeline flip coin',
  description='shows how to use mlp.Condition.'
)
def flipcoin():
  flip = mlp.ContainerOp(
      name='flip',
      image='python:alpine3.6',
      command=['sh', '-c'],
      arguments=['python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                 'else \'tails\'; print(result)" | tee /tmp/output'],
      file_outputs={'output': '/tmp/output'})

  with mlp.Condition(flip.output=='heads'):
    flip_again = flip.clone('flip-again')

    with mlp.Condition(flip_again.output=='tails'):
      print_tail = mlp.ContainerOp(
          name='print-tail', image='alpine:3.6', command=['echo', '"it was tail"'])

  with mlp.Condition(flip.output=='tails'):
      print_tail.clone(name='print-tail1')
