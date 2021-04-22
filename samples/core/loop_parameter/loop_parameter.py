from kfp import components
from kfp import dsl
from typing import List


@components.create_component_from_func
def print_op(text: str) -> str:
    print(text)
    return text


@components.create_component_from_func
def sum_op(a: float, b: float) -> float:
    print(a + b)
    return a + b


@components.create_component_from_func
def generate_op() -> list:
    return [{'a': i, 'b': i * 10} for i in range(1, 5)]


@dsl.pipeline(name='pipeline-with-loop-parameter')
def my_pipeline(greeting='this is a test for looping through parameters'):
    print_task = print_op(text=greeting)

    generate_task = generate_op()
    with dsl.ParallelFor(generate_task.output) as item:
        sum_task = sum_op(a=item.a, b=item.b)
        sum_task.after(print_task)
        print_task_2 = print_op(sum_task.output.ignore_type())
