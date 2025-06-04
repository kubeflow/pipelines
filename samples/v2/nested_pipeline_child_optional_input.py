
from kfp import compiler
from kfp import dsl


@dsl.component()
def component_a(componentInput1: str = None):
    if componentInput1 != 'Input 1 - nested pipeline':
        raise ValueError(f"componentInput2 should be 'Input 1 - nested pipeline' but is {componentInput1}")


@dsl.component()
def component_b(componentInput2: str = None):
    if componentInput2 != 'Input 2 -nested pipeline':
        raise ValueError(f"componentInput2 should be 'Input 2 - nested pipeline' but is {componentInput2}")

@dsl.pipeline()
def nested_pipeline(nestedInput1: str = 'Input 1 - nested pipeline',  nestedInput2: str = 'Input 2 -nested pipeline'):
    component_a(componentInput1=nestedInput1).set_caching_options(False)
    component_b(componentInput2=nestedInput2).set_caching_options(False)

@dsl.pipeline()
def pipeline():
    nested_pipeline().set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(pipeline_func=pipeline, package_path=__file__.replace('.py', '_compiled.yaml'))

