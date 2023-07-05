from kfp import dsl
from kfp import compiler
from src.my_component import add

@dsl.pipeline
def addition_pipeline(x: int, y: int) -> int:
    task1 = add(a=x, b=y)
    task2 = add(a=task1.output, b=x)
    return task2.output

compiler.Compiler().compile(addition_pipeline, 'pipeline.yaml')