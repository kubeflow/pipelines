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

from kfp.v2 import components
from kfp.v2 import dsl
from kfp.v2 import compiler
from kfp.v2.dsl import component


@component
def flip_coin_op() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    return result


@component
def print_op(msg: str):
    """Print a message."""
    print(msg)


@dsl.pipeline(name='nested-conditions-pipeline')
def my_pipeline():
    flip1 = flip_coin_op()
    print_op(msg=flip1.output)
    flip2 = flip_coin_op()
    print_op(msg=flip2.output)

    with dsl.Condition(flip1.output != 'no-such-result'):  # always true
        flip3 = flip_coin_op()
        print_op(msg=flip3.output)

        with dsl.Condition(flip2.output == flip3.output):
            flip4 = flip_coin_op()
            print_op(msg=flip4.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))
