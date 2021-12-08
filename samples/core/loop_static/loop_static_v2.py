import os
from kfp.v2 import dsl
from typing import List

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


_DEFAULT_LOOP_ARGUMENTS = [{'a': '1', 'b': '2'}, {'a': '10', 'b': '20'}]


@dsl.pipeline(name='pipeline-with-loop-static')
def my_pipeline(
    static_loop_arguments: List[dict] = _DEFAULT_LOOP_ARGUMENTS,
    greeting: str = 'this is a test for looping through parameters',
):
    print_task = print_op(text=greeting)

    with dsl.ParallelFor(static_loop_arguments) as item:
        concat_task = concat_op(a=item.a, b=item.b)
        concat_task.after(print_task)
        print_task_2 = print_op(text=concat_task.output)
