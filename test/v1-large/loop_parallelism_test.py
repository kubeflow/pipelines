from kfp import dsl
import kfp

from conftest import runner


@kfp.components.create_component_from_func
def print_op(s: str):
    print(s)

@dsl.pipeline(name='my-pipeline')
def pipeline():
    loop_args = [{'A_a': 1, 'B_b': 2}, {'A_a': 10, 'B_b': 20}]
    with dsl.ParallelFor(loop_args, parallelism=10) as item:
        print_op(item)
        print_op(item.A_a)
        print_op(item.B_b)

def test_loop_parallelism_pipeline():
    runner(pipeline)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline, __file__ + '.yaml')
