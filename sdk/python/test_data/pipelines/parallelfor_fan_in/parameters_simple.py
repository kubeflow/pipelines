from typing import List

from kfp import compiler
from kfp import dsl


@dsl.component
def double(num: int) -> int:
    return 2 * num


@dsl.component
def add(nums: List[int]) -> int:
    return sum(nums)


@dsl.container_component
def add_container(nums: List[int], sum: dsl.OutputPath(int)):
    return dsl.ContainerSpec(
        image='stedolan/jq',
        command=['sh', '-c'],
        args=[
            f"""
            set -ex
            mkdir -p $(dirname {sum})
            echo {nums} | jq 'add' > {sum}
            """
        ],
    )


@dsl.pipeline
def math_pipeline() -> List[int]:
    with dsl.ParallelFor([1, 2, 3]) as x:
        t = double(num=x)
    add(nums=dsl.Collected(t.output))
    add_container(nums=dsl.Collected(t.output))
    return dsl.Collected(t.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
