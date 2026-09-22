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

# Simple two-step pipeline with 'producer' and 'consumer' steps
from kfp import compiler
from kfp import dsl


@dsl.container_component
def producer(input_text: str, output_value: dsl.OutputPath(str)):
    """
    Args:
        input_text: Represents an input parameter.
        output_value: Represents an output paramter."""
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=[
            'sh', '-c',
            'set -e -x\nmkdir -p "$(dirname "$1")"\necho "$0, this is an output parameter" > "$1"\n',
            input_text, output_value
        ],
    )


producer_op = producer


@dsl.container_component
def consumer(input_value: str):
    """
    Args:
        input_value: Represents an input parameter. It connects to an upstream output parameter."""
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=[
            'sh', '-c',
            'set -e -x\necho "Read from an input parameter: " && echo "$0"\n',
            input_value
        ],
    )


consumer_op = consumer


@dsl.pipeline(name='producer-consumer-param-pipeline')
def producer_consumer_param_pipeline(text: str = 'Hello world'):
    producer = producer_op(input_text=text)
    consumer = consumer_op(input_value=producer.outputs['output_value'])


if __name__ == "__main__":
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=producer_consumer_param_pipeline,
        package_path='producer_consumer_param_pipeline.yaml')
