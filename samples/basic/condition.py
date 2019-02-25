#!/usr/bin/env python3
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


import kfp.dsl as dsl


class RandomNumOp(dsl.ContainerOp):
  """Generate a random number between low and high."""

  def __init__(self, low, high):
    super(RandomNumOp, self).__init__(
      name='Random number',
      image='python:alpine3.6',
      command=['sh', '-c'],
      arguments=['python -c "import random; print(random.randint(%s,%s))" | tee /tmp/output' % (low, high)],
      file_outputs={'output': '/tmp/output'})


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
    

@dsl.pipeline(
  name='pipeline flip coin',
  description='shows how to use dsl.Condition.'
)
def flipcoin():
  flip = FlipCoinOp()
  with dsl.Condition(flip.output == 'heads'):
    random_num_head = RandomNumOp(0, 9)
    with dsl.Condition(random_num_head.output > 5):
      PrintOp('heads and %s > 5!' % random_num_head.output)
    with dsl.Condition(random_num_head.output <= 5):
      PrintOp('heads and %s <= 5!' % random_num_head.output)

  with dsl.Condition(flip.output == 'tails'):
    random_num_tail = RandomNumOp(10, 19)
    with dsl.Condition(random_num_tail.output > 15):
      PrintOp('tails and %s > 15!' % random_num_tail.output)
    with dsl.Condition(random_num_tail.output <= 15):
      PrintOp('tails and %s <= 15!' % random_num_tail.output)


if __name__ == '__main__':
  import kfp.compiler as compiler
  compiler.Compiler().compile(flipcoin, __file__ + '.zip')
