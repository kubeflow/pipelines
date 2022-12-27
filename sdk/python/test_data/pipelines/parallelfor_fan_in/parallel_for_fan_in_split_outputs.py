from typing import NamedTuple

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> NamedTuple('Outputs', [('a', int), ('b', int)]):
    Outputs = NamedTuple('Outputs', [('a', int), ('b', int)])
    res = 2 * num
    return Outputs(a=res, b=res)


@dsl.component
def add(nums: list) -> int:
    import itertools
    return sum(itertools.chain(*nums))


@dsl.component
def add_two_nums(a: int, b: int) -> int:
    return a + b


@dsl.pipeline
def math_pipeline() -> int:
    with dsl.ParallelFor(['a', 'b', 'c']):
        with dsl.ParallelFor([1, 2, 3]) as f:
            result = double(num=f)

        inner_task = add(nums=dsl.Aggregated(result.outputs['a']))

    outer_list = add(nums=dsl.Aggregated(result.outputs['a']))

    return add_two_nums(a=inner_task.output, b=outer_list.output).output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
