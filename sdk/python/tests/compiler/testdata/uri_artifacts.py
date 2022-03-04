# Copyright 2020 The Kubeflow Authors
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
from kfp.deprecated import compiler
from kfp.deprecated import components
from kfp.deprecated import dsl


# Patch to make the test result deterministic.
class Coder:

    def __init__(self,):
        self._code_id = 0

    def get_code(self,):
        self._code_id += 1
        return '{code:0{num_chars:}d}'.format(
            code=self._code_id,
            num_chars=dsl._for_loop.LoopArguments.NUM_CODE_CHARS)


dsl.ParallelFor._get_unique_id_code = Coder().get_code

write_to_gcs = components.load_component_from_text("""
name: Write to GCS
inputs:
- {name: text, type: String, description: 'Content to be written to GCS'}
outputs:
- {name: output_gcs_path, type: Artifact, description: 'GCS file path'}
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
- {name: input_gcs_path, type: Artifact, description: 'GCS file path'}
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

flip_coin_op = components.load_component_from_text("""
name: Flip coin
outputs:
- {name: output, type: String}
implementation:
  container:
    image: python:alpine3.6
    command:
    - sh
    - -c
    args:
    - mkdir -p "$(dirname $0)" && python -c "import random; result = \'heads\' if random.randint(0,1) == 0 else \'tails\'; print(result, end='')" | tee $0
    - {outputPath: output}
""")


@dsl.pipeline(
    name='uri-artifact-pipeline', pipeline_root='gs://my-bucket/my-output-dir')
def uri_artifact(text: str = 'Hello world!'):
    task_1 = write_to_gcs(text=text)
    task_2 = read_from_gcs(input_gcs_path=task_1.outputs['output_gcs_path'])

    # Test use URI within ParFor loop.
    loop_args = [1, 2, 3, 4]
    with dsl.ParallelFor(loop_args) as loop_arg:
        loop_task_2 = read_from_gcs(
            input_gcs_path=task_1.outputs['output_gcs_path'])

    # Test use URI within condition.
    flip = flip_coin_op()
    with dsl.Condition(flip.output == 'heads'):
        condition_task_2 = read_from_gcs(
            input_gcs_path=task_1.outputs['output_gcs_path'])


if __name__ == '__main__':
    compiler.Compiler(mode=dsl.PipelineExecutionMode.V2_COMPATIBLE).compile(
        uri_artifact, __file__.replace('.py', '.yaml'))
