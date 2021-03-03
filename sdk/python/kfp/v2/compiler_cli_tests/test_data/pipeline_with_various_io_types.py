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

import pathlib

from kfp import components
from kfp.v2 import dsl
import kfp.v2.compiler as compiler

component_op_1 = components.load_component_from_text("""
name: upstream
inputs:
- {name: input_1, type: String}
- {name: input_2, type: Float}
- {name: input_3, type: }
- {name: input_4}
- {name: input_5, type: Metrics}
- {name: input_6, type: Datasets}
- {name: input_7, type: Some arbitrary type}
- {name: input_8, type: {GcsPath: {data_type: TSV}}}
outputs:
- {name: output_1, type: Integer}
- {name: output_2, type: Model}
- {name: output_3}
implementation:
  container:
    image: gcr.io/image
    args:
    - {inputValue: input_1}
    - {inputValue: input_2}
    - {inputUri: input_3}
    - {inputUri: input_4}
    - {inputUri: input_5}
    - {inputUri: input_6}
    - {inputUri: input_7}
    - {inputUri: input_8}
    - {outputPath: output_1}
    - {outputUri: output_2}
    - {outputPath: output_3}
""")

component_op_2 = components.load_component_from_text("""
name: downstream
inputs:
- {name: input_a, type: Integer}
- {name: input_b, type: Model}
- {name: input_c}
implementation:
  container:
    image: gcr.io/image
    args:
    - {inputValue: input_a}
    - {inputUri: input_b}
    - {inputPath: input_c}
""")


@dsl.pipeline(name='pipeline-with-various-types')
def my_pipeline(input1,
                input3,
                input4='',
                input5='gs://bucket/metrics',
                input6='gs://bucket/dataset',
                input7='arbitrary value',
                input8='gs://path2'):
  component_1 = component_op_1(
      input_1=input1,
      input_2=3.1415926,
      input_3=input3,
      input_4=input4,
      input_5='gs://bucket/metrics',
      input_6=input6,
      input_7=input7,
      input_8=input8)
  component_2 = component_op_2(
      input_a=component_1.outputs['output_1'],
      input_b=component_1.outputs['output_2'],
      input_c=component_1.outputs['output_3'])


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      pipeline_root='dummy_root',
      output_path=__file__ + '.json')
