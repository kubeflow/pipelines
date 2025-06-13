from typing import Optional

from kfp import compiler, dsl


@dsl.component()
def component_a(componentInput1: str = 'Input 1 - component'):
    print(componentInput1)


@dsl.component()
def component_b(componentInput2: str = 'Input 2 -component'):
    print(componentInput2)

@dsl.pipeline()
def nested_pipeline(nestedInput1: str = 'Input 1 - nested pipeline',  nestedInput2: str = 'Input 2 -nested pipeline'):
    component_a(componentInput1=nestedInput1).set_caching_options(False)
    component_b(componentInput2=nestedInput2).set_caching_options(False)

@dsl.pipeline()
def pipeline(input1: str = 'Input 1 - pipeline', input2: str = 'Input 2 - pipeline'):
    nested_pipeline(nestedInput1=input1, nestedInput2=input2).set_caching_options(False)


if __name__ == "__main__":
    compiler.Compiler().compile(pipeline_func=pipeline, package_path=__file__.replace(".py", "_compiled.yaml"))

