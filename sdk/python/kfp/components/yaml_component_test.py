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

import os
import tempfile
import textwrap
import unittest

from kfp.components import structures
from kfp.components import yaml_component

SAMPLE_YAML = textwrap.dedent("""\
components:
  comp-component-1:
    executorLabel: exec-component-1
    inputDefinitions:
      parameters:
        input1:
          parameterType: STRING
    outputDefinitions:
      parameters:
        output1:
          parameterType: STRING
deploymentSpec:
  executors:
    exec-component-1:
      container:
        command:
        - sh
        - -c
        - 'set -ex

          echo "$0" > "$1"'
        - '{{$.inputs.parameters[''input1'']}}'
        - '{{$.outputs.parameters[''output1''].output_file}}'
        image: alpine
pipelineInfo:
  name: component-1
root:
  dag:
    tasks:
      component-1:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-component-1
        inputs:
          parameters:
            input1:
              componentInputParameter: input1
        taskInfo:
          name: component-1
  inputDefinitions:
    parameters:
      input1:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.3
        """)

V1_COMPONENTS_TEST_DATA_DIR = os.path.join(
    os.path.dirname(os.path.dirname(__file__)), 'compiler', 'test_data',
    'v1_component_yaml')

V1_COMPONENT_YAML_TEST_CASES = [
    'concat_placeholder_component.yaml',
    'ingestion_component.yaml',
    'serving_component.yaml',
    'if_placeholder_component.yaml',
    'trainer_component.yaml',
    'add_component.yaml',
]


class YamlComponentTest(unittest.TestCase):

    def test_load_component_from_text(self):
        component = yaml_component.load_component_from_text(SAMPLE_YAML)
        self.assertEqual(component.component_spec.name, 'component-1')
        self.assertEqual(component.component_spec.outputs,
                         {'output1': structures.OutputSpec(type='String')})
        self.assertEqual(component._component_inputs, {'input1'})
        self.assertEqual(component.name, 'component-1')
        self.assertEqual(
            component.component_spec.implementation.container.image, 'alpine')

    def test_load_component_from_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'sample_yaml.yaml')
            with open(path, 'w') as f:
                f.write(SAMPLE_YAML)
            component = yaml_component.load_component_from_file(path)
        self.assertEqual(component.component_spec.name, 'component-1')
        self.assertEqual(component.component_spec.outputs,
                         {'output1': structures.OutputSpec(type='String')})
        self.assertEqual(component._component_inputs, {'input1'})
        self.assertEqual(component.name, 'component-1')
        self.assertEqual(
            component.component_spec.implementation.container.image, 'alpine')

    def test_load_component_from_url(self):
        component_url = 'https://raw.githubusercontent.com/kubeflow/pipelines/7b49eadf621a9054e1f1315c86f95fb8cf8c17c3/sdk/python/kfp/compiler/test_data/components/identity.yaml'
        component = yaml_component.load_component_from_url(component_url)

        self.assertEqual(component.component_spec.name, 'identity')
        self.assertEqual(component.component_spec.outputs,
                         {'Output': structures.OutputSpec(type='String')})
        self.assertEqual(component._component_inputs, {'value'})
        self.assertEqual(component.name, 'identity')
        self.assertEqual(
            component.component_spec.implementation.container.image,
            'python:3.7')


if __name__ == '__main__':
    unittest.main()
