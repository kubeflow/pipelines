# Copyright 2018 Google LLC
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
import kfp.dsl as dsl
from kfp.dsl import component, graph_component
from kfp.dsl._metadata import ComponentMeta, ParameterMeta
from kfp.dsl.types import Integer, GCSPath, InconsistentTypeException
from kfp.dsl import ContainerOp, Pipeline, PipelineParam
import unittest

class TestPythonComponent(unittest.TestCase):

  def test_component_metadata(self):
    """Test component decorator metadata."""

    class MockContainerOp:
      def _set_metadata(self, component_meta):
        self._metadata = component_meta

    @component
    def componentA(a: {'ArtifactA': {'file_type': 'csv'}}, b: Integer() = 12, c: {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}} = 'gs://hello/world') -> {'model': Integer()}:
      return MockContainerOp()

    containerOp = componentA(1,2,c=3)

    golden_meta = ComponentMeta(name='componentA', description='')
    golden_meta.inputs.append(ParameterMeta(name='a', description='', param_type={'ArtifactA': {'file_type': 'csv'}}))
    golden_meta.inputs.append(ParameterMeta(name='b', description='', param_type={'Integer': {'openapi_schema_validator': {"type": "integer"}}}, default=12))
    golden_meta.inputs.append(ParameterMeta(name='c', description='', param_type={'ArtifactB': {'path_type':'file', 'file_type': 'tsv'}}, default='gs://hello/world'))
    golden_meta.outputs.append(ParameterMeta(name='model', description='', param_type={'Integer': {'openapi_schema_validator': {"type": "integer"}}}))

    self.assertEqual(containerOp._metadata, golden_meta)

  def test_type_check_with_same_representation(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': GCSPath(), 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}, 'field_o': 'GcsUri'}:
      return ContainerOp(
        name = 'operator a',
        image = 'gcr.io/ml-pipeline/component-a',
        arguments = [
          '--field-l', field_l,
        ],
        file_outputs = {
          'field_m': '/schema.txt',
          'field_n': '/feature.txt',
          'field_o': '/output.txt'
        }
      )

    @component
    def b_op(field_x: {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}},
        field_y: 'GcsUri',  # noqa: F821 TODO
        field_z: GCSPath()) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-b',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_with_different_represenation(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': 'GCSPath', 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}},
        field_y: Integer(),
        field_z: GCSPath()) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_with_inconsistent_types_property_value(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}, 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}},
        field_y: Integer(),
        field_z: {'ArtifactB': {'path_type': 'file', 'file_type':'csv'}}) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    with self.assertRaises(InconsistentTypeException):
        a = a_op(field_l=12)
        b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_with_inconsistent_types_type_name(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}, 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type_a': {'property_a': 'value_a', 'property_b': 'value_b'}},
        field_y: Integer(),
        field_z: {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    with self.assertRaises(InconsistentTypeException):
        a = a_op(field_l=12)
        b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_without_types(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}, 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x,
        field_y: Integer(),
        field_z: {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_nonnamed_inputs(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}, 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x,
        field_y: Integer(),
        field_z: {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(a.outputs['field_n'], field_z=a.outputs['field_m'], field_y=a.outputs['field_o'])

  def test_type_check_with_inconsistent_types_disabled(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = False
    @component
    def a_op(field_l: Integer()) -> {'field_m': {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}, 'field_n': {'customized_type': {'property_a': 'value_a', 'property_b': 'value_b'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type_a': {'property_a': 'value_a', 'property_b': 'value_b'}},
        field_y: Integer(),
        field_z: {'ArtifactB': {'path_type': 'file', 'file_type':'tsv'}}) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_with_openapi_schema(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': 'GCSPath', 'field_n': {'customized_type': {'openapi_schema_validator': '{"type": "string", "pattern": "^gs://.*$"}'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type': {'openapi_schema_validator': '{"type": "string", "pattern": "^gs://.*$"}'}},
        field_y: Integer(),
        field_z: GCSPath()) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

  def test_type_check_with_ignore_type(self):
    """Test type check at the decorator."""
    kfp.TYPE_CHECK = True
    @component
    def a_op(field_l: Integer()) -> {'field_m': 'GCSPath', 'field_n': {'customized_type': {'openapi_schema_validator': '{"type": "string", "pattern": "^gs://.*$"}'}}, 'field_o': 'Integer'}:
      return ContainerOp(
          name = 'operator a',
          image = 'gcr.io/ml-pipeline/component-b',
          arguments = [
              '--field-l', field_l,
          ],
          file_outputs = {
              'field_m': '/schema.txt',
              'field_n': '/feature.txt',
              'field_o': '/output.txt'
          }
      )

    @component
    def b_op(field_x: {'customized_type': {'openapi_schema_validator': '{"type": "string", "pattern": "^gcs://.*$"}'}},
        field_y: Integer(),
        field_z: GCSPath()) -> {'output_model_uri': 'GcsUri'}:
      return ContainerOp(
          name = 'operator b',
          image = 'gcr.io/ml-pipeline/component-a',
          command = [
              'python3',
              field_x,
          ],
          arguments = [
              '--field-y', field_y,
              '--field-z', field_z,
          ],
          file_outputs = {
              'output_model_uri': '/schema.txt',
          }
      )

    a = a_op(field_l=12)
    with self.assertRaises(InconsistentTypeException):
      b = b_op(field_x=a.outputs['field_n'], field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])
    b = b_op(field_x=a.outputs['field_n'].ignore_type(), field_y=a.outputs['field_o'], field_z=a.outputs['field_m'])

class TestGraphComponent(unittest.TestCase):

  def test_graphcomponent_basic(self):
    """Test graph_component decorator metadata."""
    @graph_component
    def flip_component(flip_result):
      with dsl.Condition(flip_result == 'heads'):
        flip_component(flip_result)

    with Pipeline('pipeline') as p:
      param = PipelineParam(name='param')
      flip_component(param)
      self.assertEqual(1, len(p.groups))
      self.assertEqual(1, len(p.groups[0].groups)) # pipeline
      self.assertEqual(1, len(p.groups[0].groups[0].groups)) # flip_component
      self.assertEqual(1, len(p.groups[0].groups[0].groups[0].groups)) # condition
      self.assertEqual(0, len(p.groups[0].groups[0].groups[0].groups[0].groups)) # recursive flip_component
      recursive_group = p.groups[0].groups[0].groups[0].groups[0]
      self.assertTrue(recursive_group.recursive_ref is not None)
      self.assertEqual(1, len(recursive_group.inputs))
      self.assertEqual('param', recursive_group.inputs[0].name)
