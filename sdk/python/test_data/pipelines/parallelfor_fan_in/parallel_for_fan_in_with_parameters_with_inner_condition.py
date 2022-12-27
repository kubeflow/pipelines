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
def math_pipeline(threshold: int = 2) -> int:
    with dsl.ParallelFor([1, 2, 3]) as f:
        with dsl.Condition(f >= threshold):
            t = double(num=f)

    return add(nums=dsl.Aggregated(t.output)).output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
