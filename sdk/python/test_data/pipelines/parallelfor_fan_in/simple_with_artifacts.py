from typing import List

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.component
def double(num: int, dataset: Output[Dataset]):
    with open(dataset.path, 'w') as f:
        f.write(str(2 * num))


@dsl.component
def add(datasets: Input[List[Dataset]]) -> int:
    nums = []
    for dataset in datasets:
        with open(dataset.path) as f:
            nums.append(int(f.read()))
    return sum(nums)


@dsl.pipeline
def math_pipeline() -> int:
    with dsl.ParallelFor([1, 2, 3]) as x:
        t = double(num=x)

    return add(datasets=dsl.Collected(t.outputs['dataset'])).output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
