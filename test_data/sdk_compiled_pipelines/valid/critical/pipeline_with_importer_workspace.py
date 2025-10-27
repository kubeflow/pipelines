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
from kfp.dsl import importer
import os

@dsl.component
def train(
    dataset: dsl.Input[dsl.Dataset]
) -> NamedTuple('Outputs', [
    ('scalar', str),
    ('message', str),
]):
    """Dummy Training step."""
    with open(dataset.path) as f:
        data = f.read()
    print('Dataset:', data)

    scalar = '123'
    message = f'My model trained using data: {data}'

    from collections import namedtuple
    output = namedtuple('Outputs', ['scalar', 'message'])
    return output(scalar, message)

@dsl.component
def read_dir(data: dsl.Input[dsl.Dataset]) -> str:
    """Walk the directory and return a summary of file names."""
    import os
    path = data.path
    
    if not os.path.exists(path):
        print(f"ERROR: Path does not exist: {path}")
        return "ERROR: Path not found"
    
    if os.path.isdir(path):
        names = []
        for root, _, files in os.walk(path):
            for name in files:
                names.append(os.path.relpath(os.path.join(root, name), path))
        names.sort()
        result = ",".join(names) if names else "EMPTY_DIRECTORY"
        print(f"Found {len(names)} files: {result}")
        return result
    elif os.path.isfile(path):
        print(f"Path is a single file: {path}")
        return os.path.basename(path)
    
    return "ERROR: Unknown path type"

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
def pipeline_with_importer_workspace(    
    dataset_dir: str = 'gs://ml-pipeline-playground',
) -> NamedTuple('Outputs', [('train_result', str), ('dir_result', str)]):
    """Test pipeline for importer with download_to_workspace feature."""
    
    # Import a file artifact from a constant URI    
    importer1 = importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class=dsl.Dataset,
        reimport=False,
        metadata={'key': 'value'},
        download_to_workspace=True,
    )
    train_task = train(dataset=importer1.output)

    # Import a directory artifact by URI
    dir_import = importer(
        artifact_uri=dataset_dir,
        artifact_class=dsl.Dataset,
        reimport=True,
        download_to_workspace=True,
        metadata={'source': 'directory'},
    )
    dir_task = read_dir(data=dir_import.output)
    
    # Return outputs for validation
    Outputs = NamedTuple('Outputs', [('train_result', str), ('dir_result', str)])
    return Outputs(train_task.outputs['scalar'], dir_task.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_importer_workspace,
        package_path=__file__.replace('.py', '.yaml'))