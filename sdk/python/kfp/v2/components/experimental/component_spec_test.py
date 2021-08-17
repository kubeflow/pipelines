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
"""Tests for kfp.v2.components.experimental.component_spec."""

import unittest
import os

from kfp.v2.components.experimental import component_spec


class ComponentSpecTest(unittest.TestCase):

  # Uncomment when init method for ComponentSpec is done.
  # def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
  #     self):
  #   with self.assertRaisesRegex(
  #       ValueError,
  #       'Argument "InputValuePlaceholder\(name=\'input000\'\)" '
  #       'references non-existing input.'):
  #     component_spec.ComponentSpec(
  #         name='component_1',
  #         implementation=component_spec.ContainerSpec(
  #             image='alpine',
  #             commands=[
  #                 'sh',
  #                 '-c',
  #                 'set -ex\necho "$0" > "$1"',
  #                 component_spec.InputValuePlaceholder(name='input000'),
  #                 component_spec.OutputPathPlaceholder(name='output1'),
  #             ],
  #         ),
  #         input_specs=[
  #             component_spec.InputSpec(name='input1', type="str"),
  #         ],
  #         output_specs=[
  #             component_spec.OutputSpec(name='output1', type="str"),
  #         ],
  #     )

  #   with self.assertRaisesRegex(
  #       ValueError,
  #       'Argument "OutputPathPlaceholder\(name=\'output000\'\)" '
  #       'references non-existing output.'):
  #     component_spec.ComponentSpec(
  #         name='component_1',
  #         implementation=component_spec.ContainerSpec(
  #             image='alpine',
  #             commands=[
  #                 'sh',
  #                 '-c',
  #                 'set -ex\necho "$0" > "$1"',
  #                 component_spec.InputValuePlaceholder('input1'),
  #                 component_spec.OutputPathPlaceholder('output000'),
  #             ],
  #         ),
  #         input_specs=[
  #             component_spec.InputSpec(name='input1', type=str),
  #         ],
  #         output_specs=[
  #             component_spec.OutputSpec(name='output1', type=str),
  #         ],
  #     )

  def test_component_spec_save_to_component_yaml(self):
    component_spec.ComponentSpec(
          name='component_1',
          implementation=component_spec.ContainerSpec(
              image='alpine',
              commands=[
                  'sh',
                  '-c',
                  'set -ex\necho "$0" > "$1"',
                  component_spec.InputValuePlaceholder(name='input1'),
                  component_spec.OutputPathPlaceholder(name='output1'),
              ],
          ),
          inputs={
            'input1': component_spec.InputSpec(type="str")
          },
          outputs={
            'output1': component_spec.OutputSpec(type="str")
          },
    ).save_to_component_yaml('test_save_file.txt')

    expected_yaml = """annotations: null
description: null
implementation:
  arguments: null
  commands:
  - sh
  - -c
  - 'set -ex

    echo "$0" > "$1"'
  - name: input1
  - name: output1
  env: null
  image: alpine
  resources: null
inputs:
  input1:
    default: null
    type: str
labels: null
name: component_1
outputs:
  output1:
    type: str
"""

    with open('test_save_file.txt', 'r') as output_file:
      content = output_file.read()
      output_file.close()
      os.remove('test_save_file.txt')
    self.assertMultiLineEqual(content, expected_yaml)

if __name__ == '__main__':
  unittest.main()
