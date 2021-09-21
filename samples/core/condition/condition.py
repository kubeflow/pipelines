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

from kfp import components
from kfp import dsl


def flip_coin(force_flip_result: str = '') -> str:
    """Flip a coin and output heads or tails randomly."""
    if force_flip_result:
        return force_flip_result
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    return result


def print_msg(msg: str):
    """Print a message."""
    print(msg)


flip_coin_op = components.create_component_from_func(flip_coin)

print_op = components.create_component_from_func(print_msg)


@dsl.pipeline(name='single-condition-pipeline')
def my_pipeline(text: str = 'condition test', force_flip_result: str = ''):
    flip1 = flip_coin_op(force_flip_result)
    print_op(flip1.output)

    with dsl.Condition(flip1.output == 'heads'):
        flip2 = flip_coin_op()
        print_op(flip2.output)
        print_op(text)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
