# Copyright 2021 The Kubeflow Authors
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
import json
from typing import List

from kfp import components
from kfp import dsl
from kfp.v2 import compiler


@components.create_component_from_func
def args_generator_op() -> str:
  import json
  return json.dumps(
      [{'A_a': '1', 'B_b': '2'}, {'A_a': '10', 'B_b': '20'}], sort_keys=True)


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
def my_pipeline(text_parameter: str = json.dumps([
    {'p_a': -1, 'p_b': 'hello'},
    {'p_a': 2, 'p_b': 'halo'},
    {'p_a': 3, 'p_b': 'ni hao'},
], sort_keys=True)):

  flip1 = flip_coin_op()

  with dsl.Condition(flip1.output != 'no-such-result'): # always true

    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
      print_op(text_parameter)

      with dsl.Condition(flip1.output == 'heads'):
        print_op(item.A_a)

      with dsl.Condition(flip1.output == 'tails'):
        print_op(item.B_b)

      with dsl.Condition(item.A_a == '1'):
        with dsl.ParallelFor([{'a':'-1'}, {'a':'-2'}]) as item:
          print_op(item)

  with dsl.ParallelFor(text_parameter) as item:
    with dsl.Condition(item.p_a > 0):
      print_op(item.p_a)
      print_op(item.p_b)


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      package_path=__file__.replace('.py', '.json'))
