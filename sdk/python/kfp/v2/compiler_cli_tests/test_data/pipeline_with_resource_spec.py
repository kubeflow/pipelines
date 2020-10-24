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

import kfp
from kfp import dsl
import kfp.v2.compiler as compiler
import pathlib

test_data_dir = pathlib.Path(__file__).parent / 'component_yaml'

ingestion_op = kfp.components.load_component_from_file(
    str(test_data_dir / 'ingestion_component.yaml')
)

training_op = kfp.components.load_component_from_file(
    str(test_data_dir / 'fancy_trainer_component.yaml')
)


@dsl.pipeline(
    name='two-step-pipeline-with-ontology',
    description='A linear two-step pipeline with artifact ontology types.'
)
def my_pipeline(
    input_location='gs://test-bucket/pipeline_root',
    optimizer: str = 'sgd',
    n_epochs: int = 200
):
  ingestor = ingestion_op(input_location=input_location)
  _ = (training_op(
      examples=ingestor.outputs['examples'],
      schema=ingestor.outputs['schema'],
      optimizer=optimizer,
      n_epochs=n_epochs).
       set_cpu_limit('4').
       set_memory_limit('14Gi').
       add_node_selector_constraint(
      'cloud.google.com/gke-accelerator',
      'tpu-v3').
       set_gpu_limit(1))


if __name__ == '__main__':
  compiler.Compiler().compile(my_pipeline, __file__ + '.json')
