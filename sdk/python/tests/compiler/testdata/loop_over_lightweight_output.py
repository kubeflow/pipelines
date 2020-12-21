#!/usr/bin/env python3
# Copyright 2019 Google LLC
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
from kfp.dsl import _for_loop

produce_op = kfp.components.load_component_from_text('''\
name: Produce list
outputs:
- name: data_list
implementation:
  container:
    image: busybox
    command:
    - sh
    - -c
    - echo "[1, 2, 3]" > "$0"
    - outputPath: data_list
''')

consume_op = kfp.components.load_component_from_text('''\
name: Consume data
inputs:
- name: data
implementation:
  container:
    image: busybox
    command:
    - echo
    - inputValue: data
''')

@dsl.pipeline(
    name='Loop over lightweight output',
    description='Test pipeline to verify functions of par loop.'
)
def pipeline():
  source_task = produce_op()
  with dsl.ParallelFor(source_task.output) as item:
    consume_op(item)
