from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def add(nums: List[List[int]]) -> int:
    import itertools
    return sum(itertools.chain(*nums))


@dsl.component
def add_two_nums(x: int, y: int) -> int:
    return x + y


@dsl.pipeline
def math_pipeline() -> List[int]:
    with dsl.ParallelFor([1, 2, 3]) as x:
        with dsl.ParallelFor([1, 2, 3]) as y:
            t1 = double(num=x)
            t2 = double(num=y)
            t3 = add_two_nums(x=t1.output, y=t2.output)
    t4 = add(nums=dsl.Collected(t3.output))
    return dsl.Collected(t3.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
