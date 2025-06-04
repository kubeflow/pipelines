from typing import List

import kfp
from kfp import dsl
from kfp.dsl import Artifact, Dataset, Model
from kfp.dsl import Output

# This sample pipeline is meant to cover the following cases during artifact resolution:
# 1. ParallelFor task consuming input from another task within the same loop.
# 2. A nested ParallelFor task consuming input from another ParallelFor Iterator.
# 3. Resolving input with dsl.Collected inside another ParallelFor loop.
# 4. Resolving input that comes from a subdag using dsl.Collected inside of a ParallelFor loop.
# 5. Returning a dsl.Collected, loop-nested, task as subdag's output.
# 6. Feeding in the subdag's collected output as input for a downstream task.

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

@dsl.component()
def split_chars(model_ids: str) -> list:
    return model_ids.split(',')


@dsl.component()
def create_dataset(data: Output[Dataset], content: str):
    print(f'Creating file with content: {content}')
    with open(data.path, 'w') as f:
        f.write(content)


@dsl.component()
def read_datasets(data: List[Dataset]) -> str:
    for d in data:
        print(f'Reading dataset {d.name} file: {d.path}')
        with open(d.path, 'r') as f:
            print(f.read())

    return 'files read'


@dsl.component()
def read_single_dataset_generate_model(data: Dataset, id: str, results:Output[Model]):
    print(f'Reading file: {data.path}')
    with open(data.path, 'r') as f:
        info = f.read()
        with open(results.path, 'w') as f2:
            f2.write(f"{info}-{id}")
            results.metadata['model'] = info
            results.metadata['model_name'] = f"model-artifact-inner-iteration-{info}-{id}"


@dsl.component()
def read_models(models: List[Model],) -> str:
    for m in models:
        print(f'Reading model {m.name} file: {m.path}')
        with open(m.path, 'r') as f:
            info = f.read()
            print(f"Model raw data: {info}")
            print(f"Model metadata: {m.metadata}")
    return 'models read'
    
@dsl.pipeline()
def single_node_dag(char:str)-> Dataset:
    create_dataset_op = create_dataset(content=char)
    create_dataset_op.set_caching_options(False)
    return create_dataset_op.outputs["data"]

@dsl.pipeline()
def collecting_artifacts(model_ids: str = '', model_chars: str = '') -> List[Model]:
    ids_split_op = split_ids(model_ids=model_ids)
    ids_split_op.set_caching_options(False)

    char_split_op = split_chars(model_ids=model_chars)
    char_split_op.set_caching_options(False)
    
    with dsl.ParallelFor(ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)
        create_file_op.set_caching_options(False)

        read_single_file_op = read_single_file(
            file=create_file_op.outputs['file'])
        read_single_file_op.set_caching_options(False)

        with dsl.ParallelFor(char_split_op.output) as model_char:
            single_dag_op = single_node_dag(char=model_char)
            single_dag_op.set_caching_options(False)

            random_subdag_op = read_single_dataset_generate_model_op = read_single_dataset_generate_model(data=single_dag_op.output, id=model_id)
            read_single_dataset_generate_model_op.set_caching_options(False)
        
        read_models_op = read_models(
            models=dsl.Collected(read_single_dataset_generate_model_op.outputs['results']))
        read_models_op.set_caching_options(False)

        read_datasets_op = read_datasets(
            data=dsl.Collected(single_dag_op.output))
        read_datasets_op.set_caching_options(False)
    
    return dsl.Collected(read_single_dataset_generate_model_op.outputs["results"])


@dsl.pipeline()
def collected_artifact_pipeline():
    model_ids = 's1,s2,s3'
    model_chars = 'x,y,z'
    dag = collecting_artifacts(model_ids=model_ids, model_chars=model_chars)
    dag.set_caching_options(False)
    read_files_op = read_models(models=dag.output)
    read_files_op.set_caching_options(False)


if __name__ == '__main__':
    client = kfp.Client()
    run = client.create_run_from_pipeline_func(
        collected_artifact_pipeline,
        arguments={},
        enable_caching=False,
    )
