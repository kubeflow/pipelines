from kfp import dsl


@dsl.component
def my_component(string: str, number: int) -> str:
    return string


@dsl.pipeline(name='pipeline')
def my_pipeline(string: str, number: int):
    op1 = my_component(string=string, number=number)
    op2 = my_component(string=op1.output, number=number)
    my_component('string', 1)


my_pipeline('string', 1)
