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

from kfp import components
from kfp import dsl


@components.create_component_from_func
def args_generator_op() -> str:
    return '[1.1, 1.2, 1.3]'


@components.create_component_from_func
def print_op(s: float):
    print(s)


@dsl.pipeline(name='pipeline-with-loop-output')
def my_pipeline():
    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
        print_op(item)
