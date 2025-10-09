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
"""Pipeline using dsl.importer and GCPC types."""

from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.dsl import importer


class VertexDataset(dsl.Artifact):
    """An artifact representing a GCPC Vertex Dataset."""
    schema_title = 'google.VertexDataset'


consumer_op = components.load_component_from_text("""
name: consumer_op
inputs:
  - {name: dataset, type: google.VertexDataset}
implementation:
  container:
    image: dummy
    command:
    - cmd
    args:
    - {inputPath: dataset}
""")


@dsl.pipeline(name='pipeline-with-importer-and-gcpc-type')
def my_pipeline():

    importer1 = importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class=VertexDataset,
        reimport=False,
        metadata={'key': 'value'})
    consume1 = consumer_op(dataset=importer1.output)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
