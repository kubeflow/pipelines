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
"""Two step v2-compatible pipeline."""

from kfp.deprecated import components
from kfp.deprecated import dsl
from kfp.deprecated.components import InputPath
from kfp.deprecated.components import OutputPath


def preprocess(uri: str, some_int: int, output_parameter_one: OutputPath(int),
               output_dataset_one: OutputPath('Dataset')):
    """Dummy Preprocess Step."""
    with open(output_dataset_one, 'w') as f:
        f.write('Output dataset')
    with open(output_parameter_one, 'w') as f:
        f.write("{}".format(1234))


preprocess_op = components.create_component_from_func(
    preprocess, base_image='python:3.9')


@components.create_component_from_func
def train_op(dataset: InputPath('Dataset'),
             model: OutputPath('Model'),
             num_steps: int = 100):
    """Dummy Training Step."""

    with open(dataset, 'r') as input_file:
        input_string = input_file.read()
        with open(model, 'w') as output_file:
            for i in range(num_steps):
                output_file.write("Step {}\n{}\n=====\n".format(
                    i, input_string))


@dsl.pipeline(name='two_step_pipeline')
def two_step_pipeline():
    preprocess_task = preprocess_op(uri='uri-to-import', some_int=12)
    train_task = train_op(
        num_steps=preprocess_task.outputs['output_parameter_one'],
        dataset=preprocess_task.outputs['output_dataset_one'])
