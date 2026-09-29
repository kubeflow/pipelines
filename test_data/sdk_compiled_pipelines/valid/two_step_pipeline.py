# Copyright 2020 The Kubeflow Authors
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


@dsl.container_component
def write_to_gcs(text: str, output_gcs_path: dsl.Output[dsl.Artifact]):
    """
    Args:
        text: Content to be written to GCS
        output_gcs_path: GCS file path"""
    return dsl.ContainerSpec(
        image='google/cloud-sdk:slim',
        command=[
            'sh', '-c', 'set -e -x\necho "$0" | gsutil cp - "$1"\n', text,
            output_gcs_path.uri
        ],
    )


component_op_1 = write_to_gcs


@dsl.container_component
def read_from_gcs(input_gcs_path: dsl.Input[dsl.Artifact]):
    """
    Args:
        input_gcs_path: GCS file path"""
    return dsl.ContainerSpec(
        image='google/cloud-sdk:slim',
        command=[
            'sh', '-c', 'set -e -x\ngsutil cat "$0"\n', input_gcs_path.uri
        ],
    )


component_op_2 = read_from_gcs


@dsl.pipeline(name='simple-two-step-pipeline')
def my_pipeline(text: str = 'Hello world!'):
    component_1 = component_op_1(text=text).set_display_name('Producer')
    component_2 = component_op_2(
        input_gcs_path=component_1.outputs['output_gcs_path'])
    component_2.set_display_name('Consumer')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        pipeline_parameters={'text': 'Hello KFP!'},
        package_path=__file__.replace('.py', '.yaml'))
