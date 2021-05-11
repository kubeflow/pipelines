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
from kfp import dsl, components

echo = components.load_component_from_text(
    """
name: Echo
inputs:
- {name: text, type: String}
implementation:
  container:
    image: alpine
    command:
    - echo
    - {inputValue: text}
"""
)


@dsl.pipeline(name='parameter_value_missing')
def pipeline(
    parameter:
    str  # parameter should be specified when submitting, but we are missing it in the test
):
    echo_op = echo(text=parameter)
