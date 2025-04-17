from typing import List

import kfp
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Output
import kfp.kubernetes


@dsl.component()
def split_ids(model_ids: str) -> list:
    return model_ids.split(',')


@dsl.component()
def create_file(file: Output[Artifact], content: str):
    print(f'Creating file with content: {content}')
    with open(file.path, 'w') as f:
        f.write(content)


@dsl.component()
def read_files(files: List[Artifact]) -> str:
    for f in files:
        print(f'Reading artifact {f.name} file: {f.path}')
        with open(f.path, 'r') as f:
            print(f.read())

    return 'files read'


@dsl.component()
def read_single_file(file: Artifact) -> str:
    print(f'Reading file: {file.path}')
    with open(file.path, 'r') as f:
        print(f.read())

    return file.uri


@dsl.pipeline()
def collecting_artifacts(model_ids: str = '',) -> List[Artifact]:
    ids_split_op = split_ids(model_ids=model_ids)
    with dsl.ParallelFor(ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)

        read_single_file_op = read_single_file(
            file=create_file_op.outputs['file'])

    read_file_op = read_files(
        files=dsl.Collected(create_file_op.outputs['file']))

    return dsl.Collected(create_file_op.outputs['file'])


@dsl.pipeline()
def collected_artifact_pipeline():
    model_ids = 's1,s2,s3'
    dag = collecting_artifacts(model_ids=model_ids)
    read_files_op = read_files(files=dag.output)


if __name__ == '__main__':
    client = kfp.Client()
    run = client.create_run_from_pipeline_func(
        collected_artifact_pipeline,
        arguments={},
        enable_caching=False,
    )
