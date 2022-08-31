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

import pathlib

from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input


@dsl.component
def print_op1(data: Input[Artifact]):
    with open(data.path, 'r') as f:
        print(f.read())


reuse_yaml_pipeline = components.load_component_from_file(
    pathlib.Path(__file__).parent / 'pipeline_with_outputs.yaml')


@dsl.pipeline(name='pipeline-in-pipeline')
def my_pipeline():
    task = reuse_yaml_pipeline(msg='Hello')
    print_op1(data=task.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
