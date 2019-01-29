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

from gcp_common import validators
import unittest

class ValidatorsTest(unittest.TestCase):

    def test_validate_project_id_valid(self):
        self.assertEqual('project-1', 
            validators.validate_project_id('project-1'))

    def test_validate_project_id_invalid_char(self):
        self._assert_value_error(
            lambda: validators.validate_project_id('$'))

    def test_validate_gcs_path_valid(self):
        valid_value = 'gs://bucket/path'
        self.assertEqual(valid_value, 
            validators.validate_gcs_path(valid_value, 'gcs path'))

    def test_validate_gcs_path_invalid_char(self):
        self._assert_value_error(
            lambda: validators.validate_gcs_path(
                '/invalid/path', 'gcs path'))

    def test_validate_required_valid_values(self):
        self.assertEqual('project-1', 
            validators.validate_required('project-1', 'string'))
        self.assertEqual(False, 
            validators.validate_required(False, 'flag'))
        self.assertEqual(0, 
            validators.validate_required(0, 'int'))

    def test_validate_required_invalid_values(self):
        self._assert_value_error(
            lambda: validators.validate_required(
                '', 'empty'))
        self._assert_value_error(
            lambda: validators.validate_required(
                None, 'none'))

    def test_validate_list_valid_values(self):
        self.assertEqual([], 
            validators.validate_list([], 'empty list'))
        self.assertEqual(['e1'], 
            validators.validate_list(['e1'], 'string list'))

    def test_validate_list_invalid_values(self):
        self._assert_value_error(
            lambda: validators.validate_list(
                {}, 'dict'))
        self._assert_value_error(
            lambda: validators.validate_list(
                None, 'none'))
    
    def test_validate_dict_valid_values(self):
        self.assertEqual({}, 
            validators.validate_dict({}, 'empty dict'))
        self.assertEqual({'key': {}}, 
            validators.validate_dict({'key': {}}, 'nested dict'))

    def test_validate_dict_invalid_values(self):
        self._assert_value_error(
            lambda: validators.validate_dict(
                [], 'list'))
        self._assert_value_error(
            lambda: validators.validate_dict(
                None, 'none'))
    
    def _assert_value_error(self, func):
        with self.assertRaises(ValueError):
            func()