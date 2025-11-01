from typing import List

from kfp.dsl import Output, Input
from kfp.dsl.types.artifact_types import Artifact, Dataset
from kfp.v2 import dsl, compiler


def validate_custom_artifact_path(num: int, out_dataset: Output[Dataset]):
    with open(out_dataset.path, 'w') as f:
        f.write(str(2 * num))
    out_dataset._set_path('/etc/test/file/path')

    if out_dataset.path != '/etc/test/file/path':
        raise ValueError(f"File path is {out_dataset.path} but should be '/etc/test/file/path'.")
    if out_dataset.custom_path != '/etc/test/file/path':
        raise ValueError(f"File uri is {out_dataset.custom_path} but should be '/etc/test/file/path'.")

@dsl.pipeline(description='custom path task')
def pipeline_with_custom_path_artifact():
    # Generate an artifact, set a custom path and validate the path.
    task = validate_custom_artifact_path(num=1).set_caching_options(False)

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_custom_path_artifact,
        package_path=__file__.replace('.py', '.yaml'))

