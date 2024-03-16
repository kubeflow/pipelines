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

from kfp import dsl, compiler
from typing import NamedTuple

@dsl.component
def training_op(n: int) -> int:
    # quickly allocate a lot of memory to verify memory is enough
    a = [i for i in range(n)]
    return len(a)

@dsl.component
def generate_resource_request() -> NamedTuple('output', memory=str, cpu=str):
    '''Returns the memory and cpu request'''
    resource_output = NamedTuple('output', memory=str, cpu=str)
    return resource_output('500Mi', '200m')

@dsl.pipeline(
    name='Runtime resource request pipeline',
    description='An example on how to make resource requests at runtime.'
)
def resource_request_pipeline(n: int = 11234567):
    resource_task = generate_resource_request()

    # TODO: support PipelineParameterChannel for resource input
    # TypeError: expected string or bytes-like object, got 'PipelineParameterChannel'
    # traning_task = training_op(n=n)\
    #     .set_memory_limit(resource_task.outputs['memory'])\
    #     .set_cpu_limit(resource_task.outputs['cpu'])\
    #     .set_cpu_request('200m')
    traning_task = training_op(n=n)\
        .set_memory_limit('500Mi')\
        .set_cpu_limit('200m')\
        .set_cpu_request('200m')

if __name__ == '__main__':
    compiler.Compiler().compile(resource_request_pipeline, __file__ + '.yaml')
