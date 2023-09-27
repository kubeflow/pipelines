from kfp import dsl


@dsl.component
def identity(string: str) -> str:
    return string


@dsl.pipeline
def my_pipeline(string: str = 'string'):
    op1 = identity(string=string)
    op2 = identity(string=op1.output)
