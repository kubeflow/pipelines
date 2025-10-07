from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def add(nums: List[int]) -> int:
    return sum(nums)


@dsl.pipeline
def math_pipeline(threshold: int = 2) -> List[int]:
    with dsl.ParallelFor([1, 2, 3]) as x:
        with dsl.Condition(x >= threshold):
            t = double(num=x)

    with dsl.Condition(threshold == 2):
        t = add(nums=dsl.Collected(t.output))
    return dsl.Collected(t.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
