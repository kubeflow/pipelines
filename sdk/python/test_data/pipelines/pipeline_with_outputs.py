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
import collections
from typing import NamedTuple

from kfp import compiler
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Output


@dsl.component
def print_op1(msg: str) -> str:
    print(msg)
    return msg


@dsl.container_component
def print_op2(msg: str, data: Output[Artifact]):
    return dsl.ContainerSpec(
        image='alpine',
        command=[
            'sh',
            '-c',
            'mkdir --parents $(dirname "$1") && echo "$0" > "$1"',
        ],
        args=[msg, data.path],
    )


@dsl.pipeline
def inner_pipeline(
        msg: str) -> NamedTuple('Outputs', [
            ('msg', str),
            ('data', Artifact),
        ]):
    task1 = print_op1(msg=msg)
    task2 = print_op2(msg=task1.output)

    output = collections.namedtuple('Outputs', ['msg', 'data'])
    return output(task1.output, task2.output)


@dsl.pipeline(name='pipeline-in-pipeline')
def my_pipeline(msg: str = 'Hello') -> Artifact:
    task1 = print_op1(msg=msg)
    task2 = inner_pipeline(msg='world')
    return task2.outputs['data']


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
