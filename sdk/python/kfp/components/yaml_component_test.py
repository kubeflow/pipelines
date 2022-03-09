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
"""Tests for kfp.components.yaml_component."""

import requests
import unittest
import textwrap

from pathlib import Path
from unittest import mock

from kfp.components import yaml_component
from kfp.components import structures

SAMPLE_YAML = textwrap.dedent("""\
        name: component_1
        inputs:
          input1: {type: String}
        outputs:
          output1: {type: String}
        implementation:
          container:
            image: alpine
            command:
            - sh
            - -c
            - 'set -ex

            echo "$0" > "$1"'
            - {inputValue: input1}
            - {outputPath: output1}
        """)


class YamlComponentTest(unittest.TestCase):

    def test_load_component_from_text(self):
        component = yaml_component.load_component_from_text(SAMPLE_YAML)
        self.assertEqual(component.component_spec.name, 'component_1')
        self.assertEqual(component.component_spec.outputs,
                         {'output1': structures.OutputSpec(type='String')})
        self.assertEqual(component._component_inputs, {'input1'})
        self.assertEqual(component.name, 'component_1')
        self.assertEqual(
            component.component_spec.implementation.container.image, 'alpine')

    def test_load_component_from_file(self):
        component_path = Path(
            __file__).parent / 'test_data' / 'simple_yaml.yaml'
        component = yaml_component.load_component_from_file(component_path)
        self.assertEqual(component.component_spec.name, 'component_1')
        self.assertEqual(component.component_spec.outputs,
                         {'output1': structures.OutputSpec(type='String')})
        self.assertEqual(component._component_inputs, {'input1'})
        self.assertEqual(component.name, 'component_1')
        self.assertEqual(
            component.component_spec.implementation.container.image, 'alpine')

    def test_load_component_from_url(self):
        component_url = 'https://raw.githubusercontent.com/some/repo/components/component_group/component.yaml'

        def mock_response_factory(url, params=None, **kwargs):
            if url == component_url:
                response = requests.Response()
                response.url = component_url
                response.status_code = 200
                response._content = SAMPLE_YAML
                return response
            raise RuntimeError('Unexpected URL "{}"'.format(url))

        with mock.patch('requests.get', mock_response_factory):
            component = yaml_component.load_component_from_url(component_url)
            self.assertEqual(component.component_spec.name, 'component_1')
            self.assertEqual(component.component_spec.outputs,
                             {'output1': structures.OutputSpec(type='String')})
            self.assertEqual(component._component_inputs, {'input1'})
            self.assertEqual(component.name, 'component_1')
            self.assertEqual(
                component.component_spec.implementation.container.image,
                'alpine')


if __name__ == '__main__':
    unittest.main()
