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
def math_pipeline():
    with dsl.ParallelFor([1, 2, 3]) as outer:
        with dsl.ParallelFor([4, 5, 6]) as inner:
            task_a = double(num=inner)
            task_b = double(num=outer)
        task_c = double(num=outer)
        collected_a = add(nums=dsl.Collected(task_a.output))
        collected_b = add(nums=dsl.Collected(task_b.output))
    collected_aa = add(nums=dsl.Collected(task_a.output))
    with dsl.ParallelFor([1, 2, 3]) as third:
        collected_bb = add(nums=dsl.Collected(task_b.output))
        collected_c = add(nums=dsl.Collected(task_c.output))


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
