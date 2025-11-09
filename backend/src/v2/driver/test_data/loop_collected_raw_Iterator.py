import functools
from typing import List

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
        f.write('cat')
    output_dataset.metadata["item_count"] = 5
    output_dataset.metadata["description"] = "A simple dataset with integers"

@component
def process_dataset(
        model_id_in: str,
        input_dataset: Input[Dataset],
        output_artifact: Output[Artifact],
):
    with open(input_dataset.path, "r") as f:
        data = f.read()
    with open(output_artifact.path, "w") as f:
        data_out = f"{data}-{model_id_in}"
        f.write(data_out)
        print(data_out)
    output_artifact.metadata["model_id"] = model_id_in

@component
def analyze_artifact(analyze_artifact_input: Input[Artifact], analyze_output_artifact: Output[Artifact]):
    with open(analyze_artifact_input.path, "r") as f:
        data = f.read()
    with open(analyze_output_artifact.path, "w") as f:
        f.write(f'{{"values": {data}}}')

@component
def analyze_artifact_list(artifact_list_input: List[Artifact]):
    expected_values = ['cat-1', 'cat-2', 'cat-3']
    expected_metadata = ['1', '2', '3']
    actual_values = []
    actual_metadata = []
    for artifact in artifact_list_input:
        with open(artifact.path, "r") as f:
            data = f.read()
            actual_values.append(data)
            actual_metadata.append(artifact.metadata["model_id"])

    print("actual_values: ", actual_values)
    print("actual_metadata: ", actual_metadata)
    print("expected_values: ", expected_values)
    print("expected_metadata: ", expected_metadata)
    assert sorted(actual_values) == sorted(expected_values)
    assert sorted(actual_metadata) == sorted(expected_metadata)

@dsl.pipeline
def secondary_pipeline() -> List[Artifact]:
    create_dataset_task = create_dataset()
    with dsl.ParallelFor(items=['1', '2', '3']) as model_id:
        process_dataset_task = process_dataset(model_id_in=model_id, input_dataset=create_dataset_task.outputs['output_dataset'])
        analyze_artifact(analyze_artifact_input=process_dataset_task.outputs["output_artifact"])

    # Case one, pass collected result as TaskOutput to analyze artifact
    analyze_artifact_list(artifact_list_input=dsl.Collected(process_dataset_task.outputs["output_artifact"]))

    # Case two, return the collected result for dag.outputs
    return dsl.Collected(process_dataset_task.outputs["output_artifact"])

@dsl.pipeline
def primary_pipeline():
    secondary_pipeline_output = secondary_pipeline()
    analyze_artifact_list(artifact_list_input=secondary_pipeline_output.output)

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
