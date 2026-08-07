from typing import List
from kfp.dsl import (
    Output,
    Artifact,
    component, pipeline, ParallelFor, Collected
)

@component()
def split_ids(model_ids: str) -> list:
    return model_ids.split(',')

@component()
def create_file(file: Output[Artifact], content: str):
    print(f'Creating file with content: {content}')
    with open(file.path, 'w') as f:
        f.write(content)

@component()
def read_values(values: List[str]) -> str:
    collect = []
    for v in values:
        collect.append(v)
    print(collect)
    assert sorted(collect) == sorted(['s1', 's2', 's3', 's4'])
    return 'values read'


@component()
def read_single_file(file: Artifact, expected: str) -> str:
    print(f'Reading file: {file.path}')
    with open(file.path, 'r') as f:
        data = f.read()
        print(data)
        assert expected == data
    return data

@pipeline()
def secondary_pipeline(model_ids: str = '',) -> List[str]:
    ids_split_op = split_ids(model_ids=model_ids)
    with ParallelFor(ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)
        read_single_file_task = read_single_file(file=create_file_op.outputs['file'], expected=model_id)
    read_values(values=Collected(read_single_file_task.output))
    return Collected(read_single_file_task.output)


@pipeline()
def primary_pipeline():
    model_ids = 's1,s2,s3,s4'
    dag = secondary_pipeline(model_ids=model_ids)
    read_values(values=dag.output)

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
