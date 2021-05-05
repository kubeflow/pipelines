#!/usr/bin/env python3
# Copyright 2019-2021 The Kubeflow Authors
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

import kfp
from kfp import dsl, components

gcs_download_op = kfp.components.load_component_from_url(
    'https://raw.githubusercontent.com/kubeflow/pipelines/1.5.0-rc.3/components/google-cloud/storage/download_blob/component.yaml'
)


@components.create_component_from_func
def echo_op(msg: str):
    """Print a message."""
    print(msg)


@dsl.pipeline(
    name='Exit Handler',
    description=
    'Downloads a message and prints it. The exit handler will run after the pipeline finishes (successfully or not).'
)
def pipeline_exit_handler(url='gs://ml-pipeline/shakespeare1.txt'):
    """A sample pipeline showing exit handler."""

    exit_task = echo_op('exit!')

    with dsl.ExitHandler(exit_task):
        download_task = gcs_download_op(url)
        echo_task = echo_op(download_task.output)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(pipeline_exit_handler, __file__ + '.yaml')
