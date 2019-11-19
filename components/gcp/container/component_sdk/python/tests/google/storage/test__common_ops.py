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

import unittest

from kfp_component.google.storage import is_gcs_path, parse_blob_path

class CommonOpsTest(unittest.TestCase):

    def test_is_gcs_path(self):
        self.assertTrue(is_gcs_path('gs://foo'))
        self.assertTrue(is_gcs_path('gs://foo/bar'))
        self.assertFalse(is_gcs_path('gs:/foo/bar'))
        self.assertFalse(is_gcs_path('foo/bar'))

    def test_parse_blob_path_valid(self):
        bucket_name, blob_name = parse_blob_path('gs://foo/bar/baz/')

        self.assertEqual('foo', bucket_name)
        self.assertEqual('bar/baz/', blob_name)

    def test_parse_blob_path_invalid(self):
        # No blob name
        self.assertRaises(ValueError, lambda: parse_blob_path('gs://foo'))
        self.assertRaises(ValueError, lambda: parse_blob_path('gs://foo/'))

        # Invalid GCS path
        self.assertRaises(ValueError, lambda: parse_blob_path('foo'))
        self.assertRaises(ValueError, lambda: parse_blob_path('gs:///foo'))