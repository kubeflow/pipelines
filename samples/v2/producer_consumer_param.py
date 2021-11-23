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

# Simple two-step pipeline with 'producer' and 'consumer' steps
from kfp.v2 import components, compiler, dsl

producer_op = components.load_component_from_text("""
name: Producer
inputs:
- {name: input_text, type: String, description: 'Represents an input parameter.'}
outputs:
- {name: output_value, type: String, description: 'Represents an output paramter.'}
implementation:
  container:
    image: google/cloud-sdk:latest
    command:
    - sh
    - -c
    - |
      set -e -x
      echo "$0, this is an output parameter" | gsutil cp - "$1"
    - {inputValue: input_text}
    - {outputPath: output_value}
""")

consumer_op = components.load_component_from_text("""
name: Consumer
inputs:
- {name: input_value, type: String, description: 'Represents an input parameter. It connects to an upstream output parameter.'}
implementation:
  container:
    image: google/cloud-sdk:latest
    command:
    - sh
    - -c
    - |
      set -e -x
      echo "Read from an input parameter: " && echo "$0"
    - {inputValue: input_value}
""")


@dsl.pipeline(name='producer-consumer-param-pipeline')
def producer_consumer_param_pipeline(text: str = 'Hello world'):
    producer = producer_op(input_text=text)
    consumer = consumer_op(input_value=producer.outputs['output_value'])


if __name__ == "__main__":
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=producer_consumer_param_pipeline,
        package_path='producer_consumer_param_pipeline.json')
