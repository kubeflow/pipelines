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

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import ContainerSpec
from kfp.dsl import Output


@dsl.container_component
def container_with_placeholder_in_fstring(
    output_artifact: Output[Artifact],
    text1: str = 'text!',
):
    return ContainerSpec(
        image='python:3.9',
        command=[
            'my_program',
            f'prefix-{text1}',
            f'{output_artifact.uri}/0',
        ])


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=container_with_placeholder_in_fstring,
        package_path=__file__.replace('.py', '.yaml'))
