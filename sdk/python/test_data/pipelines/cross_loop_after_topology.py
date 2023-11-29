# Copyright 2023 The Kubeflow Authors
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
"""Tests various forms of .after usage across ParallelFor contexts. We already
have unit tests for invalid pipeline topologies, so this test is for asserting
the correctness of the compiled IR.

Each of these after() calls should fail if they were replaced by data
exchange.
"""

from kfp import dsl


@dsl.component
def print_op(message: str):
    print(message)


@dsl.pipeline()
def my_pipeline():
    with dsl.ParallelFor([1, 2]):
        one = print_op(message='one')

    with dsl.ParallelFor([1, 2]):
        two = print_op(message='two').after(one)

    with dsl.ParallelFor([1, 2]):
        three = print_op(message='three')

        with dsl.ParallelFor([1, 2]):
            four = print_op(message='four').after(three)

    with dsl.ParallelFor([1, 2]):
        five = print_op(message='five')
    six = print_op(message='six').after(five)

    with dsl.ParallelFor([1, 2]):
        with dsl.ParallelFor([1, 2]):
            seven = print_op(message='seven')
        eight = print_op(message='eight').after(seven)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
