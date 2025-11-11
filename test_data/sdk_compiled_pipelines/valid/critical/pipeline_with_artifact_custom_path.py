from kfp.dsl import Output, Input
from kfp.dsl.types.artifact_types import Dataset
from kfp.v2 import dsl, compiler

@dsl.component
def component_output_artifact(out_dataset: Output[Dataset]):
    out_dataset.set_path('/tmp/out_dataset')

    with open(out_dataset.path, 'w') as f:
        f.write('Hello, World!')

    if out_dataset.path != '/tmp/out_dataset':
        raise ValueError(f"File path is {out_dataset.path} but should be '/tmp/out_dataset'.")
    if out_dataset.custom_path != '/tmp/out_dataset':
        raise ValueError(f"File uri is {out_dataset.custom_path} but should be '/tmp/out_dataset'.")

@dsl.component
def validate_input_artifact(in_dataset: Input[Dataset]) -> bool:
    if in_dataset is None:
        raise ValueError("Input artifact is None.")

    with open(in_dataset.path, 'r') as f:
        for line in f:
            if line != 'Hello, World!':
                raise ValueError(f"File content is {line} but should be 'Hello, World!'.")
    return True

@dsl.pipeline(description='pipeline with artifact with custom path')
def pipeline_with_custom_path_artifact():
    # Generate an artifact, set a custom path and validate the path.
    task_output_artifact = component_output_artifact().set_caching_options(False)

    # Take the output artifact from above as input, and validate as non-nil.
    task_validate_input_artifact = validate_input_artifact(in_dataset=task_output_artifact.outputs['out_dataset'])

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_custom_path_artifact,
        package_path=__file__.replace('.py', '.yaml'))

