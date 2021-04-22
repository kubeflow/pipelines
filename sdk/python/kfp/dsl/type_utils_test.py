# Copyright 2020 Google LLC
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

from absl.testing import parameterized

import sys
import unittest
from kfp.components import structures
from kfp.dsl import io_types
from kfp.dsl import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2 as pb

_PARAMETER_TYPES = ['String', 'str', 'Integer', 'int', 'Float', 'Double']
_KNOWN_ARTIFACT_TYPES = ['Model', 'Dataset', 'Schema', 'Metrics']
_UNKNOWN_ARTIFACT_TYPES = [{
    'JsonObject': {
        'data_type': 'proto:tfx.components.trainer.Trai'
    }
}, None, 'Arbtrary Model', 'dummy']


class _ArbitraryClass:
  pass


class TypeUtilsTest(parameterized.TestCase):

  def test_is_parameter_type(self):
    for type_name in _PARAMETER_TYPES:
      self.assertTrue(type_utils.is_parameter_type(type_name))
    for type_name in _KNOWN_ARTIFACT_TYPES + _UNKNOWN_ARTIFACT_TYPES:
      self.assertFalse(type_utils.is_parameter_type(type_name))

  @parameterized.parameters(
    {
        'artifact_class_or_type_name': 'Model',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Model')
    },
    {
        'artifact_class_or_type_name': io_types.Model,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Model')
    },
    {
        'artifact_class_or_type_name': 'Dataset',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Dataset')
    },
    {
        'artifact_class_or_type_name': io_types.Dataset,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Dataset')
    },
    {
        'artifact_class_or_type_name': 'Metrics',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Metrics')
    },
    {
        'artifact_class_or_type_name': io_types.Metrics,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Metrics')
    },
    {
        'artifact_class_or_type_name': 'ClassificationMetrics',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.ClassificationMetrics')
    },
    {
        'artifact_class_or_type_name': io_types.ClassificationMetrics,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.ClassificationMetrics')
    },
    {
        'artifact_class_or_type_name': 'SlicedClassificationMetrics',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.SlicedClassificationMetrics')
    },
    {
        'artifact_class_or_type_name': io_types.SlicedClassificationMetrics,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.SlicedClassificationMetrics')
    },
    {
        'artifact_class_or_type_name': 'arbitrary name',
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Artifact')
    },
    {
        'artifact_class_or_type_name': _ArbitraryClass,
        'expected_result': pb.ArtifactTypeSchema(schema_title='system.Artifact')
    },
  )
  def test_get_artifact_type_schema(self, artifact_class_or_type_name, expected_result):
    self.assertEqual(expected_result,
                    type_utils.get_artifact_type_schema(
                        artifact_class_or_type_name))

  def test_get_parameter_type(self):
    # Test get parameter type by name.
    self.assertEqual(pb.PrimitiveType.INT, type_utils.get_parameter_type('Int'))
    self.assertEqual(pb.PrimitiveType.INT,
                     type_utils.get_parameter_type('Integer'))
    self.assertEqual(pb.PrimitiveType.DOUBLE,
                     type_utils.get_parameter_type('Double'))
    self.assertEqual(pb.PrimitiveType.DOUBLE,
                     type_utils.get_parameter_type('Float'))
    self.assertEqual(pb.PrimitiveType.STRING,
                     type_utils.get_parameter_type('String'))
    self.assertEqual(pb.PrimitiveType.STRING,
                     type_utils.get_parameter_type('Str'))

    # Test get parameter by Python type.
    self.assertEqual(pb.PrimitiveType.INT, type_utils.get_parameter_type(int))
    self.assertEqual(pb.PrimitiveType.DOUBLE,
                     type_utils.get_parameter_type(float))
    self.assertEqual(pb.PrimitiveType.STRING,
                     type_utils.get_parameter_type(str))

    with self.assertRaises(AttributeError):
      type_utils.get_parameter_type_schema(None)

    with self.assertRaisesRegex(TypeError, 'Got illegal parameter type.'):
      type_utils.get_parameter_type(bool)

  def test_get_input_artifact_type_schema(self):
    input_specs = [
        structures.InputSpec(name='input1', type='String'),
        structures.InputSpec(name='input2', type='Model'),
        structures.InputSpec(name='input3', type=None),
    ]
    # input not found.
    with self.assertRaises(AssertionError) as cm:
      type_utils.get_input_artifact_type_schema('input0', input_specs)
      self.assertEqual('Input not found.', str(cm))

    # input found, but it doesn't map to an artifact type.
    with self.assertRaises(AssertionError) as cm:
      type_utils.get_input_artifact_type_schema('input1', input_specs)
      self.assertEqual('Input is not an artifact type.', str(cm))

    # input found, and a matching artifact type schema returned.
    self.assertEqual(
        'system.Model',
        type_utils.get_input_artifact_type_schema('input2', input_specs).schema_title)

    # input found, and the default artifact type schema returned.
    self.assertEqual(
        'system.Artifact',
        type_utils.get_input_artifact_type_schema('input3', input_specs).schema_title)

  def test_get_parameter_type_field_name(self):
    self.assertEqual('string_value',
                     type_utils.get_parameter_type_field_name('String'))
    self.assertEqual('int_value',
                     type_utils.get_parameter_type_field_name('Integer'))
    self.assertEqual('double_value',
                     type_utils.get_parameter_type_field_name('Float'))


if __name__ == '__main__':
  unittest.main()
