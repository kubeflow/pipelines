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
from google_cloud_pipeline_components.types import artifact_types as google_artifact_types
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input


@dsl.container_component
def upstream(input_1: str, input_2: float, input_3: dsl.Input[dsl.Artifact],
             input_4: str, output_1: dsl.OutputPath(int),
             output_2: dsl.Output[dsl.Model],
             output_3: dsl.Output[dsl.Artifact],
             output_4: dsl.Output[dsl.Model],
             output_5: dsl.Output[dsl.Artifact],
             output_6: dsl.Output[dsl.Artifact],
             output_7: dsl.Output[dsl.Artifact], output_8: dsl.Output[dsl.HTML],
             output_9: dsl.Output[google_artifact_types.BQMLModel]):
    return dsl.ContainerSpec(
        image='gcr.io/image',
        args=[
            input_1, input_2, input_3.path, input_4, output_1, output_2.uri,
            output_3.path, output_4.uri, output_5.uri, output_6.path,
            output_7.path, output_8.path
        ],
    )


component_op_1 = upstream


@dsl.container_component
def downstream(input_a: int, input_b: dsl.Input[dsl.Model],
               input_c: dsl.Input[dsl.Artifact], input_d: dsl.Input[dsl.Model],
               input_e: dsl.Input[dsl.Artifact],
               input_f: dsl.Input[dsl.Artifact],
               input_g: dsl.Input[dsl.Artifact], input_h: dsl.Input[dsl.HTML],
               input_i: dsl.Input[google_artifact_types.BQMLModel]):
    return dsl.ContainerSpec(
        image='gcr.io/image',
        args=[
            input_a, input_b.uri, input_c.path, input_d.uri, input_e.uri,
            input_f.path, input_g.path, input_h.path
        ],
    )


component_op_2 = downstream


@dsl.pipeline(name='pipeline-with-various-types')
def my_pipeline(input1: str, input3: Input[Artifact], input4: str = ''):
    component_1 = component_op_1(
        input_1=input1,
        input_2=3.1415926,
        input_3=input3,
        input_4=input4,
    )
    component_2 = component_op_2(
        input_a=component_1.outputs['output_1'],
        input_b=component_1.outputs['output_2'],
        input_c=component_1.outputs['output_3'],
        input_d=component_1.outputs['output_4'],
        input_e=component_1.outputs['output_5'],
        input_f=component_1.outputs['output_6'],
        input_g=component_1.outputs['output_7'],
        input_h=component_1.outputs['output_8'],
        input_i=component_1.outputs['output_9'],
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
