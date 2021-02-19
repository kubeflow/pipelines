# Copyright 2021 Google LLC
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

from kfp.components import _data_passing


class DataPassingTest(unittest.TestCase):

  def test_get_collection_type_name(self):
    self.assertEqual(None, _data_passing._get_collection_type_name('str'))

    self.assertEqual(
        'List', _data_passing._get_collection_type_name('typing.List[int]'))

    self.assertEqual('List',
                     _data_passing._get_collection_type_name('typing.List'))

    self.assertEqual(
        'Dict',
        _data_passing._get_collection_type_name('typing.Dict[str, str]'))

  def test_serialize_value(self):
    self.assertEqual(
        '[1, 2, 3]',
        _data_passing.serialize_value(
            value=[1, 2, 3], type_name='typing.List[int]'))

    self.assertEqual(
        '[4, 5, 6]',
        _data_passing.serialize_value(value=[4, 5, 6], type_name='List'))


if __name__ == '__main__':
  unittest.main()
