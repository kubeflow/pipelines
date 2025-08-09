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
def echo(param1: str, param2: str, output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='public.ecr.aws/docker/library/python:3.12',
        command=['sh', '-c'],
        args=[f'echo {param1}-{param2} > {output}'],
    )


@dsl.pipeline
def arguments_pipeline(param1: str, param2: str) -> str:
    first = echo(param1=param1, param2=param2)
    second = echo(param1=first.outputs['output'], param2=param2)
    return second.outputs['output']


if __name__ == '__main__':
    compiler.Compiler().compile(
        arguments_pipeline,
        package_path=__file__.replace('.py', '.yaml'),
        pipeline_parameters={'param1': 'hello', 'param2': 'world'})
