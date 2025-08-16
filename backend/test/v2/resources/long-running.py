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

from kfp import dsl, compiler


@dsl.container_component
def wait_op(
    duration: int, message: str, output: dsl.OutputPath(str)
):
    return dsl.ContainerSpec(
        image='alpine:latest',
        command=['sh', '-c'],
        args=[
            f'echo {message} sleeping for {duration}s; sleep {duration}; echo done > {output}'
        ],
    )


@dsl.pipeline
def wait_awhile(duration: int = 300) -> str:
    task1 = wait_op(duration=duration, message='step-1',)
    task2 = wait_op(duration=duration, message=task1.outputs['output'])
    return task2.outputs['output']


if __name__ == '__main__':
    compiler.Compiler().compile(
        wait_awhile, package_path=__file__.replace('.py', '.yaml'))
