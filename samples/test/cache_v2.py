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
"""Random Two step v2-compatible pipeline."""

from kfp import components, dsl
from kfp.components import OutputPath
from two_step import train_op


def preprocess(
        uri: str, some_int: int, output_parameter_one: OutputPath(int),
        output_dataset_one: OutputPath('Dataset')
):
    with open(output_dataset_one, 'w') as f:
        f.write(uri)
    with open(output_parameter_one, 'w') as f:
        f.write("{}".format(some_int))


preprocess_op = components.create_component_from_func(
    preprocess, base_image='python:3.9'
)

@dsl.pipeline(name='random_two_step_pipeline')
def random_two_step_pipeline(uri: str, some_int: int):
    preprocess_task = preprocess_op(uri=uri, some_int=some_int)
    train_task = train_op(
        num_steps=preprocess_task.outputs['output_parameter_one'],
        dataset=preprocess_task.outputs['output_dataset_one']
    )
