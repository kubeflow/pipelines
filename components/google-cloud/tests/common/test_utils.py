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

import json
import os
from typing import Dict, List

import kfp
from google_cloud_pipeline_components import utils
from kfp import compiler, dsl
from kfp.dsl import Artifact, Input, Output

import unittest


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
