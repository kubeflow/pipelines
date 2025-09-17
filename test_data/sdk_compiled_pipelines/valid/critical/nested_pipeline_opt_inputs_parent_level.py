from typing import Optional

from kfp import compiler, dsl


@dsl.component()
def component_nil_str_default(componentInput: str = None):
    if componentInput != 'Input - parent pipeline':
        raise ValueError(f"componentInput should be 'Input - parent pipeline' but is {componentInput}")

@dsl.component()
def component_str_default(componentInput: str = 'Input - component'):
    if componentInput != 'Input - parent pipeline':
        raise ValueError(f"componentInput should be 'Input - parent pipeline' but is {componentInput}")

@dsl.component()
def component_nil_int_default(componentInput: int = None):
    if componentInput != 1:
        raise ValueError(f"componentInput should be 1 but is {componentInput}")

@dsl.component()
def component_int_default(componentInput: int = 0):
    if componentInput != 1:
        raise ValueError(f"componentInput should be 1 but is {componentInput}")

@dsl.component()
def component_nil_bool_default(componentInput: bool = None):
    if componentInput != True:
        raise ValueError(f"componentInput should be True but is {componentInput}")

@dsl.component()
def component_bool_default(componentInput: bool = False):
    if componentInput != True:
        raise ValueError(f"componentInput should be True but is {componentInput}")

@dsl.pipeline()
def nested_pipeline_non_nil_defaults(nestedInputStr: str = 'Input - nested pipeline', nestedInputInt: int = 0, nestedInputBool: bool = False):
    component_str_default(componentInput=nestedInputStr).set_caching_options(False)
    component_int_default(componentInput=nestedInputInt).set_caching_options(False)
    component_bool_default(componentInput=nestedInputBool).set_caching_options(False)

@dsl.pipeline()
def nested_pipeline_nil_defaults(nestedInputStr: str = 'Input - nested pipeline', nestedInputInt: int = None, nestedInputBool: bool = None):
    component_nil_str_default(componentInput=nestedInputStr).set_caching_options(False)
    component_nil_int_default(componentInput=nestedInputInt).set_caching_options(False)
    component_nil_bool_default(componentInput=nestedInputBool).set_caching_options(False)


@dsl.pipeline()
def nested_pipeline_opt_inputs_parent_level(inputStr: str = 'Input - parent pipeline', inputInt: int = 1, inputBool: bool = True):
    # verifies that the parent pipeline input overrides both nested pipeline-level and component-level default values.
    nested_pipeline_non_nil_defaults(nestedInputStr=inputStr, nestedInputInt=inputInt, nestedInputBool=inputBool).set_caching_options(False)
    # verifies that the parent pipeline input overrides both nested pipeline-level & component-level nil default input values.
    nested_pipeline_nil_defaults(nestedInputStr=inputStr, nestedInputInt=inputInt, nestedInputBool=inputBool).set_caching_options(False)

if __name__ == "__main__":
    compiler.Compiler().compile(pipeline_func=nested_pipeline_opt_inputs_parent_level, package_path=__file__.replace(".py", "_compiled.yaml"))

