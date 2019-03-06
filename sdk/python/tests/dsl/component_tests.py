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


from kfp.dsl._component import component
from kfp.dsl._types import GCSPath, Integer
import unittest

@component
def componentA(a: {'Schema': {'file_type': 'csv'}}, b: '{"number": {"step": "large"}}' = 12, c: GCSPath(path_type='file', file_type='tsv') = 'gs://hello/world') -> {'model': Integer()}:
  return 7

class TestPythonComponent(unittest.TestCase):

  def test_component(self):
    """Test component decorator."""
    componentA(1,2,3)