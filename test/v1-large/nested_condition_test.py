from kfp import components
from kfp import dsl

from conftest import runner


@components.create_component_from_func
def flip_coin_op() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    return result


@components.create_component_from_func
def print_op(msg: str):
    """Print a message."""
    print(msg)


@dsl.pipeline(name='nested-conditions-pipeline')
def my_pipeline():
    flip1 = flip_coin_op()
    print_op(flip1.output)
    flip2 = flip_coin_op()
    print_op(flip2.output)

    with dsl.Condition(flip1.output != 'no-such-result'):  # always true
        flip3 = flip_coin_op()
        print_op(flip3.output)

        with dsl.Condition(flip2.output == flip3.output):
            flip4 = flip_coin_op()
            print_op(flip4.output)


def test_nested_condition_pipeline():
    runner(my_pipeline)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
