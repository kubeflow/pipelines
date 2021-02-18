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


def flip_coin() -> str:
  """Flip a coin and output heads or tails randomly."""
  import random
  result = 'heads' if random.randint(0, 1) == 0 else 'tails'
  return result


def print_msg(msg: str):
  """Print a message."""
  print(msg)


flip_coin_op = components.create_component_from_func(flip_coin)

print_op = components.create_component_from_func(print_msg)


@dsl.pipeline(name='single-condition-pipeline')
def my_pipeline():
  flip1 = flip_coin_op()
  print_op(flip1.output)

  with dsl.Condition(flip1.output == 'heads'):
    flip2 = flip_coin_op()
    print_op(flip2.output)

if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      pipeline_root='dummy_root',
      output_path=__file__.replace('.py', '.json'))
