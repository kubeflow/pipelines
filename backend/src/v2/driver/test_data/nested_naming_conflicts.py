import functools

from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component,
    pipeline
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def a(situation: str, output_dataset: Output[Dataset]):
    with open(output_dataset.path, "w") as f:
        f.write(situation)

@component
def b(input_dataset: Input[Dataset], output_artifact_b: Output[Artifact]):
    with open(input_dataset.path, "r") as f:
        data = f.read()

    if data == "hurricane":
        analysis = "very_bad"
    elif data == "sunny":
        analysis = "very_good"
    else:
        analysis = "not_bad"

    with open(output_artifact_b.path, "w") as f:
        f.write(analysis)

@component
def c(artifact: Input[Artifact], output_artifact_c: Output[Artifact]):
    with open(artifact.path, "r") as f:
        data = f.read()
    assert data == "very_bad"
    with open(output_artifact_c.path, "w") as f:
        f.write(f'done_analyzing')

@component
def verify(verify_input: Input[Artifact]):
    with open(verify_input.path, "r") as f:
        data = f.read()
    assert data == "done_analyzing"

@pipeline
def pipeline_c(input_dataset_a: Input[Dataset], input_dataset_b: Input[Dataset]) -> Artifact:
    a_task = a(situation="hurricane")
    b_task = b(input_dataset=a_task.outputs["output_dataset"])
    c_task = c(artifact=b_task.outputs["output_artifact_b"])
    return c_task.outputs["output_artifact_c"]


@pipeline
def pipeline_b(input_dataset: Input[Dataset]) -> Artifact:
    a_task = a(situation="raining")
    b_task = b(input_dataset=input_dataset)
    pipeline_c_op = pipeline_c(
        input_dataset_a=a_task.outputs["output_dataset"],
        input_dataset_b=b_task.outputs["output_artifact_b"],
    )
    return pipeline_c_op.output

@pipeline
def pipeline_a():
    a_task = a(situation="sunny")
    nested_pipeline_op = pipeline_b(input_dataset=a_task.outputs["output_dataset"])
    verify(verify_input=nested_pipeline_op.output)
if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=pipeline_a,
        package_path=__file__.replace('.py', '.yaml'))
