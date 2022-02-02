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

from kfp.v2 import components
from kfp.v2 import dsl
from kfp.v2 import compiler
from kfp.v2.dsl import component


@component
def print_op(text: str) -> str:
    print(text)
    return text


@component
def print_op2(text1: str, text2: str) -> str:
    print(text1 + text2)
    return text1 + text2


@dsl.pipeline(name='pipeline-with-pipelineparam-containing-format')
def my_pipeline(name: str = 'KFP'):
    print_task = print_op(text='Hello {}'.format(name))
    print_op(text='{}, again.'.format(print_task.output))

    new_value = f' and {name}.'
    with dsl.ParallelFor(['1', '2']) as item:
        print_op2(text1=item, text2=new_value)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))
