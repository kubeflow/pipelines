# Copyright 2021 The Kubeflow Authors
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

from kfp.v2.components.experimental import base_component
from kfp.v2.components.experimental import component_spec
from kfp.v2 import dsl
from kfp.v2 import compiler


class TestComponent(base_component.BaseComponent):

    def execute(self, *args, **kwargs):
        pass


component_op = TestComponent(
    component_spec=component_spec.ComponentSpec(
        name='component_1',
        implementation=component_spec.ContainerSpec(
            image='alpine',
            commands=[
                'sh',
                '-c',
                'set -ex\necho "$0" > "$1"',
                component_spec.InputValuePlaceholder(input_value='input1'),
                component_spec.OutputPathPlaceholder(output_path='output1'),
            ],
        ),
        inputs={'input1': component_spec.InputSpec(type='String')},
        outputs={'output1': component_spec.OutputSpec(type='String')},
    ))


@dsl.pipeline(name='experimental-v2-component', pipeline_root='dummy_root')
def my_pipeline(text: str = 'Hello world!'):
    component_1 = component_op(input1=text)
    component_2 = component_op(input1=component_1.outputs['output1'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))
