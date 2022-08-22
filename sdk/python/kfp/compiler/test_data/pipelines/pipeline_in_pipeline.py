# Copyright 2020 The Kubeflow Authors
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

from kfp import compiler
from kfp import components
from kfp import dsl


@dsl.component
def print_op1(msg: str) -> str:
    print(msg)
    return msg


print_op2 = components.load_component_from_text("""
name: Print Op
inputs:
- {name: msg, type: String}
implementation:
  container:
    image: alpine
    command:
    - echo
    - {inputValue: msg}
""")


@dsl.pipeline(name='test-graph-component')
def graph_component(msg: str):
    task = print_op1(msg=msg)
    print_op2(msg=task.output)


@dsl.pipeline(name='pipeline-in-pipeline')
def my_pipeline():
    print_op1(msg='Hello')
    graph_component(msg='world')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
