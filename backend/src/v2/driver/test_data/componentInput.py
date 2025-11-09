import functools
import os

from kfp import dsl
from kfp.dsl import Input, Output, Dataset, Model, Artifact

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@dsl.component
def process_inputs(
        name: str,
        number: int,
        threshold: float,
        active: bool,
        a_runtime_string: str,
        a_runtime_number: int,
        a_runtime_bool: bool,
        output_text: Output[Dataset]
) -> None:
    with open(output_text.path, 'w') as f:
        f.write(f"[{name}, {number}, {threshold}, {active}]")

    assert name == "default_name"
    assert number == 42
    assert threshold == 0.5
    assert active == True
    assert a_runtime_string == "foo"
    assert a_runtime_number == 10
    assert a_runtime_bool == True

@dsl.component
def analyze_inputs(input_text: Input[Dataset]):
    with open(input_text.path, 'r') as f:
        data = f.read()
    assert data == "[default_name, 42, 0.5, True]"

@dsl.pipeline
def primary_pipeline(
        name_in: str = "default_name",
        number_in: int = 42,
        threshold_in: float = 0.5,
        active_in: bool = True,
):
    process_inputs_task = process_inputs(
        name=name_in,
        number=number_in,
        threshold=threshold_in,
        active=active_in,
        a_runtime_string="foo",
        a_runtime_number=10,
        a_runtime_bool=True,
    )
    analyze_inputs(input_text=process_inputs_task.outputs['output_text'])


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
