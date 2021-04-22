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

from typing import List

from kfp import components
from kfp import dsl
from kfp.v2 import compiler


@components.create_component_from_func
def args_generator_op() -> str:
  import json
  return json.dumps(
      [{'A_a': '1'}, {'A_a': '10'}], sort_keys=True)


@components.create_component_from_func
def print_op(msg: str):
  print(msg)


@components.create_component_from_func
def flip_coin_op() -> str:
  """Flip a coin and output heads or tails randomly."""
  import random
  result = 'heads' if random.randint(0, 1) == 0 else 'tails'
  return result


@dsl.pipeline(
    name='pipeline-with-loops-and-conditions',
    pipeline_root='dummy_root',
)
def my_pipeline(text_parameter: str = 'Hello world!'):
  flip1 = flip_coin_op()

  with dsl.Condition(flip1.output != 'no-such-result'): # always true

    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
      print_op(text_parameter)
      print_op(item)

      with dsl.Condition(flip1.output == 'heads'):
        print_op(item.A_a)

      with dsl.Condition(flip1.output == 'tails'):
        print_op(item.B_b)

      with dsl.Condition(flip1.output != 'no-such-result'): # always true
        with dsl.ParallelFor(['a', 'b','c']) as item:
          print_op(item)


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      package_path=__file__.replace('.py', '.json'))
