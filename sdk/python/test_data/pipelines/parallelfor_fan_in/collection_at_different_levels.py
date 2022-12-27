from typing import List, NamedTuple

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> NamedTuple('Outputs', [('a', int), ('b', int)]):
    Outputs = NamedTuple('Outputs', [('a', int), ('b', int)])
    res = 2 * num
    return Outputs(a=res, b=res)


@dsl.component
def sum_nested_lists(nums: List[List[int]]) -> int:
    import itertools
    return sum(itertools.chain(*nums))


@dsl.component
def sum_list(nums: List[int]) -> int:
    return sum(nums)


@dsl.component
def special_add(a: List[int], b: int) -> int:
    return sum(a) + b


@dsl.pipeline
def math_pipeline() -> int:
    with dsl.ParallelFor(['a', 'b', 'c']):
        with dsl.ParallelFor([1, 2, 3]) as f:
            result = double(num=f)

        inner_task = sum_list(nums=dsl.Collected(result.outputs['a']))

    outer_list = sum_nested_lists(nums=dsl.Collected(result.outputs['a']))

    return special_add(
        a=dsl.Collected(inner_task.output), b=outer_list.output).output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
