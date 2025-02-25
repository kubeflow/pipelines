import textwrap
from typing import List

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.component
def double(num: int, out_dataset: Output[Dataset]):
    with open(out_dataset.path, 'w') as f:
        f.write(str(2 * num))


@dsl.component
def add(in_datasets: Input[List[Dataset]], out_dataset: Output[Dataset]):
    nums = []
    for dataset in in_datasets:
        with open(dataset.path) as f:
            nums.append(int(f.read()))
    with open(out_dataset.path, 'w') as f:
        f.write(str(sum(nums)))


@dsl.container_component
def add_container(in_datasets: Input[List[Dataset]],
                  out_dataset: Output[Dataset]):
    return dsl.ContainerSpec(
        image='python:3.9',
        command=['python', '-c'],
        args=[
            textwrap.dedent("""
            import argparse
            import json
            import os

            def main(in_datasets, out_dataset_uri):
                in_dicts = json.loads(in_datasets)
                uris = [d['uri'] for d in in_dicts]
                total = 0
                for uri in uris:
                    with open(uri.replace('gs://', '/gcs/')) as f:
                        total += int(f.read())

                outpath = out_dataset_uri.replace('gs://', '/gcs/')
                os.makedirs(os.path.dirname(outpath), exist_ok=True)
                with open(outpath, 'w') as f:
                    f.write(str(total))

            parser = argparse.ArgumentParser()
            parser.add_argument('in_datasets')
            parser.add_argument('out_dataset_uri')
            args = parser.parse_args()

            main(args.in_datasets, args.out_dataset_uri)
            """),
            in_datasets,
            out_dataset.uri,
        ],
    )


@dsl.pipeline
def math_pipeline() -> List[Dataset]:
    with dsl.ParallelFor([1, 2, 3]) as x:
        t = double(num=x)
    add(in_datasets=dsl.Collected(t.outputs['out_dataset']))
    add_container(in_datasets=dsl.Collected(t.outputs['out_dataset']))
    return dsl.Collected(t.outputs['out_dataset'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=math_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
