# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Test common GCPC utils."""

from typing import Dict, List

import unittest
from google_cloud_pipeline_components import utils
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


class ComponentsCompileTest(unittest.TestCase):

  def test_no_placeholders_string(self):
    @dsl.container_component
    def comp():
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps('hello')],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '"hello"',
    )

  def test_no_placeholders_int(self):
    @dsl.container_component
    def comp():
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(1)],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '1',
    )

  def test_no_placeholders_dict(self):
    @dsl.container_component
    def comp():
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps({'key': 'value'})],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '{"key": "value"}',
    )

  def test_no_placeholders_list(self):
    @dsl.container_component
    def comp():
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(['foo', 0])],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '["foo", 0]',
    )

  def test_no_placeholders_list(self):
    @dsl.container_component
    def comp():
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(['foo', 0])],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '["foo", 0]',
    )

  def test_placeholder_string(self):
    @dsl.container_component
    def comp(s: str):
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(s)],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        '''"{{$.inputs.parameters['s']}}"''',
    )

  def test_placeholder_int(self):
    @dsl.container_component
    def comp(i: int):
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(i)],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        """{{$.inputs.parameters['i']}}""",
    )

  def test_placeholder_float(self):
    @dsl.container_component
    def comp(f: float):
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps(f)],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        """{{$.inputs.parameters['f']}}""",
    )

  def test_placeholder_dict(self):
    @dsl.container_component
    def comp(s: str):
      return dsl.ContainerSpec(
          image='alpine',
          command=['echo'],
          args=[utils.container_component_dumps({'key': s})],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        """{"key": "{{$.inputs.parameters['s']}}"}""",
    )

  def test_with_all_placeholders(self):
    @dsl.container_component
    def comp(
        in_artifact: Input[Artifact],
        out_artifact: Output[Artifact],
        string: str = 'hello',
        integer: int = 1,
        floating_pt: float = 0.1,
        boolean: bool = True,
        dictionary: Dict = {'key': 'value'},
        array: List = [1, 2, 3],
        hlist: List = [
            {'k': 'v'},
            1,
            ['a'],
            'a',
        ],
    ):
      return dsl.ContainerSpec(
          image='alpine',
          command=[
              'echo',
          ],
          args=[
              utils.container_component_dumps([
                  string,
                  integer,
                  floating_pt,
                  boolean,
                  dictionary,
                  array,
                  hlist,
                  in_artifact.path,
                  out_artifact.path,
              ])
          ],
      )

    self.assertEqual(
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
        """["{{$.inputs.parameters['string']}}", {{$.inputs.parameters['integer']}}, {{$.inputs.parameters['floating_pt']}}, {{$.inputs.parameters['boolean']}}, {{$.inputs.parameters['dictionary']}}, {{$.inputs.parameters['array']}}, {{$.inputs.parameters['hlist']}}, "{{$.inputs.artifacts['in_artifact'].path}}", "{{$.outputs.artifacts['out_artifact'].path}}"]""",
    )


class TestGCPCOutputNameConverter(unittest.TestCase):

  def test_container_component_parameter(self):

    @utils.gcpc_output_name_converter('parameter', 'output__parameter')
    @dsl.container_component
    def comp(
        parameter: str,
        output__parameter: dsl.OutputPath(str),
    ):
      return dsl.ContainerSpec(
          image='alpine',
          command=[
              'echo',
              output__parameter,
          ],
          args=[output__parameter],
      )

    # check inner component
    self.assertIn(
        'parameter',
        comp.pipeline_spec.components[
            'comp-comp'
        ].output_definitions.parameters,
    )
    self.assertNotIn(
        'output__parameter',
        comp.pipeline_spec.components[
            'comp-comp'
        ].output_definitions.parameters,
    )

    # check root
    self.assertIn(
        'parameter',
        comp.pipeline_spec.root.output_definitions.parameters,
    )
    self.assertNotIn(
        'output__parameter',
        comp.pipeline_spec.root.output_definitions.parameters,
    )

    # check command and args
    self.assertEqual(
        "{{$.outputs.parameters['parameter'].output_file}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['command'][1],
    )
    self.assertEqual(
        "{{$.outputs.parameters['parameter'].output_file}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
    )

  def test_container_component_artifact(self):

    @utils.gcpc_output_name_converter('artifact', 'output__artifact')
    @dsl.container_component
    def comp(
        artifact: Input[Artifact],
        output__artifact: Output[Artifact],
    ):
      return dsl.ContainerSpec(
          image='alpine',
          command=[
              'echo',
              output__artifact.path,
          ],
          args=[output__artifact.path],
      )

    # check inner component
    self.assertIn(
        'artifact',
        comp.pipeline_spec.components['comp-comp'].output_definitions.artifacts,
    )
    self.assertNotIn(
        'output__artifact',
        comp.pipeline_spec.components['comp-comp'].output_definitions.artifacts,
    )

    # check root
    self.assertIn(
        'artifact',
        comp.pipeline_spec.root.output_definitions.artifacts,
    )
    self.assertNotIn(
        'output__artifact',
        comp.pipeline_spec.root.output_definitions.artifacts,
    )

    # check command and args
    self.assertEqual(
        "{{$.outputs.artifacts['artifact'].path}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['command'][1],
    )
    self.assertEqual(
        "{{$.outputs.artifacts['artifact'].path}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
    )

  def test_omitted_original_name(self):
    @utils.gcpc_output_name_converter('artifact')
    @dsl.container_component
    def comp(
        artifact: Input[Artifact],
        output__artifact: Output[Artifact],
    ):
      return dsl.ContainerSpec(
          image='alpine',
          command=[
              'echo',
              output__artifact.path,
          ],
          args=[output__artifact.path],
      )

    # check inner component
    self.assertIn(
        'artifact',
        comp.pipeline_spec.components['comp-comp'].output_definitions.artifacts,
    )
    self.assertNotIn(
        'output__artifact',
        comp.pipeline_spec.components['comp-comp'].output_definitions.artifacts,
    )

    # check root
    self.assertIn(
        'artifact',
        comp.pipeline_spec.root.output_definitions.artifacts,
    )
    self.assertNotIn(
        'output__artifact',
        comp.pipeline_spec.root.output_definitions.artifacts,
    )

    # check command and args
    self.assertEqual(
        "{{$.outputs.artifacts['artifact'].path}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['command'][1],
    )
    self.assertEqual(
        "{{$.outputs.artifacts['artifact'].path}}",
        comp.pipeline_spec.deployment_spec['executors']['exec-comp'][
            'container'
        ]['args'][0],
    )

  def test_cannot_use_on_pipeline(self):
    @dsl.container_component
    def comp(
        artifact: Input[Artifact],
    ):
      return dsl.ContainerSpec(
          image='alpine',
          command=[
              'echo',
              artifact.path,
          ],
      )

    with self.assertRaisesRegex(
        ValueError,
        r'The \'gcpc_output_name_converter\' decorator can only be used on'
        r' primitive container components. You are trying to use it on a'
        r' pipeline\.',
    ):

      @utils.gcpc_output_name_converter('artifact', 'output__artifact')
      @dsl.pipeline
      def pipeline(artifact: Input[Artifact]):
        comp(artifact=artifact)

  def test_cannot_use_on_python_component(self):
    with self.assertRaisesRegex(
        ValueError,
        r'The \'gcpc_output_name_converter\' decorator can only be used on'
        r' primitive container components. You are trying to use it on a'
        r' Python component\.',
    ):

      @utils.gcpc_output_name_converter('artifact', 'output__artifact')
      @dsl.component
      def comp(
          artifact: Input[Artifact],
          output__artifact: Output[Artifact],
      ):
        pass
