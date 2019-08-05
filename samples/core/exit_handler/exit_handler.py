#!/usr/bin/env python3
# Copyright 2019 Google LLC
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
from kfp import dsl


def gcs_download_op(url):
    return dsl.ContainerOp(
        name='GCS - Download',
        image='google/cloud-sdk:216.0.0',
        command=['sh', '-c'],
        arguments=['gsutil cat $0 | tee $1', url, '/tmp/results.txt'],
        file_outputs={
            'data': '/tmp/results.txt',
        }
    )


def echo_op(text, is_exit_handler=False):
    return dsl.ContainerOp(
        name='echo',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['echo "$0"', text],
        is_exit_handler=is_exit_handler
    )


@dsl.pipeline(
    name='Exit Handler',
    description='Downloads a message and prints it. The exit handler will run after the pipeline finishes (successfully or not).'
)
def download_and_print(url='gs://ml-pipeline-playground/shakespeare1.txt'):
    """A sample pipeline showing exit handler."""

    exit_task = echo_op('exit!', is_exit_handler=True)

    with dsl.ExitHandler(exit_task):
        download_task = gcs_download_op(url)
        echo_task = echo_op(download_task.output)


if __name__ == '__main__':
    kfp.compiler.Compiler().compile(download_and_print, __file__ + '.zip')
