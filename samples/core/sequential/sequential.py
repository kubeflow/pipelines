#!/usr/bin/env python3
# Copyright 2019-2023 The Kubeflow Authors
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


from kfp import dsl, compiler


@dsl.container_component
def gcs_download_op(url: str, output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='google/cloud-sdk:279.0.0',
        command=['sh', '-c'],
        args=['gsutil cat $0 | tee $1', url, output],
    )


@dsl.container_component
def echo_op(text: str):
    return dsl.ContainerSpec(
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        args=['echo "$0"', text]
    )


@dsl.pipeline(
    name='sequential-pipeline',
    description='A pipeline with two sequential steps.'
)
def sequential_pipeline(url: str = 'gs://ml-pipeline/sample-data/shakespeare/shakespeare1.txt'):
    """A pipeline with two sequential steps."""

    download_task = gcs_download_op(url=url)
    echo_task = echo_op(text=download_task.output)


if __name__ == '__main__':
    compiler.Compiler().compile(sequential_pipeline, __file__ + '.yaml')
