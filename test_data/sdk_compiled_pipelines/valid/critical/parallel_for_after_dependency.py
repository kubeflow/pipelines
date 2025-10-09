from kfp import Client, dsl


@dsl.component
def print_op(message: str) -> str:
    print(message)
    return message


@dsl.pipeline()
def loop_with_after_dependency_set():
    with dsl.ParallelFor([1, 2, 3]):
        one = print_op(message='foo')
    # Ensure that the dependecy is set downstream for all loop iterations
    two = print_op(message='bar').after(one)
    three = print_op(message='baz').after(one)


if __name__ == '__main__':
    client = Client()
    run = client.create_run_from_pipeline_func(loop_with_after_dependency_set)
