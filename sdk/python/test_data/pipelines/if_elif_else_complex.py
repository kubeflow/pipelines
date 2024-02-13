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
from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def int_0_to_9999() -> int:
    import random
    return random.randint(0, 9999)


@dsl.component
def is_even_or_odd(num: int) -> str:
    return 'odd' if num % 2 else 'even'


@dsl.component
def print_and_return(text: str) -> str:
    print(text)
    return text


@dsl.component
def print_strings(strings: List[str]):
    print(strings)


@dsl.component
def print_ints(ints: List[int]):
    print(ints)


@dsl.pipeline
def lucky_number_pipeline(add_drumroll: bool = True,
                          repeat_if_lucky_number: bool = True,
                          trials: List[int] = [1, 2, 3]):
    with dsl.ParallelFor(trials) as trial:
        int_task = int_0_to_9999().set_caching_options(False)
        with dsl.If(add_drumroll == True):
            with dsl.If(trial == 3):
                print_and_return(text='Adding drumroll on last trial!')

        with dsl.If(int_task.output < 5000):

            even_or_odd_task = is_even_or_odd(num=int_task.output)

            with dsl.If(even_or_odd_task.output == 'even'):
                t1 = print_and_return(text='Got a low even number!')
            with dsl.Else():
                t2 = print_and_return(text='Got a low odd number!')

            repeater_task = print_and_return(
                text=dsl.OneOf(t1.output, t2.output))

        with dsl.Elif(int_task.output > 5000):

            even_or_odd_task = is_even_or_odd(num=int_task.output)

            with dsl.If(even_or_odd_task.output == 'even'):
                t3 = print_and_return(text='Got a high even number!')
            with dsl.Else():
                t4 = print_and_return(text='Got a high odd number!')

            repeater_task = print_and_return(
                text=dsl.OneOf(t3.output, t4.output))

        with dsl.Else():
            print_and_return(
                text='Announcing: Got the lucky number 5000! A one in 10,000 chance.'
            )
            with dsl.If(repeat_if_lucky_number == True):
                with dsl.ParallelFor([1, 2]) as _:
                    print_and_return(
                        text='Announcing again: Got the lucky number 5000! A one in 10,000 chance.'
                    )

    print_ints(ints=dsl.Collected(int_task.output))


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=lucky_number_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
