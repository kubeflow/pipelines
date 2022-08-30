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
"""Two step pipeline using dsl.container_component decorator."""

from kfp import compiler
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import pipeline


@container_component
def component1(text: str, output_gcs: Output[Dataset]):
    return ContainerSpec(
        image='alpine',
        command=[
            'sh',
            '-c',
            'mkdir --parents $(dirname "$1") && echo "$0" > "$1"',
        ],
        args=[text, output_gcs.path])


@container_component
def component2(input_gcs: Input[Dataset]):
    return ContainerSpec(image='alpine', command=['cat'], args=[input_gcs.path])


@pipeline(name='two-step-pipeline-containerized')
def two_step_pipeline_containerized():
    component_1 = component1(text='hi')
    component_2 = component2(input_gcs=component_1.outputs['output_gcs'])


if __name__ == '__main__':
    # execute only if run as a script

    compiler.Compiler().compile(
        pipeline_func=two_step_pipeline_containerized,
        package_path='two_step_pipeline_containerized.yaml')
