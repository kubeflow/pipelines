from kfp import compiler, dsl


@dsl.component
def print_op(text: str) -> str:
    print(text)
    return text


@dsl.component
def concat_op(a: str, b: str) -> str:
    print(a + b)
    return a + b


@dsl.pipeline(name='pipeline-with-loop-static')
def my_pipeline(
    greeting: str = 'this is a test for looping through parameters',):
    import json
    print_task = print_op(text=greeting)
    static_loop_arguments = [json.dumps({'a': '1', 'b': '2'}), 
                             json.dumps({'a': '10', 'b': '20'})]

    with dsl.ParallelFor(static_loop_arguments) as item:
        concat_task = concat_op(a=item.a, b=item.b)
        print_task_2 = print_op(text=concat_task.output)

if __name__ == '__main__':
    compiler.Compiler().compile(my_pipeline, __file__ + '.yaml')
