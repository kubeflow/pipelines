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

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def print_op(text: str) -> str:
    print(text)
    return text


@component
def print_list(l: list):
    print(l)


@component
def print_dict(d: dict):
    print(d)


@dsl.pipeline(name='pipeline-with-pipelineparam-containing-format')
def my_pipeline(name: str = 'KFP'):
    print_task = print_op(text=f'Hello {name}')
    print_op(text=f'{print_task.output}, again.')
    print_list(l=[f'Hello {name}'])
    print_dict(d={f'Hello {name}': f'How are you, {name}?'})

    with dsl.ParallelFor(['Hi', 'Howdy']) as greeting:
        new_value = f'{greeting}, {name}'
        print_op(text=new_value)
        print_list(l=[f'Hello {name}'])
        print_dict(d={f'Hello {name}': f'How are you, {name}?'})


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
