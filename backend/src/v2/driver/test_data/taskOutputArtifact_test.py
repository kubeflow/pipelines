import functools

from kubernetes import client
import kfp
from kfp import dsl
from kfp.dsl import (
    Input,
    Output,
    Artifact,
    Dataset,
    component
)

base_image="quay.io/opendatahub/ds-pipelines-ci-executor-image:v1.0"
dsl.component = functools.partial(dsl.component, base_image=base_image)

@component
def create_dataset(output_dataset: Output[Dataset]):
    with open(output_dataset.path, "w") as f:
        f.write('hurricane')
    output_dataset.metadata["category"] = 5
    output_dataset.metadata["description"] = "A simple dataset"

@component
def process_dataset(input_dataset: Input[Dataset], output_artifact: Output[Artifact]):
    with open(input_dataset.path, "r") as f:
        data = f.read()
    assert data == "hurricane"
    with open(output_artifact.path, "w") as f:
        f.write(f'very_bad')

@component
def analyze_artifact(data_input: Input[Artifact], output_artifact: Output[Artifact]):
    with open(data_input.path, "r") as f:
        data = f.read()
    assert data == "very_bad"
    with open(output_artifact.path, "w") as f:
        f.write(f'done_analyzing')

@dsl.pipeline
def primary_pipeline():
    dataset_op = create_dataset()
    processed = process_dataset(input_dataset=dataset_op.outputs["output_dataset"])
    analyze_artifact(data_input=processed.outputs["output_artifact"])

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
