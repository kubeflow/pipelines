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
"""Pipeline using ExitHandler with PipelineTaskFinalStatus (YAML)."""

from kfp import compiler
from kfp import components
from kfp import dsl

exit_op = components.load_component_from_text("""
name: Exit Op
inputs:
- {name: user_input, type: String}
- {name: status, type: PipelineTaskFinalStatus}
implementation:
  container:
    image: python:3.9
    command:
    - echo
    - "user input:"
    - {inputValue: user_input}
    - "pipeline status:"
    - {inputValue: status}
""")

print_op = components.load_component_from_text("""
name: Print Op
inputs:
- {name: message, type: String}
implementation:
  container:
    image: python:3.9
    command:
    - echo
    - {inputValue: message}
""")


@dsl.pipeline(name='pipeline-with-task-final-status-yaml')
def my_pipeline(message: str = 'Hello World!'):
    exit_task = exit_op(user_input=message)

    with dsl.ExitHandler(exit_task, name='my-pipeline'):
        print_op(message=message)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
