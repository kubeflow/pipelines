from kfp import Client
from kfp import dsl
from kfp.dsl import Artifact, Input, Output


@dsl.component
def split_input(input: str) -> list:
    return input.split(',')


@dsl.component
def create_file(file: Output[Artifact], content: str):
    with open(file.path, 'w') as f:
        f.write(content)


@dsl.component
def read_file(file: Input[Artifact]) -> str:
    with open(file.path, 'r') as f:
        print(f.read())
    return file.path


@dsl.pipeline()
def loop_consume_upstream():
    model_ids_split_op = split_input(input='component1,component2,component3')
    model_ids_split_op.set_caching_options(False)

    with dsl.ParallelFor(model_ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)
        create_file_op.set_caching_options(False)
        # Consume the output from a op in the loop iteration DAG context
        read_file_op = read_file(file=create_file_op.outputs['file'])
        read_file_op.set_caching_options(False)


if __name__ == '__main__':
    client = Client()
    run = client.create_run_from_pipeline_func(loop_consume_upstream)
