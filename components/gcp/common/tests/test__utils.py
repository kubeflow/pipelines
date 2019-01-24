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

from gcp_common import normalize_name
import unittest

class UtilsTest(unittest.TestCase):

    def test_normalize_name_valid_name_unchanged(self):
        name = 'validname'

        normalized_name = normalize_name(name)

        self.assertEqual(name, normalized_name)

    def test_normalize_name_replace_invalid_char(self):
        name = 'invalid-name' # - is an invalid char

        normalized_name = normalize_name(name)

        self.assertEqual('invalid_name', normalized_name)

    def test_normalize_name_add_prefix(self):
        name = '9invalid-name' # 9 is invalid in first char.

        normalized_name = normalize_name(name)

        self.assertEqual('x_9invalid_name', normalized_name)  
