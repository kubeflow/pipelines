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

from kfp.dsl.types import check_types, GCSPath
import unittest

class TestTypes(unittest.TestCase):

  def test_class_to_dict(self):
    """Test _class_to_dict function."""
    gcspath_dict = GCSPath().to_dict()
    golden_dict = {
        'GCSPath': {
          'openapi_schema_validator': {
              "type": "string",
              "pattern": "^gs://.*$"
          }
        }
    }
    self.assertEqual(golden_dict, gcspath_dict)

  def test_check_types(self):
    #Core types
    typeA = {'ArtifactA': {'path_type': 'file', 'file_type':'csv'}}
    typeB = {'ArtifactA': {'path_type': 'file', 'file_type':'csv'}}
    self.assertTrue(check_types(typeA, typeB))
    typeC = {'ArtifactA': {'path_type': 'file', 'file_type':'tsv'}}
    self.assertFalse(check_types(typeA, typeC))

    # Custom types
    typeA = {
        'A':{
            'X': 'value1',
            'Y': 'value2'
        }
    }
    typeB = {
        'B':{
            'X': 'value1',
            'Y': 'value2'
        }
    }
    typeC = {
        'A':{
            'X': 'value1'
        }
    }
    typeD = {
        'A':{
            'X': 'value1',
            'Y': 'value3'
        }
    }
    self.assertFalse(check_types(typeA, typeB))
    self.assertFalse(check_types(typeA, typeC))
    self.assertTrue(check_types(typeC, typeA))
    self.assertFalse(check_types(typeA, typeD))
