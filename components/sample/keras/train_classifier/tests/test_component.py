# Copyright 2018 Google LLC
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

import os
import subprocess
import tempfile
import unittest
from contextlib import contextmanager
from pathlib import Path

import kfp.components as comp

@contextmanager
def components_local_output_dir_context(output_dir: str):
    old_dir = comp._components._outputs_dir
    try:
        comp._components._outputs_dir = output_dir
        yield output_dir
    finally:
        comp._components._outputs_dir = old_dir

class KerasTrainClassifierTestCase(unittest.TestCase):
    def test_handle_training_xor(self):
        tests_root = os.path.abspath(os.path.dirname(__file__))
        component_root = os.path.abspath(os.path.join(tests_root, '..'))
        testdata_root = os.path.abspath(os.path.join(tests_root, 'testdata'))
        
        train_op = comp.load_component(os.path.join(component_root, 'component.yaml'))

        with tempfile.TemporaryDirectory() as temp_dir_name:
            with components_local_output_dir_context(temp_dir_name):
                train_task = train_op(
                    training_set_features_path=os.path.join(testdata_root, 'training_set_features.tsv'),
                    training_set_labels_path=os.path.join(testdata_root, 'training_set_labels.tsv'),
                    output_model_uri=os.path.join(temp_dir_name, 'outputs/output_model/data'),
                    model_config=Path(testdata_root).joinpath('model_config.json').read_text(),
                    number_of_classes=2,
                    number_of_epochs=10,
                    batch_size=32,
                )

            full_command = train_task.command + train_task.arguments
            full_command[0] = 'python'
            full_command[1] = os.path.join(component_root, 'src', 'train.py')

            process = subprocess.run(full_command)

            (output_model_uri_file, ) = (train_task.file_outputs['output-model-uri'], )
            output_model_uri = Path(output_model_uri_file).read_text()


if __name__ == '__main__':
    unittest.main()
