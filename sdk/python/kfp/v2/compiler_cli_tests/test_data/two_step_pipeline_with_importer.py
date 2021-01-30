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

test_data_dir = pathlib.Path(__file__).parent / 'component_yaml'
trainer_op = components.load_component_from_file(
    str(test_data_dir / 'trainer_component.yaml'))
serving_op = components.load_component_from_file(
    str(test_data_dir / 'serving_component.yaml'))


@dsl.pipeline(
    name='two-step-pipeline-with-importer',
    description='A linear two-step pipeline.')
def my_pipeline(input_gcs='gs://test-bucket/pipeline_root',
                optimizer: str = 'sgd',
                epochs: int = 200):
  trainer = trainer_op(
      input_location=input_gcs, train_optimizer=optimizer, num_epochs=epochs)
  serving = serving_op(
      model=trainer.outputs['model_output'],
      model_cfg=trainer.outputs['model_config'])


if __name__ == '__main__':
  compiler.Compiler().compile(
      pipeline_func=my_pipeline,
      pipeline_root='dummy_root',
      output_path=__file__ + '.json')
