# Copyright 2020 Google LLC
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
"""Pipeline DSL code for testing URI-based artifact passing."""
from kfp import compiler
from kfp import components
from kfp import dsl


write_to_gcs = components.load_component_from_text("""
name: Write to GCS
inputs:
- {name: text, type: String, description: 'Content to be written to GCS'}
outputs:
- {name: output_gcs_path, type: GCSPath, description: 'GCS file path'}
implementation:
  container:
    image: google/cloud-sdk:slim
    command:
    - sh
    - -c
    - |
      set -e -x
      echo "$0" | gsutil cp - "$1"
    - {inputValue: text}
    - {outputUri: output_gcs_path}
""")

read_from_gcs = components.load_component_from_text("""
name: Read from GCS
inputs:
- {name: input_gcs_path, type: GCSPath, description: 'GCS file path'}
implementation:
  container:
    image: google/cloud-sdk:slim
    command:
    - sh
    - -c
    - |
      set -e -x
      gsutil cat "$0"
    - {inputUri: input_gcs_path}
""")


@dsl.pipeline(
    name='uri-artifact-pipeline',
    output_directory='gs://jxzheng-helloworld/kfp-uri-passing')
def uri_artifact(text='Hello world!'):
  component_1 = write_to_gcs(text=text)
  component_2 = read_from_gcs(
      input_gcs_path=component_1.outputs['output_gcs_path'])


if __name__ == '__main__':
  compiler.Compiler().compile(uri_artifact, __file__ + '.tar.gz')


