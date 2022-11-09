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
"""Pipeline using multiple ExitHandlers."""

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def print_op(message: str):
    """Prints a message."""
    print(message)


@component
def fail_op(message: str):
    """Fails."""
    import sys
    print(message)
    sys.exit(1)


@dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
def my_pipeline(message: str = 'Hello World!'):

    first_exit_task = print_op(message='First exit handler has worked!')

    with dsl.ExitHandler(first_exit_task):
        first_exit_print_task = print_op(message=message)
        fail_op(message='Task failed.')

    second_exit_task = print_op(message='Second exit handler has worked!')

    with dsl.ExitHandler(second_exit_task):
        print_op(message=message)

    third_exit_task = print_op(message='Third exit handler has worked!')

    with dsl.ExitHandler(third_exit_task):
        print_op(message=message)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
