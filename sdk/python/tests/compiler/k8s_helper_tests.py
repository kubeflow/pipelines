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

from kfp.compiler._k8s_helper import convert_k8s_obj_to_json
from datetime import datetime
import unittest


class TestCompiler(unittest.TestCase):
  def test_convert_k8s_obj_to_dic_accepts_dict(self):
    now = datetime.now()
    converted = convert_k8s_obj_to_json({
      "ENV": "test",
      "number": 3,
      "list": [1,2,3],
      "time": now
    })
    self.assertEqual(converted, {
      "ENV": "test",
      "number": 3,
      "list": [1,2,3],
      "time": now.isoformat()
    })