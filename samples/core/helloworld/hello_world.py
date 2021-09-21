# Copyright 2019 The Kubeflow Authors
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
from kfp import dsl
import kfp.components as comp


@comp.create_component_from_func
def echo_op():
    print("Hello world")

@dsl.pipeline(
    name='my-first-pipeline',
    description='A hello world pipeline.'
)
def hello_world_pipeline():
    echo_task = echo_op()

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(hello_world_pipeline, __file__ + '.yaml')