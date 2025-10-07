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

"""A pipeline using workspace functionality."""
from kfp import dsl, compiler


@dsl.component
def write_to_workspace(workspace_path: str) -> str:
    """Write a file to the workspace."""   
    import os
    
    # Create a file in the workspace
    file_path = os.path.join(workspace_path, "data", "test_file.txt")
    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    
    with open(file_path, "w") as f:
        f.write("Hello from workspace!")
    
    print(f"Wrote file to: {file_path}")
    return file_path


@dsl.component
def read_from_workspace(file_path: str) -> str:
    """Read a file from the workspace using the provided file path."""    
    import os
    
    if os.path.exists(file_path):
        with open(file_path, "r") as f:
            content = f.read()
        print(f"Read content from: {file_path}")
        print(f"Content: {content}")
        assert content == "Hello from workspace!"
        return content
    else:
        print(f"File not found at: {file_path}")
        return "File not found"


@dsl.pipeline(
    name="pipeline-with-workspace",
    description="A pipeline that demonstrates workspace functionality",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='1Gi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={'storageClassName': 'standard'}
            )
        ),
    ),
)
def pipeline_with_workspace() -> str:
    """A pipeline using workspace functionality with write and read components."""
    
    # Write to workspace
    write_task = write_to_workspace(
        workspace_path=dsl.WORKSPACE_PATH_PLACEHOLDER
    )
    
    # Read from workspace
    read_task = read_from_workspace(
        file_path=write_task.output
    )
    
    return read_task.output


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_workspace,
        package_path=__file__.replace('.py', '.yaml'))