from kfp import components
from kfp import dsl

from conftest import runner


def flip_coin(force_flip_result: str = '') -> str:
    """Flip a coin and output heads or tails randomly."""
    if force_flip_result:
        return force_flip_result
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    return result


def print_msg(msg: str):
    """Print a message."""
    print(msg)


flip_coin_op = components.create_component_from_func(flip_coin)

print_op = components.create_component_from_func(print_msg)


@dsl.pipeline(name='single-condition-pipeline')
def my_pipeline(text: str = 'condition test', force_flip_result: str = ''):
    flip1 = flip_coin_op(force_flip_result)
    print_op(flip1.output)

    with dsl.Condition(flip1.output == 'heads'):
        flip2 = flip_coin_op()
        print_op(flip2.output)
        print_op(text)

def test_condition_pipeline():
    runner(my_pipeline)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
