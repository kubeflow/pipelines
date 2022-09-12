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
"""Pipeline using ExitHandler with PipelineTaskFinalStatus."""

from kfp import compiler
from kfp import dsl
from kfp.dsl import component
from kfp.dsl import PipelineTaskFinalStatus


@component
def exit_op(user_input: str, status: PipelineTaskFinalStatus):
    """Checks pipeline run status."""
    print('Pipeline status: ', status.state)
    print('Job resource name: ', status.pipeline_job_resource_name)
    print('Pipeline task name: ', status.pipeline_task_name)
    print('Error code: ', status.error_code)
    print('Error message: ', status.error_message)


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


@dsl.pipeline(name='pipeline-with-task-final-status')
def my_pipeline(message: str = 'Hello World!'):
    exit_task = exit_op(user_input=message)

    with dsl.ExitHandler(exit_task, name='my-pipeline'):
        print_op(message=message)
        fail_op(message='Task failed.')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
