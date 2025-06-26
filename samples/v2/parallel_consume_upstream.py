# This pipeline tests the ability to consume outputs from upstream components in
# a loop context as well as having the inputs resolve when set_display_name is
# used within the pipeline.
from kfp import Client
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


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


@dsl.component
def print_input(input: list):
    for item in input:
        print(f'Input item: {item}')


@dsl.pipeline()
def loop_consume_upstream():
    model_ids_split_op = split_input(input='component1,component2,component3')
    model_ids_split_op.set_caching_options(False)
    model_ids_split_op.set_display_name('same display mame')
    with dsl.ParallelFor(model_ids_split_op.output) as model_id:
        create_file_op = create_file(content=model_id)
        create_file_op.set_caching_options(False)
        create_file_op.set_display_name('same display name')
        # Consume the output from a op in the loop iteration DAG context
        read_file_op = read_file(file=create_file_op.outputs['file'])
        read_file_op.set_caching_options(False)
        read_file_op.set_display_name('same display name')

    print_input_op = print_input(input=model_ids_split_op.output)
    print_input_op.set_caching_options(False)
    print_input_op.set_display_name('same display name')


if __name__ == '__main__':
    client = Client()
    run = client.create_run_from_pipeline_func(loop_consume_upstream)
