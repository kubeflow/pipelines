"""Pipeline used to validate successful compilation of Literal input parameters.

This test ensures that pipelines using typing.Literal inputs compile correctly
and produce stable IR output.
"""

from typing import Literal
from kfp import dsl


@dsl.component
def literal_component(literal_param: Literal['a', 'b']):
    print(literal_param)


@dsl.pipeline(name='literal-pipeline')
def literal_pipeline():
    literal_component(literal_param='a')


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=literal_pipeline,
        package_path=__file__.replace('.py', '.yaml'),
    )
