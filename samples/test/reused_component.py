# Copyright 2020-2021 Google LLC
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

from kfp import components
from kfp import dsl

add_op = components.load_component_from_text(
    '''
name: Add
description: |
    Component to add two numbers
inputs:
- name: op1
  type: Integer
- name: op2
  type: Integer
outputs:
- name: sum
  type: Integer
implementation:
  container:
    image: google/cloud-sdk:latest
    command:
    - sh
    - -c
    - |
      set -e -x
      echo "$(($0+$1))" | gsutil cp - "$2"
    - {inputValue: op1}
    - {inputValue: op2}
    - {outputPath: sum}
'''
)


@dsl.pipeline(name='add-pipeline')
def my_pipeline(
    a: int = 2,
    b: int = 5,
):
    first_add_task = add_op(a, 3)
    second_add_task = add_op(first_add_task.outputs['sum'], b)
    third_add_task = add_op(second_add_task.outputs['sum'], 7)
