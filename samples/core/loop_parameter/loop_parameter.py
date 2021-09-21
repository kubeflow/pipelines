from kfp import components
from kfp import dsl
from typing import List


@components.create_component_from_func
def print_op(text: str) -> str:
    print(text)
    return text


@components.create_component_from_func
def concat_op(a: str, b: str) -> str:
    print(a + b)
    return a + b


@components.create_component_from_func
def generate_op() -> str:
    import json
    return json.dumps([{'a': i, 'b': i * 10} for i in range(1, 5)])


@dsl.pipeline(name='pipeline-with-loop-parameter')
def my_pipeline(greeting:str='this is a test for looping through parameters'):
    print_task = print_op(text=greeting)

    generate_task = generate_op()
    with dsl.ParallelFor(generate_task.output) as item:
        concat_task = concat_op(a=item.a, b=item.b)
        concat_task.after(print_task)
        print_task_2 = print_op(concat_task.output)
