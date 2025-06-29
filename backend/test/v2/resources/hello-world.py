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
def echo(message: str, output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='public.ecr.aws/docker/library/python:3.12',
        command=['sh', '-c'],
        args=[f'echo {message} > {output}'],
    )


@dsl.pipeline(name='hello-world-pipeline')
def hello_world_pipeline(input_message: str) -> str:
    first = echo(message=input_message)
    second = echo(message=first.outputs['output'])
    return second.outputs['output']


if __name__ == '__main__':
    compiler.Compiler().compile(
        hello_world_pipeline, package_path=__file__.replace('.py', '.yaml'))
