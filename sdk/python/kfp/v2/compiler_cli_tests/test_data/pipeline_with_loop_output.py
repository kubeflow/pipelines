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


@dsl.pipeline(
    name='pipeline-with-loop-output-args',
    pipeline_root='dummy_root',
)
def my_pipeline():

  args_generator = args_generator_op()
  with dsl.ParallelFor(args_generator.output) as item:
    print_op(item)
    print_op(item.A_a)
    print_op(item.B_b)


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      package_path=__file__.replace('.py', '.json'))
