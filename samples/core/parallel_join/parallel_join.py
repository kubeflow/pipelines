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

@dsl.container_component()
def gcs_download_op(url: str, output: dsl.OutputPath(str)):
    return dsl.ContainerSpec(
        image='google/cloud-sdk:279.0.0',
        command=['sh', '-c', '''mkdir -p $(dirname $1)\
                               && gsutil cat $0 | tee $1'''],
        args=[url, output],
    )


@dsl.container_component()
def echo2_op(text1: str, text2: str):
    return dsl.ContainerSpec(
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        args=['echo "Text 1: $0"; echo "Text 2: $1"', text1, text2]
    )


@dsl.pipeline(
  name='parallel-pipeline',
  description='Download two messages in parallel and prints the concatenated result.'
)
def download_and_join(
    url1: str='gs://ml-pipeline/sample-data/shakespeare/shakespeare1.txt',
    url2: str='gs://ml-pipeline/sample-data/shakespeare/shakespeare2.txt'
):
    """A three-step pipeline with first two running in parallel."""

    download1_task = gcs_download_op(url=url1)
    download2_task = gcs_download_op(url=url2)

    echo_task = echo2_op(text1=download1_task.output, text2=download2_task.output)

if __name__ == '__main__':
    compiler.Compiler().compile(download_and_join, __file__ + '.yaml')
