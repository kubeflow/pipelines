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

from kfp import compiler
from kfp import dsl
from kfp.dsl import component


@component
def print_op(msg: str, value: str):
    print(msg, value)


@dsl.pipeline(name='pipeline-with-placeholders')
def my_pipeline():

    print_op(
        msg='job name:',
        value=dsl.PIPELINE_JOB_NAME_PLACEHOLDER,
    )
    print_op(
        msg='job resource name:',
        value=dsl.PIPELINE_JOB_RESOURCE_NAME_PLACEHOLDER,
    )
    print_op(
        msg='job id:',
        value=dsl.PIPELINE_JOB_ID_PLACEHOLDER,
    )
    print_op(
        msg='task name:',
        value=dsl.PIPELINE_TASK_NAME_PLACEHOLDER,
    )
    print_op(
        msg='task id:',
        value=dsl.PIPELINE_TASK_ID_PLACEHOLDER,
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
