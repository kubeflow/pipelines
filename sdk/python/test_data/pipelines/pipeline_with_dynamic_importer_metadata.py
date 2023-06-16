# Copyright 2022 The Kubeflow Authors
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
"""Pipeline using dsl.importer, with dynamic metadata."""

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import importer

DEFAULT_ARTIFACT_URI = 'gs://ml-pipeline-playground/shakespeare1.txt'
DEFAULT_IMAGE_URI = 'us-docker.pkg.dev/vertex-ai/prediction/tf2-gpu.2-5:latest'


@dsl.component
def make_name(name: str) -> str:
    return name


@dsl.pipeline(name='pipeline-with-importer')
def my_pipeline(name: str = 'default-name',
                int_input: int = 1,
                pipeline_input_artifact_uri: str = DEFAULT_ARTIFACT_URI,
                pipeline_input_image_uri: str = DEFAULT_IMAGE_URI):

    importer1 = importer(
        artifact_uri=pipeline_input_artifact_uri,
        artifact_class=Dataset,
        reimport=False,
        metadata={
            'name': [name, 'alias-name'],
            'containerSpec': {
                'imageUri': pipeline_input_image_uri
            }
        })

    make_name_op = make_name(name='a-different-name')

    importer2 = importer(
        artifact_uri=DEFAULT_ARTIFACT_URI,
        artifact_class=Dataset,
        reimport=False,
        metadata={
            'name': f'prefix-{make_name_op.output}',
            'list-of-data': [make_name_op.output, name, int_input],
            make_name_op.output: make_name_op.output,
            name: DEFAULT_IMAGE_URI,
            'containerSpec': {
                'imageUri': DEFAULT_IMAGE_URI
            }
        })


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
