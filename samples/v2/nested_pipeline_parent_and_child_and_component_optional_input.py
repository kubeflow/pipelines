from typing import Optional

from kfp import compiler, dsl


@dsl.component()
def component_nil_default(componentInput: str = None):
    if componentInput != 'Input - parent pipeline':
        raise ValueError(f"componentInput should be 'Input - parent pipeline' but is {componentInput}")

@dsl.component()
def component_non_nil_default(componentInput: str = 'Input - component'):
    if componentInput != 'Input - parent pipeline':
        raise ValueError(f"componentInput should be 'Input - parent pipeline' but is {componentInput}")

@dsl.pipeline()
def nested_pipeline_non_nil_defaults(nestedInput: str = 'Input - nested pipeline'):
    component_nil_default(componentInput=nestedInput).set_caching_options(False)

@dsl.pipeline()
def nested_pipeline_nil_defaults(nestedInput: str = None):
    component_nil_default(componentInput=nestedInput).set_caching_options(False)

@dsl.pipeline()
def nested_pipeline_and_component_non_nil_defaults(nestedInput: str = 'Input - nested pipeline'):
    component_non_nil_default(componentInput=nestedInput).set_caching_options(False)

@dsl.pipeline()
def pipeline(input: str = 'Input - parent pipeline'):
    # verifies that the parent pipeline input overrides a nested pipeline-level default input value.
    nested_pipeline_non_nil_defaults(nestedInput=input).set_caching_options(False)
    # verifies that the parent pipeline input overrides a nested pipeline-level nil default input value.
    nested_pipeline_non_nil_defaults(nestedInput=input).set_caching_options(False)
    # verifies that the parent pipeline input overrides both nested pipeline-level and component-level default values.
    nested_pipeline_and_component_non_nil_defaults(nestedInput=input).set_caching_options(False)

if __name__ == "__main__":
    compiler.Compiler().compile(pipeline_func=pipeline, package_path=__file__.replace(".py", "_compiled.yaml"))

