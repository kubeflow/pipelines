# Copyright 2020 The Kubeflow Authors
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
import kfp.compiler as compiler

component_op = components.load_component_from_text("""
name: Component with concat placeholder
inputs:
- {name: input_one, type: String}
- {name: input_two, type: String}
implementation:
  container:
    image: ghcr.io/containerd/busybox
    command:
    - sh
    - -ec
    args:
    - echo "$0" > /tmp/test && [[ "$0" == 'one+two=three' ]]
    - concat: [{inputValue: input_one}, '+', {inputValue: input_two}, '=three']
""")


@dsl.pipeline(name='one-step-pipeline-with-concat-placeholder')
def pipeline_with_concat_placeholder():
    component = component_op(input_one='one', input_two='two')
