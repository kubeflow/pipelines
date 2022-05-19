import functools


def args_as_ints(func):

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        args = [int(x) for x in args]
        kwargs = dict((k, int(v)) for k, v in kwargs.items())
        return func(*args, **kwargs)

    return wrapper


@args_as_ints
def funny_function(x, y, z=3):
    """Computes x*y + 2*z"""
    return x * y + 2 * z


import inspect

print(inspect.signature(funny_function))

from kfp import dsl


@dsl.component
def my_component(string: str, number: int) -> str:
    return string


@dsl.pipeline(name='pipeline')
def my_pipeline(string: str, number: int):
    op1 = my_component(string=string, number=number)
    op2 = my_component(string=op1.output, number=number)


print(inspect.signature(my_component))
