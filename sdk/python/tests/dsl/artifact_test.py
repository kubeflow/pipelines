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
"""Tests for kfp.dsl.artifact module."""
import unittest
import textwrap

from kfp.dsl import artifact


class _MyArtifact(artifact.Artifact):
  TYPE_NAME = 'MyTypeName'
  PROPERTIES = {
      'int1': artifact.Property(
          type=artifact.PropertyType.INT,
          description='An integer-typed property'),
      'int2': artifact.Property(type=artifact.PropertyType.INT),
      'float1': artifact.Property(
          type=artifact.PropertyType.DOUBLE,
          description='A float-typed property'),
      'float2': artifact.Property(type=artifact.PropertyType.DOUBLE),
      'string1': artifact.Property(
          type=artifact.PropertyType.STRING,
          description='A string-typed property'),
      'string2': artifact.Property(type=artifact.PropertyType.STRING),
  }


_SERIALIZED_INSTANCE = """\
{
  "properties": {
    "float1": {
      "doubleValue": 1.11
    },
    "int1": {
      "intValue": "1"
    },
    "string1": {
      "stringValue": "111"
    }
  },
  "type": {
    "instanceSchema": "properties:\\n  float1:\\n    description: A float-typed property\\n    type: double\\n  float2:\\n    description:\\n    type: double\\n  int1:\\n    description: An integer-typed property\\n    type: int\\n  int2:\\n    description:\\n    type: int\\n  string1:\\n    description: A string-typed property\\n    type: string\\n  string2:\\n    description:\\n    type: string\\ntitle: kfp.MyTypeName\\ntype: object\\n"
  }
}"""


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

    self.assertEqual(textwrap.dedent("""\
        Artifact(artifact: name: "1"
        type {
          instance_schema: "properties:\\n  float1:\\n    description: A float-typed property\\n    type: double\\n  float2:\\n    description:\\n    type: double\\n  int1:\\n    description: An integer-typed property\\n    type: int\\n  int2:\\n    description:\\n    type: int\\n  string1:\\n    description: A string-typed property\\n    type: string\\n  string2:\\n    description:\\n    type: string\\ntitle: kfp.MyTypeName\\ntype: object\\n"
        }
        uri: "/tmp/uri2"
        custom_properties {
          key: "int_key"
          value {
            int_value: 20
          }
        }
        custom_properties {
          key: "string_key"
          value {
            string_value: "string_value"
          }
        }
        , type_schema: properties:
          float1:
            description: A float-typed property
            type: double
          float2:
            description:
            type: double
          int1:
            description: An integer-typed property
            type: int
          int2:
            description:
            type: int
          string1:
            description: A string-typed property
            type: string
          string2:
            description:
            type: string
        title: kfp.MyTypeName
        type: object
        )"""), str(instance))

  def testArtifactProperties(self):
    my_artifact = _MyArtifact()

    self.assertEqual(0, my_artifact.int1)
    self.assertEqual(0, my_artifact.int2)
    my_artifact.int1 = 111
    my_artifact.int2 = 222
    self.assertEqual('', my_artifact.string1)
    self.assertEqual('', my_artifact.string2)
    my_artifact.string1 = '111'
    my_artifact.string2 = '222'
    self.assertEqual(0.0, my_artifact.float1)
    self.assertEqual(0.0, my_artifact.float2)
    my_artifact.float1 = 1.11
    my_artifact.float2 = 2.22
    self.assertEqual(my_artifact.int1, 111)
    self.assertEqual(my_artifact.int2, 222)
    self.assertEqual(my_artifact.string1, '111')
    self.assertEqual(my_artifact.string2, '222')
    self.assertEqual(1.11, my_artifact.float1)
    self.assertEqual(2.22, my_artifact.float2)
    self.assertEqual(my_artifact.get_string_custom_property('invalid'), '')
    self.assertEqual(my_artifact.get_int_custom_property('invalid'), 0)
    self.assertNotIn('invalid', my_artifact._artifact.custom_properties)

    with self.assertRaisesRegex(
        AttributeError, "Cannot set unknown property 'invalid' on artifact"):
      my_artifact.invalid = 1

    with self.assertRaisesRegex(
        AttributeError, "Cannot set unknown property 'invalid' on artifact"):
      my_artifact.invalid = 'x'

    with self.assertRaisesRegex(AttributeError,
                                "\D+ artifact has no property 'invalid'"):
      _ = my_artifact.invalid

  def testSerialize(self):
    instance = _MyArtifact()
    instance.int1 = 1
    instance.string1 = '111'
    instance.float1 = 1.11

    self.assertEqual(_SERIALIZED_INSTANCE, instance.serialize())

  def testDeserialize(self):
    instance = artifact.Artifact.deserialize(_SERIALIZED_INSTANCE)
    self.assertEqual(1, instance.int1)
    self.assertEqual('111', instance.string1)
    self.assertEqual(1.11, instance.float1)
    self.assertEqual('kfp.MyTypeName', instance.type_name)
