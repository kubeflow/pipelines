# Copyright 2021 The Kubeflow Authors
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
"""Two step pipeline with URI placeholders."""
from kfp import dsl


@dsl.container_component
def write_to_gcs(msg: str, artifact: dsl.Output[dsl.Artifact]):
    """
    Args:
        msg: Content to be written to GCS
        artifact: GCS file path"""
    return dsl.ContainerSpec(
        image='google/cloud-sdk:slim',
        command=[
            'sh', '-c', 'set -e -x\necho "$0" | gsutil cp - "$1"\n', msg,
            artifact.uri
        ],
    )


write_to_gcs_op = write_to_gcs


@dsl.container_component
def read_from_gcs(artifact: dsl.Input[dsl.Artifact]):
    """
    Args:
        artifact: GCS file path"""
    return dsl.ContainerSpec(
        image='google/cloud-sdk:slim',
        command=['sh', '-c', 'set -e -x\ngsutil cat "$0"\n', artifact.uri],
    )


read_from_gcs_op = read_from_gcs


@dsl.pipeline(name='two-step-with-uri-placeholders')
def two_step_with_uri_placeholder(msg: str = 'Hello world!'):
    write_to_gcs = write_to_gcs_op(msg=msg)
    read_from_gcs = read_from_gcs_op(artifact=write_to_gcs.outputs['artifact'])
