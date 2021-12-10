import os
from kfp.v2 import dsl

# In tests, we install a KFP package from the PR under test. Users should not
# normally need to specify `kfp_package_path` in their component definitions.
_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def print_op(text: str) -> str:
    print(text)
    return text


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def concat_op(a: str, b: str) -> str:
    print(a + b)
    return a + b


@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def generate_op() -> str:
    import json
    return json.dumps([{'a': i, 'b': i * 10} for i in range(1, 5)])


@dsl.pipeline(name='pipeline-with-loop-parameter')
def my_pipeline(
        greeting: str = 'this is a test for looping through parameters'):
    print_task = print_op(text=greeting)

    generate_task = generate_op()
    with dsl.ParallelFor(generate_task.output) as item:
        concat_task = concat_op(a=item.a, b=item.b)
        concat_task.after(print_task)
        print_task_2 = print_op(text=concat_task.output)
