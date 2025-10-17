from typing import List, NamedTuple

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.component
def double(
    num: int,
    out_dataset: Output[Dataset],
):
    with open(out_dataset.path, 'w') as f:
        f.write(str(2 * num))


@dsl.component
def add(
    in_datasets: Input[List[Dataset]],
    out_dataset: Output[Dataset],
):
    nums = []
    for dataset in in_datasets:
        with open(dataset.path) as f:
            nums.append(int(f.read()))
    with open(out_dataset.path, 'w') as f:
        f.write(str(sum(nums)))


@dsl.component
def add_two_ints(
    in_dataset1: Input[Dataset],
    in_dataset2: Input[Dataset],
    out_dataset: Output[Dataset],
):
    with open(in_dataset1.path) as f:
        in_dataset1 = int(f.read())

    with open(in_dataset2.path) as f:
        in_dataset2 = int(f.read())

    with open(out_dataset.path, 'w') as f:
        f.write(str(in_dataset1 + in_dataset2))


@dsl.pipeline
def add_two_lists_of_datasets(
    in_datasets1: Input[List[Dataset]],
    in_datasets2: Input[List[Dataset]],
) -> Dataset:
    sum1_task = add(in_datasets=in_datasets1)
    sum2_task = add(in_datasets=in_datasets2)
    return add_two_ints(
        in_dataset1=sum1_task.outputs['out_dataset'],
        in_dataset2=sum2_task.outputs['out_dataset']).outputs['out_dataset']


@dsl.pipeline
def math_pipeline(
    threshold: int = 2
) -> NamedTuple('Outputs', [
    ('sum', Dataset),
    ('datasets', List[Dataset]),
]):
    with dsl.ParallelFor([1, 2, 3]) as x:
        double_task1 = double(num=x)
        with dsl.ParallelFor([1, 2, 3]) as y:
            with dsl.Condition(y >= threshold):
                double_task2 = double(num=y)
    sum_task = add_two_lists_of_datasets(
        in_datasets1=dsl.Collected(double_task1.outputs['out_dataset']),
        in_datasets2=dsl.Collected(double_task2.outputs['out_dataset']),
    )
    Outputs = NamedTuple('Outputs', [
        ('sum', Dataset),
        ('datasets', List[Dataset]),
    ])
    return Outputs(
        sum=sum_task.output,
        datasets=dsl.Collected(double_task1.outputs['out_dataset']))


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
