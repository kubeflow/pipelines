# Copyright 2021 Google LLC
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
"""Tests for kfp.dsl._component."""


import kfp
from kfp import dsl


def print_op(msg: str):
    """Print a message."""
    return dsl.ContainerOp(
        name='Print',
        image='alpine:3.6',
        command=['echo', msg],
    )


def flip_coin_op():
    """Flip a coin and output heads or tails randomly."""
    return dsl.ContainerOp(
        name='Flip coin',
        image='python:alpine3.6',
        command=['sh', '-c'],
        arguments=['python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                   'else \'tails\'; print(result)" | tee /tmp/output'],
        file_outputs={'output': '/tmp/output'}
    )


@dsl.graph_component
def flip_component(*args):
    # Using dynamic parameters
    print_flip = print_op(*args)
    flipA = flip_coin_op().after(print_flip)
    with dsl.Condition(flipA.output == 'heads'):
        flip_component(*args)


@dsl.pipeline(
    name='my-pipeline',
    description='A pipeline.'
)
def run():
    first_flip = flip_coin_op()
    first_flip.execution_options.caching_strategy.max_cache_staleness = "P0D"
    flip_loop = flip_component(first_flip.output)
    print_op('cool, it is over.').after(flip_loop)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(run, __file__ + '.yaml')
