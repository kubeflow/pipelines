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

import kfp
from kfp import dsl, components
from typing import NamedTuple

@components.create_component_from_func
def training_op(n: int) -> int:
    # quickly allocate a lot of memory to verify memory is enough
    a = [i for i in range(n)]
    return len(a)

@components.create_component_from_func
def generate_resouce_request() -> NamedTuple('output', [('memory', str), ('cpu', str)]):
    '''Returns the memory and cpu request'''
    from collections import namedtuple
    
    resouce_output = namedtuple('output', ['memory', 'cpu'])
    return resouce_output('500Mi', '200m')

@dsl.pipeline(
    name='Runtime resource request pipeline',
    description='An example on how to make resource requests at runtime.'
)
def resource_request_pipeline(n: int = 11234567):
    resouce_task = generate_resouce_request()
    traning_task = training_op(n)\
        .set_memory_limit(resouce_task.outputs['memory'])\
        .set_cpu_limit(resouce_task.outputs['cpu'])\
        .set_cpu_request('200m')

    # Disable cache for KFP v1 mode.
    traning_task.execution_options.caching_strategy.max_cache_staleness = 'P0D'

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(resource_request_pipeline, __file__ + '.yaml')