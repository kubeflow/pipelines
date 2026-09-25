# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Compile the native IR fixtures used by the browser integration tests."""

from pathlib import Path

from kfp import compiler
from kfp import dsl


@dsl.container_component
def echo(message: str, node: str):
    return dsl.ContainerSpec(
        image='alpine:3.21',
        command=['echo'],
        args=[message, 'from node:', node],
    )


@dsl.pipeline(name='helloworld')
def helloworld(message: str = 'hello world'):
    a = echo(message=message, node='A').set_display_name('A')
    b = echo(message=message, node='B').set_display_name('B').after(a)
    c = echo(message=message, node='C').set_display_name('C').after(a)
    echo(message=message, node='D').set_display_name('D').after(b, c)


@dsl.component(base_image='python:3.11-slim')
def tensorboard_metadata(mlpipeline_ui_metadata: dsl.Output[dsl.Artifact]):
    import json

    with open(mlpipeline_ui_metadata.path, 'w') as metadata_file:
        json.dump(
            {
                'outputs': [{
                    'type': 'tensorboard',
                    'source': 'gs://ml-pipeline-dataset/tensorboard-train',
                }],
            }, metadata_file)


@dsl.pipeline(name='tensorboard-example')
def tensorboard_example():
    tensorboard_metadata()


if __name__ == '__main__':
    directory = Path(__file__).resolve().parent
    compiler.Compiler().compile(helloworld, str(directory / 'helloworld.yaml'))
    compiler.Compiler().compile(tensorboard_example,
                                str(directory / 'tensorboard-example.yaml'))
