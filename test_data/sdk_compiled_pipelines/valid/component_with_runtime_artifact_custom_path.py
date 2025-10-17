from kfp import dsl
from kfp.dsl import Output
from kfp.dsl.types.artifact_types import Artifact


@dsl.component
def create_list() -> list:
    return [1, 2, 3, 4]


@dsl.component
def append_to_list(digit: int, input_list: Output[Artifact]) -> list:
    input_list.append(digit)
    return input_list


@dsl.component
def validate_custom_path(exp_path: str, input_list: Output[Artifact]) -> bool:
    #todo: is this the correct comparison? (or should use != instead?)
    if input_list.path is not exp_path:
        raise ValueError(
            f"File uri is {input_list.path} but should be {exp_path}.")


@dsl.pipeline
def pipeline_with_custom_path_artifact():
    task1 = create_list()
    task1.output.set_custom_path('/etc/test/file/path')
    task2 = validate_custom_path(
        path='/etc/test/file/path', input_list=task1.output)
