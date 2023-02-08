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

from kfp import compiler
from kfp import dsl


@dsl.component
def fail_op(message: str) -> str:
    """Fails."""
    import sys
    print(message)
    sys.exit(1)
    return message


@dsl.component
def print_op(message: str = 'msg'):
    """Prints a message."""
    print(message)


@dsl.pipeline()
def my_pipeline(sample_input1: str = 'message'):
    task = fail_op(message=sample_input1)
    clean_up = print_op(message=task.output)
    task.set_exit_task(exit_task=clean_up)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
