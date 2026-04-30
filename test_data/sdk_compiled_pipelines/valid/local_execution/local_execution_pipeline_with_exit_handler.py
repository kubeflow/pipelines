# Copyright 2026 The Kubeflow Authors
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
"""Pipeline exercising `dsl.ExitHandler` + `PipelineTaskFinalStatus` for local-
execution regression tests.

On success the body task runs, and the exit task runs with
`state='SUCCEEDED'`. The pipeline has no output because KFP rejects
returning a value from inside an ExitHandler.
"""
from kfp import compiler
from kfp import dsl
from kfp.dsl import PipelineTaskFinalStatus


@dsl.component
def record_exit(status: PipelineTaskFinalStatus) -> str:
    return f'exit:{status.state}'


@dsl.component
def echo(message: str) -> str:
    return message


@dsl.pipeline
def pipeline_with_exit_handler(message: str = 'hello'):
    """Runs `echo` inside an ExitHandler.

    On success the exit task receives a PipelineTaskFinalStatus with
    state='SUCCEEDED'. No pipeline output — KFP's compiler rejects
    returning a value from inside an ExitHandler.
    """
    exit_task = record_exit()
    with dsl.ExitHandler(exit_task):
        echo(message=message)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_exit_handler,
        package_path=__file__.replace('.py', '.yaml'),
    )
