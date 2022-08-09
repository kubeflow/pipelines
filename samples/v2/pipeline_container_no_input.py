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
import os

from kfp import compiler
from kfp import dsl


@dsl.container_component
def container_no_input():
    return dsl.ContainerSpec(
        image='python:3.7',
        command=['echo', 'hello world'],
        args=[],
    )


@dsl.pipeline(name='v2-container-component-no-input')
def pipeline_container_no_input():
    container_no_input()


if __name__ == '__main__':
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=pipeline_container_no_input,
        package_path='pipeline_container_no_input.yaml')
