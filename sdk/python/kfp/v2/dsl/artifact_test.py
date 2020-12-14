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
"""Tests for kfp.v2.ds.artifact module."""
import unittest

from kfp.v2.dsl import artifact


class _MyArtifact(artifact.Artifact):
  TYPE_NAME = 'MyTypeName'
  PROPERTIES = {
      'int1': artifact.Property(type=artifact.PropertyType.INT),
      'int2': artifact.Property(type=artifact.PropertyType.INT),
      'float1': artifact.Property(type=artifact.PropertyType.DOUBLE),
      'float2': artifact.Property(type=artifact.PropertyType.DOUBLE),
      'string1': artifact.Property(type=artifact.PropertyType.STRING),
      'string2': artifact.Property(type=artifact.PropertyType.STRING),
  }


class ArtifactTest(unittest.TestCase):

  def testArtifact(self):
    instance = _MyArtifact()

    # Test property getters.
    self.assertEqual('', instance.uri)
    self.assertEqual('', instance.name)

    # Default property does not have span or split_names.
    with self.assertRaisesRegex(AttributeError, "has no property 'span'"):
      _ = instance.span
    with self.assertRaisesRegex(AttributeError,
                                 "has no property 'split_names'"):
      _ = instance.split_names

    # Test property setters.
    instance.uri = '/tmp/uri2'
    self.assertEqual('/tmp/uri2', instance.uri)

    instance.name = '1'
    self.assertEqual('1', instance.name)

    # Testing artifact does not have span.
    with self.assertRaisesRegex(AttributeError, "unknown property 'span'"):
      instance.span = 20190101
    # Testing artifact does not have span.
    with self.assertRaisesRegex(AttributeError,
                                 "unknown property 'split_names'"):
      instance.split_names = ''

    instance.set_int_custom_property('int_key', 20)
    self.assertEqual(
        20, instance.runtime_artifact.custom_properties['int_key'].int_value)

    instance.set_string_custom_property('string_key', 'string_value')
    self.assertEqual(
        'string_value',
        instance.runtime_artifact.custom_properties['string_key'].string_value)

    # self.assertEqual(
    #     'Artifact(artifact: id: 1\n'
    #     'type_id: 2\n'
    #     'uri: "/tmp/uri2"\n'
    #     'custom_properties {\n'
    #     '  key: "int_key"\n'
    #     '  value {\n'
    #     '    int_value: 20\n'
    #     '  }\n'
    #     '}\n'
    #     'custom_properties {\n'
    #     '  key: "state"\n'
    #     '  value {\n'
    #     '    string_value: "deleted"\n'
    #     '  }\n'
    #     '}\n'
    #     'custom_properties {\n'
    #     '  key: "string_key"\n'
    #     '  value {\n'
    #     '    string_value: "string_value"\n'
    #     '  }\n'
    #     '}\n'
    #     ', artifact_type: name: "MyTypeName"\n'
    #     'properties {\n'
    #     '  key: "float1"\n'
    #     '  value: DOUBLE\n'
    #     '}\n'
    #     'properties {\n'
    #     '  key: "float2"\n'
    #     '  value: DOUBLE\n'
    #     '}\n'
    #     'properties {\n'
    #     '  key: "int1"\n'
    #     '  value: INT\n'
    #     '}\n'
    #     'properties {\n'
    #     '  key: "int2"\n'
    #     '  value: INT\n'
    #     '}\n'
    #     'properties {\n'
    #     '  key: "string1"\n'
    #     '  value: STRING\n'
    #     '}\n'
    #     'properties {\n'
    #     '  key: "string2"\n'
    #     '  value: STRING\n'
    #     '}\n'
    #     ')', str(instance))
