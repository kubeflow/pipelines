# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""A pipeline that tests the importer with download_to_workspace for file and directory artifacts."""
from kfp import dsl, compiler
from typing import NamedTuple
from kfp.dsl import importer, Output
import os

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest')
def train(
    dataset: dsl.Input[dsl.Dataset]
) -> NamedTuple('Outputs', [
    ('scalar', str),
    ('message', str),
]):
    """Dummy Training step."""
    with open(dataset.path, encoding="utf-8") as f:
        data = f.read()
    print('Dataset:', data)

    scalar = '123'
    message = f'My model trained using data: {data}'

    from collections import namedtuple
    output = namedtuple('Outputs', ['scalar', 'message'])
    return output(scalar, message)

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest')
def read_dir(data: dsl.Input[dsl.Dataset]) -> str:
    """Walk the directory and return a summary of file names."""
    import os
    path = data.path
    
    if not os.path.exists(path):
        raise FileNotFoundError(f"Path does not exist: {path}")
    
    if os.path.isdir(path):
        names = []
        for root, _, files in os.walk(path):
            for name in files:
                names.append(os.path.relpath(os.path.join(root, name), path))
        names.sort()
        result = ",".join(names) if names else "EMPTY_DIRECTORY"
        print(f"Found {len(names)} files: {result}")
        return result
    
    raise ValueError(f"Not a directory: {path}")

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest')
def write_file_artifact(out_ds: Output[dsl.Dataset]):
    import os
    os.makedirs(os.path.dirname(out_ds.path), exist_ok=True)
    with open(out_ds.path, "w", encoding="utf-8") as f:
        f.write("Hello from producer file\n")

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest')
def write_dir_artifact(out_ds: Output[dsl.Dataset]):
    import os
    os.makedirs(out_ds.path, exist_ok=True)
    with open(os.path.join(out_ds.path, "part-000.txt"), "w", encoding="utf-8") as f:
        f.write("First file in directory.\n")
    with open(os.path.join(out_ds.path, "part-001.txt"), "w", encoding="utf-8") as f:
        f.write("Second file in directory.\n")

@dsl.component(base_image='registry.access.redhat.com/ubi9/python-311:latest')
def get_uri(d: dsl.Input[dsl.Dataset]) -> str:
    print(f"Artifact URI: {d.uri}")
    return d.uri

@dsl.pipeline(name="import-stage")
def import_stage(
    file_uri: str,
    dir_uri: str,
) -> NamedTuple('ImportOutputs', [('train_result', str), ('dir_result', str)]):
    """Nested stage that imports by URI and runs consumers."""
    importer1 = importer(
        artifact_uri=file_uri,
        artifact_class=dsl.Dataset,
        reimport=False,
        metadata={
            'key': 'value',
        },
        download_to_workspace=True,
    )

    dir_import = importer(
        artifact_uri=dir_uri,
        artifact_class=dsl.Dataset,
        reimport=True,
        download_to_workspace=True,
        metadata={
            'source': 'directory',
        },
    )

    train_task = train(dataset=importer1.output)
    dir_task = read_dir(data=dir_import.output)

    ImportOutputs = NamedTuple('ImportOutputs', [('train_result', str), ('dir_result', str)])
    return ImportOutputs(train_task.outputs['scalar'], dir_task.output)

@dsl.pipeline(
    name="pipeline-with-importer-workspace",
    description="Importer downloads an artifact into workspace; downstream reads it",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='1Gi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={'storageClassName': 'standard', 'accessModes': ['ReadWriteOnce']}
            ),
        ),
    ),
)
def pipeline_with_importer_workspace() -> NamedTuple('Outputs', [('train_result', str), ('dir_result', str)]):
    """Test pipeline for importer with download_to_workspace feature."""
    
    # Produce a file artifact and compute its runtime URI
    file_writer = write_file_artifact()
    file_uri = get_uri(d=file_writer.outputs["out_ds"])    

    # Produce a directory artifact and compute its runtime URI
    dir_writer = write_dir_artifact()
    dir_uri = get_uri(d=dir_writer.outputs["out_ds"])    
    
    # Import and consume inside a nested sub-pipeline
    stage = import_stage(file_uri=file_uri.output, dir_uri=dir_uri.output)
    
    # Return outputs for validation
    Outputs = NamedTuple('Outputs', [('train_result', str), ('dir_result', str)])
    return Outputs(stage.outputs['train_result'], stage.outputs['dir_result'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_importer_workspace,
        package_path=__file__.replace('.py', '.yaml'))
