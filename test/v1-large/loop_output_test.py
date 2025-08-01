from kfp import components
from kfp import dsl

from conftest import runner


@components.create_component_from_func
def args_generator_op() -> str:
    return '[1.1, 1.2, 1.3]'


@components.create_component_from_func
def print_op(s: float):
    print(s)


@dsl.pipeline(name='pipeline-with-loop-output')
def my_pipeline():
    args_generator = args_generator_op()
    with dsl.ParallelFor(args_generator.output) as item:
        print_op(item)

def test_loop_output_pipeline():
    runner(my_pipeline)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
