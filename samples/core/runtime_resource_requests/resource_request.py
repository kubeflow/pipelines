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

@components.create_component_from_func
def print_op():
   '''Print hello world'''
   print("hello world")

@dsl.pipeline(
    name='Runtime resource request pipeline',
    description='An example on how to make resource requests at runtime.'
)
def resource_request_pipeline(memory_request: str="100Mi", cpu_request: str="100m"):
    echo_task = print_op().add_memory_request(memory_request).add_cpu_request(cpu_request)

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(resource_request_pipeline, __file__ + '.yaml')