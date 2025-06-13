
from kfp import compiler
from kfp import dsl


@dsl.component()
def component_a(componentInput1: str = 'Input 1 - component'):
    print(componentInput1)


@dsl.component()
def component_b(componentInput2: str = 'Input 2 -component'):
    print(componentInput2)

@dsl.pipeline()
def nested_pipeline():
    component_a().set_caching_options(False)
    component_b().set_caching_options(False)

@dsl.pipeline()
def pipeline():
    nested_pipeline().set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(pipeline_func=pipeline, package_path=__file__.replace('.py', '_compiled.yaml'))

