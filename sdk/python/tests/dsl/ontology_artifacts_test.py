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
"""Tests for kfp.v2.dsl.ontology_artifacts module."""
import unittest
from kfp.dsl import artifact
from kfp.dsl import ontology_artifacts

_EXPECTED_SERIALIZATION = """\
{
  "customProperties": {
    "float1": {
      "doubleValue": 1.1
    },
    "int1": {
      "intValue": "1"
    },
    "string1": {
      "stringValue": "testString"
    }
  },
  "type": {
    "instanceSchema": "properties:\\ntitle: kfp.Model\\ntype: object\\n"
  }
}"""


class ArtifactsTest(unittest.TestCase):

  def testSerialization(self):
    my_model = ontology_artifacts.Model()
    my_model.set_float_custom_property('float1', 1.1)
    my_model.set_int_custom_property('int1', 1)
    my_model.set_string_custom_property('string1', 'testString')

    self.assertEqual(_EXPECTED_SERIALIZATION, my_model.serialize())

    rehydrated_model = artifact.Artifact.deserialize(_EXPECTED_SERIALIZATION)
    self.assertEqual(ontology_artifacts.Model, type(rehydrated_model))
    self.assertEqual('testString',
                     rehydrated_model.get_string_custom_property('string1'))
    self.assertEqual(1, rehydrated_model.get_int_custom_property('int1'))
    self.assertEqual(1.1, rehydrated_model.get_float_custom_property('float1'))
