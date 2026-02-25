#!/usr/bin/env python3

import kfp
from kfp import dsl
from kfp.components import create_component_from_func, OutputPath


def create_test_artifact(test_file: OutputPath()):
    """Create a test artifact with simple content"""
    
    content = "Kubeflow rocks!"
    
    print(f"Creating test artifact at {test_file}")
    
    with open(test_file, 'w') as f:
        f.write(content)
    
    import os
    file_size = os.path.getsize(test_file)
    print(f"Created test file: {file_size} bytes")
    print(f"Artifact saved at: {test_file}")


# Create component from function
create_artifact_op = create_component_from_func(
    create_test_artifact,
    base_image='python:3.11',
    packages_to_install=[]
)


@dsl.pipeline(name="artifact-streaming-test-v1")
def artifact_pipeline_v1():
    """V1 pipeline that creates a test artifact with simple content"""
    
    create_task = create_artifact_op()
    create_task.set_display_name("Create Test Artifact")


if __name__ == "__main__":
    # Compile the pipeline
    kfp.compiler.Compiler().compile(
        pipeline_func=artifact_pipeline_v1,
        package_path="artifact_pipeline.yaml"
    )
    print("Pipeline compiled to artifact_pipeline.yaml")