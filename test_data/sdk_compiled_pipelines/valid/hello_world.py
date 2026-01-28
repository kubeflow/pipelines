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
def echo():
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=['echo'],
        args=['hello world'],
    )


@dsl.container_component
def echo_producer(message: str, output_file: dsl.Output[dsl.Dataset]):
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=[
            'sh',
            '-c',
            'mkdir -p $(dirname "$1") && echo "$0" > "$1"',
        ],
        args=[message, output_file.path],
    )


@dsl.container_component
def echo_consumer(input_file: dsl.Input[dsl.Dataset]):
    return dsl.ContainerSpec(
        image='registry.access.redhat.com/ubi9/python-311:latest',
        command=['cat'],
        args=[input_file.path],
    )


@dsl.pipeline(name='hello-world-pipeline')
def hello_world_pipeline(greeting: str = 'hello world'):
    producer = echo_producer(message=greeting)
    consumer = echo_consumer(input_file=producer.outputs['output_file'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        echo, package_path=__file__.replace('.py', '.yaml'))
