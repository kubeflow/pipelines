from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def sum(num: List[int]) -> int:
    return sum(num)


@dsl.pipeline
def math_pipeline() -> List[int]:
    with dsl.ParallelFor([1, 2, 3]) as f:
        t = double(num=f)
    return dsl.Collected(t.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
