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
"""Pipeline using dsl.importer."""

import kfp.v2.components.experimental as components
import kfp.v2.dsl.experimental as dsl
from kfp.v2.compiler.experimental import compiler
from kfp.v2.components.experimental import importer

train = components.load_component_from_text("""\
name: train
inputs:
  input1: {type: String}
outputs:
  output1: {type: String}
implementation:
  container:
    image: alpine
    commands:
    - sh
    - -c
    - 'set -ex

        echo "$0" > "$1"'
    - {inputValue: input1}
    - {outputPath: output1}
""")

@dsl.pipeline(name='pipeline-with-importer', pipeline_root='dummy_root')
def my_pipeline(dataset2: str = 'gs://ml-pipeline-playground/shakespeare2.txt'):
    importer1 = importer.importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class='Dataset',
        reimport=False)
    train(dataset=importer1.output)

if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.json'))
