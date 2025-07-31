import kfp
from kfp import dsl
import kfp.components as comp

from conftest import runner


@comp.create_component_from_func
def random_failure_op(exit_codes):
    """A component that fails randomly."""
    import random
    import sys

    exit_code = int(random.choice(exit_codes.split(",")))
    print(exit_code)
    sys.exit(exit_code)


@dsl.pipeline(
    name='retry-random-failures',
    description='The pipeline includes two steps which fail randomly. It shows how to use ContainerOp(...).set_retry(...).'
)
def retry_sample_pipeline():
    op1 = random_failure_op('0,1,2,3').set_retry(10)
    op2 = random_failure_op('0,1').set_retry(5)


def test_retry_pipeline():
    runner(retry_sample_pipeline)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(retry_sample_pipeline, __file__ + '.yaml')
