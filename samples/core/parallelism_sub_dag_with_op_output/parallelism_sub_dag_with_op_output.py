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

import kfp.dsl as dsl
import kfp

@kfp.components.create_component_from_func
def print_op(s: str):
    import time
    time.sleep(3)
    print(s)

@kfp.components.create_component_from_func
def dump_loop_args() -> list:
    return [{'A_a': 1, 'B_b': 2}, {'A_a': 10, 'B_b': 20}]

@dsl.pipeline(name='my-pipeline')
def pipeline():
    dump_loop_args_op = dump_loop_args()
    with dsl.SubGraph(parallelism=2):
        with dsl.ParallelFor(dump_loop_args_op.output) as item:
            print_op(item)
            print_op(item.A_a)
            print_op(item.B_b)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline, __file__ + '.yaml')
