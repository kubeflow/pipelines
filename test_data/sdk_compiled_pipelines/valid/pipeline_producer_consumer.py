from typing import List, NamedTuple

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def add(nums: List[List[int]]) -> int:
    import itertools
    return sum(itertools.chain(*nums))


@dsl.pipeline
def double_pipeline(num: int) -> int:
    return double(num=num).output


@dsl.component
def echo_and_return(string: str) -> str:
    print(string)
    return string


@dsl.component
def join_and_print(strings: List[str]):
    print(''.join(strings))


@dsl.pipeline
def add_pipeline(
    nums: List[List[int]]
) -> NamedTuple('Outpus', [('out1', int), ('out2', List[str])]):
    with dsl.ParallelFor(['m', 'a', 't', 'h']) as char:
        id_task = echo_and_return(string=char)

    Outputs = NamedTuple('Outpus', [('out1', int), ('out2', List[str])])

    return Outputs(
        out1=add(nums=nums).output, out2=dsl.Collected(id_task.output))


@dsl.pipeline
def math_pipeline() -> int:
    with dsl.ParallelFor([1, 2, 3]) as x:
        with dsl.ParallelFor([1, 2, 3]) as x:
            t1 = double_pipeline(num=x)

    t2 = add_pipeline(nums=dsl.Collected(t1.output))
    join_and_print(strings=t2.outputs['out2'])
    return t2.outputs['out1']


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
