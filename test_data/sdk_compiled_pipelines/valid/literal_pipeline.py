"""Pipeline used to validate successful compilation of Literal input parameters.

This test ensures that pipelines using typing.Literal inputs compile correctly
and produce stable IR output.
"""

import typing
from kfp import dsl


@dsl.component
def literal_component(x: typing.Literal['a', 'b']) -> str:
    return x


@dsl.pipeline(name='literal-pipeline')
def literal_pipeline():
    literal_component(x='a')


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=literal_pipeline,
        package_path=__file__.replace('.py', '.yaml'),
    )
