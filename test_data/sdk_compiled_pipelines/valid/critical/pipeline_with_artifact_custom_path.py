from kfp.dsl import Output
from kfp.dsl.types.artifact_types import Artifact
from kfp.v2 import dsl


@dsl.component
def generate_artifact() -> list:
    return [1, 2, 3, 4]

@dsl.component
def validate_artifact_path(exp_path: str, input_list: Output[Artifact]) -> bool:
    if input_list.path is not exp_path:
        raise ValueError(f"File uri is {input_list.path} but should be {exp_path}.")

@dsl.pipeline
def pipeline_with_custom_path_artifact():
    # Generate artifact, and set its custom path.
    output_artifact_task = generate_artifact()
    output_artifact_task.output.set_path('/etc/test/file/path')

    # Validate generated artifact's path.
    validate_artifact_task = validate_artifact_path(path='/etc/test/file/path', input_list=output_artifact_task.output)
