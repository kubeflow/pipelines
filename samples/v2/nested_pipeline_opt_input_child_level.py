
from kfp import compiler
from kfp import dsl


@dsl.component()
def component_a_str(componentInputStr: str = None):
    if componentInputStr != 'Input - pipeline':
        raise ValueError(f"componentInputStr should be 'Input - pipeline' but is {componentInputStr}")


@dsl.component()
def component_b_str(componentInputStr: str = None):
    if componentInputStr != 'Input 2 - nested pipeline':
        raise ValueError(f"componentInputStr should be 'Input 2 - nested pipeline' but is {componentInputStr}")

@dsl.component()
def component_a_int(componentInputInt: int = None):
    if componentInputInt != 1:
        raise ValueError(f"componentInputInt should be 1 but is {componentInputInt}")

@dsl.component()
def component_b_int(componentInputInt: int = None):
    if componentInputInt != 0:
        raise ValueError(f"componentInputInt should be 0 but is {componentInputInt}")

@dsl.component()
def component_a_bool(componentInputBool: bool = None):
    if componentInputBool != True:
        raise ValueError(f"componentInputBool should be True but is {componentInputBool}")

@dsl.component()
def component_b_bool(componentInputBool: bool = None):
    if componentInputBool != False:
        raise ValueError(f"componentInputBool should be False but is {componentInputBool}")

@dsl.pipeline()
def nested_pipeline(nestedInputStr1: str = 'Input 1 - nested pipeline',  nestedInputStr2: str = 'Input 2 - nested pipeline',
                        nestedInputInt1: int = 0, nestedInputInt2: int = 0,
                        nestedInputBool1: bool = False, nestedInputBool2: bool = False):
    component_a_str(componentInputStr=nestedInputStr1).set_caching_options(False)
    component_b_str(componentInputStr=nestedInputStr2).set_caching_options(False)

    component_a_int(componentInputInt=nestedInputInt1).set_caching_options(False)
    component_b_int(componentInputInt=nestedInputInt2).set_caching_options(False)

    component_a_bool(componentInputBool=nestedInputBool1).set_caching_options(False)
    component_b_bool(componentInputBool=nestedInputBool2).set_caching_options(False)


@dsl.pipeline()
def nested_pipeline_opt_input_child_level():
    # validate that input value overrides default value, and that when input is not provided, default is used.
    nested_pipeline(nestedInputStr1='Input - pipeline', nestedInputInt1=1, nestedInputBool1=True).set_caching_options(False)



if __name__ == '__main__':
    compiler.Compiler().compile(pipeline_func=nested_pipeline_opt_input_child_level, package_path=__file__.replace('.py', '_compiled.yaml'))

