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
def echo(message: str):
    return dsl.ContainerSpec(
        image='library/bash',
        command=['sh', '-c'],
        args=[f'echo {message}'],
    )


@dsl.pipeline
def sequential(param1: str, param2: str):
    echo(message=param1)
    echo(message=param2)


if __name__ == '__main__':
    compiler.Compiler().compile(
        sequential, package_path=__file__.replace('.py', '.yaml'))
