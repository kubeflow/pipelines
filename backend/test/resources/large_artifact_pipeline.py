#!/usr/bin/env python3

import kfp
from kfp import dsl


@dsl.pipeline(name="large-artifact-memory-test-v1-1gb")
def large_artifact_pipeline_v1():
    """V1 pipeline that generates a large artifact for testing streaming endpoint"""

    generate_task = dsl.ContainerOp(
        name='generate-large-artifact',
        image='python:3.9-slim',
        command=['python3', '-c'],
        arguments=['''
import os

size_mb = 10  # 10MB
chunk_size = 1024 * 1024  # 1MB chunks

# Create a file that will be saved as an artifact
artifact_path = "/tmp/large_artifact.bin"

print(f"Generating {size_mb}MB test file at {artifact_path}")

# Ensure the directory exists
os.makedirs(os.path.dirname(artifact_path), exist_ok=True)

with open(artifact_path, 'wb') as f:
    for i in range(size_mb):
        data = f"CHUNK_{i:04d}_" + "X" * (chunk_size - 20) + f"_END_{i:04d}\\n"
        f.write(data[:chunk_size].encode('utf-8'))

file_size = os.path.getsize(artifact_path)
print(f"Generated file size: {file_size / (1024 * 1024):.2f}MB")
print(f"Artifact saved at: {artifact_path}")
'''],
        file_outputs={'large_file': '/tmp/large_artifact.bin'}
    )

    generate_task.set_display_name("Generate Large Artifact V1 - 1GB")


if __name__ == "__main__":
    # Compile the pipeline in v1 mode
    kfp.compiler.Compiler().compile(
        pipeline_func=large_artifact_pipeline_v1,
        package_path="large_artifact_pipeline.yaml"
    )
    print("Pipeline compiled to large_artifact_pipeline.yaml")