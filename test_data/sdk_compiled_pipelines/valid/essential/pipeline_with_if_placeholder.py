# Copyright 2020,2021 The Kubeflow Authors
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


@dsl.container_component
def component_with_optional_inputs(required_input: str,
                                   optional_input_1: str = None,
                                   optional_input_2: str = None):
    return dsl.ContainerSpec(
        image='ghcr.io/containerd/busybox',
        command=['echo'],
        args=[
            '--arg0', required_input,
            dsl.IfPresentPlaceholder(
                input_name='optional_input_1',
                then=['--arg1', optional_input_1]),
            dsl.IfPresentPlaceholder(
                input_name='optional_input_2',
                then=['--arg2', optional_input_2],
                else_=['--arg2', 'default value'])
        ],
    )


component_op = component_with_optional_inputs


@dsl.pipeline(name='one-step-pipeline-with-if-placeholder-supply-both')
def pipeline_both(input0: str = 'input0',
                  input1: str = 'input1',
                  input2: str = 'input2'):
    # supply both optional_input_1 and optional_input_2
    component = component_op(
        required_input=input0, optional_input_1=input1, optional_input_2=input2)


@dsl.pipeline(name='one-step-pipeline-with-if-placeholder-supply-none')
def pipeline_none(input0: str = 'input0'):
    # supply neither optional_input_1 nor optional_input_2
    # Note, KFP only supports compile-time optional arguments, e.g. it's not
    # supported to write a pipeline that supplies both inputs and pass None
    # at runtime -- in that case, the input arguments will be interpreted as
    # the raw text "None".
    component = component_op(required_input=input0)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=pipeline_none,
        package_path=__file__.replace('.py', '.yaml'))
