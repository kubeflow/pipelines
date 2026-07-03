# Copyright 2022 The Kubeflow Authors
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


@dsl.container_component
def print_op2(msg: str):
    return dsl.ContainerSpec(
        image='alpine',
        command=['echo', msg],
    )


print_op3 = components.load_component_from_text("""
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


@dsl.pipeline(name='inner-pipeline')
def graph_component(msg: str):
    task = print_op1(msg=msg)
    with dsl.Condition(task.output == 'Hello'):
        print_op2(msg='world')
    with dsl.Condition(task.output != 'Hello'):
        print_op3(msg='Bye!')


@dsl.pipeline(name='pipeline-not-used')
def pipeline_not_used():
    graph_component(msg='abc')


@dsl.pipeline(name='pipeline-in-pipeline-complex')
def my_pipeline(msg: str = 'Hello'):
    print_op1(msg=msg)
    with dsl.ParallelFor(['Hello', 'world!']) as item:
        graph_component(msg=item)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
