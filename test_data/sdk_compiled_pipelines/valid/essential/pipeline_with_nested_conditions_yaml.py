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

from kfp import compiler
from kfp import dsl


@dsl.container_component
def generate_random_number(low: int, high: int, output: dsl.OutputPath(int)):
    return dsl.ContainerSpec(
        image='python:alpine3.9',
        command=['sh', '-c'],
        args=[
            'mkdir -p "$(dirname $2)" && python -c "import random; print(random.randint($0, $1), end=\'\')" | tee $2',
            low, high, output
        ],
    )


@dsl.container_component
def flip_coin(output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='python:alpine3.9',
        command=['sh', '-c'],
        args=[
            'mkdir -p "$(dirname $0)" && python -c "import random; print(\'heads\' if random.randint(0,1) == 0 else \'tails\', end=\'\')" | tee $0',
            output
        ],
    )


@dsl.container_component
def print_op(msg: str):
    return dsl.ContainerSpec(image='python:alpine3.9', command=['echo', msg])


@dsl.pipeline(
    name='conditional-execution-pipeline',
    display_name='Conditional execution pipeline.',
    description='Shows how to use dsl.Condition().')
def my_pipeline():
    flip = flip_coin()
    with dsl.Condition(flip.output == 'heads'):
        random_num_head = generate_random_number(low=0, high=9)
        with dsl.Condition(random_num_head.output > 5):
            print_op(msg='heads and %s > 5!' % random_num_head.output)
        with dsl.Condition(random_num_head.output <= 5):
            print_op(msg='heads and %s <= 5!' % random_num_head.output)

    with dsl.Condition(flip.output == 'tails'):
        random_num_tail = generate_random_number(low=10, high=19)
        with dsl.Condition(random_num_tail.output > 15):
            print_op(msg='tails and %s > 15!' % random_num_tail.output)
        with dsl.Condition(random_num_tail.output <= 15):
            print_op(msg='tails and %s <= 15!' % random_num_tail.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
