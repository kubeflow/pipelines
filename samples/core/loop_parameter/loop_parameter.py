from kfp import compiler, dsl


@dsl.component
def print_op(text: str) -> str:
    print(text)
    return text


@dsl.component
def concat_op(a: str, b: str) -> str:
    print(a + b)
    return a + b


@dsl.component
def generate_op() -> list:
    return [{'a': i, 'b': i * 10} for i in range(1, 5)]

@dsl.pipeline(name='pipeline-with-loop-parameter')
def my_pipeline(
        greeting: str = 'this is a test for looping through parameters'):
    print_task = print_op(text=greeting)

    generate_task = generate_op()
    with dsl.ParallelFor(generate_task.output) as item:
        concat_task = concat_op(a=item.a, b=item.b)
        print_task_2 = print_op(text=concat_task.output)

if __name__ == '__main__':
    compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
