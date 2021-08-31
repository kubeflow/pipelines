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
from unittest.mock import patch, mock_open
from kfp.v2.components.experimental import component_spec
import textwrap


class ComponentSpecTest(unittest.TestCase):

  @unittest.skip("Placeholder check is not completed. ")
  def test_component_spec_with_placeholder_referencing_nonexisting_input_output(
      self):
    with self.assertRaisesRegex(
        ValueError,
        'Argument "InputValuePlaceholder\(name=\'input000\'\)" '
        'references non-existing input.'):
      component_spec.ComponentSpec(
          name='component_1',
          implementation=component_spec.ContainerSpec(
              image='alpine',
              commands=[
                  'sh',
                  '-c',
                  'set -ex\necho "$0" > "$1"',
                  component_spec.InputValuePlaceholder(name='input000'),
                  component_spec.OutputPathPlaceholder(name='output1'),
              ],
          ),
          input_specs=[
              component_spec.InputSpec(name='input1', type='String'),
          ],
          output_specs=[
              component_spec.OutputSpec(name='output1', type='String'),
          ],
      )

    with self.assertRaisesRegex(
        ValueError,
        'Argument "OutputPathPlaceholder\(name=\'output000\'\)" '
        'references non-existing output.'):
      component_spec.ComponentSpec(
          name='component_1',
          implementation=component_spec.ContainerSpec(
              image='alpine',
              commands=[
                  'sh',
                  '-c',
                  'set -ex\necho "$0" > "$1"',
                  component_spec.InputValuePlaceholder(name='input1'),
                  component_spec.OutputPathPlaceholder(name='output000'),
              ],
          ),
          input_specs=[
              component_spec.InputSpec(name='input1', type='String'),
          ],
          output_specs=[
              component_spec.OutputSpec(name='output1', type='String'),
          ],
      )

  def test_component_spec_save_to_component_yaml(self):
    open_mock = mock_open()
    expected_yaml = textwrap.dedent("""\
        implementation:
          commands:
          - sh
          - -c
          - 'set -ex

            echo "$0" > "$1"'
          - name: input1
          - name: output1
          image: alpine
        inputs:
          input1:
            type: String
        name: component_1
        outputs:
          output1:
            type: String
        """)

    with patch("builtins.open", open_mock, create=True):
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
              'input1': component_spec.InputSpec(type='String')
            },
            outputs={
              'output1': component_spec.OutputSpec(type='String')
            },
      ).save_to_component_yaml('test_save_file.txt')

    open_mock.assert_called_with('test_save_file.txt', 'a')
    open_mock.return_value.write.assert_called_once_with(expected_yaml)

if __name__ == '__main__':
  unittest.main()
