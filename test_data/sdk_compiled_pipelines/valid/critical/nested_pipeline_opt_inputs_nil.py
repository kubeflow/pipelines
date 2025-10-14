
from kfp import compiler
from kfp import dsl


@dsl.component()
def component_str(componentInput: str = None):
    if componentInput is not None:
        raise ValueError(f"componentInput should be None but is {componentInput}")

@dsl.component()
def component_int(componentInput: int = None):
    if componentInput is not None:
        raise ValueError(f"componentInput should be None but is {componentInput}")

@dsl.component()
def component_bool(componentInput: bool = None):
    if componentInput is not None:
        raise ValueError(f"componentInput should be None but is {componentInput}")

@dsl.pipeline()
def nested_pipeline(nestedInputStr: str = None, nestedInputInt: int = None, nestedInputBool: bool = None):
    component_str(componentInput=nestedInputStr).set_caching_options(False)
    component_int(componentInput=nestedInputInt).set_caching_options(False)
    component_bool(componentInput=nestedInputBool).set_caching_options(False)


@dsl.pipeline()
def nested_pipeline_opt_inputs_nil():
    nested_pipeline().set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(pipeline_func=nested_pipeline_opt_inputs_nil, package_path=__file__.replace('.py', '_compiled.yaml'))

