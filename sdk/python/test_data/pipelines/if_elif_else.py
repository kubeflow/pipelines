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
from kfp import compiler
from kfp import dsl


@dsl.component
def flip_three_sided_die() -> str:
    import random
    val = random.randint(0, 2)

    if val == 0:
        return 'heads'
    elif val == 1:
        return 'tails'
    else:
        return 'draw'


@dsl.component
def print_and_return(text: str) -> str:
    print(text)
    return text


@dsl.pipeline
def roll_die_pipeline():
    flip_coin_task = flip_three_sided_die()
    with dsl.If(flip_coin_task.output == 'heads'):
        print_and_return(text='Got heads!')
    with dsl.Elif(flip_coin_task.output == 'tails'):
        print_and_return(text='Got tails!')
    with dsl.Else():
        print_and_return(text='Draw!')


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=roll_die_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
