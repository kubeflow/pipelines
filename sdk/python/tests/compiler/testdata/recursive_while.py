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

import kfp.dsl as dsl

class FlipCoinOp(dsl.ContainerOp):
  """Flip a coin and output heads or tails randomly."""

  def __init__(self):
    super(FlipCoinOp, self).__init__(
        name='Flip',
        image='python:alpine3.6',
        command=['sh', '-c'],
        arguments=['python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                   'else \'tails\'; print(result)" | tee /tmp/output'],
        file_outputs={'output': '/tmp/output'})

class PrintOp(dsl.ContainerOp):
  """Print a message."""

  def __init__(self, msg):
    super(PrintOp, self).__init__(
        name='Print',
        image='alpine:3.6',
        command=['echo', msg],
    )

@dsl._component.graph_component
def flip_component(flip_result, maxVal):
  with dsl.Condition(flip_result == 'heads'):
    print_flip = PrintOp(flip_result)
    flipA = FlipCoinOp().after(print_flip)
    flip_component(flipA.output, maxVal)

@dsl.pipeline(
    name='pipeline flip coin',
    description='shows how to use dsl.Condition.'
)
def flipcoin(maxVal=12):
  flipA = FlipCoinOp()
  flipB = FlipCoinOp()
  flip_loop = flip_component(flipA.output, maxVal)
  flip_loop.after(flipB)
  PrintOp('cool, it is over. %s' % flipA.output).after(flip_loop)

if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(flipcoin, __file__ + '.tar.gz')
