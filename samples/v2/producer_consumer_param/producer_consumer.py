PIPELINE_ROOT = 'gs://gongyuan-test/pipeline_root/'

# Simple two-step pipeline with 'producer' and 'consumer' steps
from kfp.v2 import components
from kfp.v2 import compiler
from kfp.v2 import dsl

producer_op = components.load_component_from_text(
    """
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
"""
)

consumer_op = components.load_component_from_text(
    """
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
"""
)


@dsl.pipeline(name='simple-two-step-pipeline')
def two_step_pipeline(text='Hello world'):
    producer = producer_op(input_text=text)
    consumer = consumer_op(input_value=producer.outputs['output_value'])


compiler.Compiler().compile(
    pipeline_func=two_step_pipeline,
    pipeline_root=PIPELINE_ROOT,
    output_path='two_step_pipeline_job.json'
)
