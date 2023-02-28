from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def simple_add(nums: List[int]) -> int:
    return sum(nums)


@dsl.component
def nested_add(nums: List[List[int]]) -> int:
    import itertools
    return sum(itertools.chain(*nums))


@dsl.component
def add_two_numbers(x: List[int], y: List[int]) -> int:
    return sum(x) + sum(y)


@dsl.pipeline
def math_pipeline() -> int:
    with dsl.ParallelFor([1, 2, 3]) as a:
        t1 = double(num=a)
        with dsl.ParallelFor([4, 5, 6]) as b:
            t2 = double(num=b)

        simple_add(nums=dsl.Collected(t2.output))

    nested_add(nums=dsl.Collected(t2.output))

    with dsl.ParallelFor([0, 0, 0]) as _:
        t3 = simple_add(nums=dsl.Collected(t1.output))
        t4 = nested_add(nums=dsl.Collected(t2.output))

    return add_two_numbers(
        x=dsl.Collected(t3.output), y=dsl.Collected(t4.output)).output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
