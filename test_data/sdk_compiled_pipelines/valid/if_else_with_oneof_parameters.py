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
from kfp import dsl


@dsl.component
def flip_coin() -> str:
    import random
    return 'heads' if random.randint(0, 1) == 0 else 'tails'


@dsl.component
def print_and_return(text: str) -> str:
    print(text)
    return text


@dsl.pipeline
def flip_coin_pipeline() -> str:
    flip_coin_task = flip_coin()
    with dsl.If(flip_coin_task.output == 'heads'):
        print_task_1 = print_and_return(text='Got heads!')
    with dsl.Else():
        print_task_2 = print_and_return(text='Got tails!')
    x = dsl.OneOf(print_task_1.output, print_task_2.output)
    print_and_return(text=x)
    return x


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=flip_coin_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
