# Copyright 2022 The Kubeflow Authors
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
# limitations under the License.# Copyright 2021-2022 The Kubeflow Authors
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

import os
import tempfile
import unittest

import yaml
from kfp.utils import ir_utils

json_dict = {"key": "val", "list": ["1", 2, 3.0]}


def load_from_file(filepath: str) -> str:
    with open(filepath, "r") as f:
        return yaml.safe_load(f)


class TestWriteIrToFile(unittest.TestCase):

    def test_yaml(self):
        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'output.yaml')
            ir_utils._write_ir_to_file(json_dict, temp_filepath)
            actual = load_from_file(temp_filepath)
        self.assertEqual(actual, json_dict)

    def test_yml(self):
        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'output.yml')
            ir_utils._write_ir_to_file(json_dict, temp_filepath)
            actual = load_from_file(temp_filepath)
        self.assertEqual(actual, json_dict)

    def test_json(self):
        with tempfile.TemporaryDirectory() as tempdir, self.assertWarnsRegex(
                DeprecationWarning, r"Compiling to JSON is deprecated"):
            temp_filepath = os.path.join(tempdir, 'output.json')
            ir_utils._write_ir_to_file(json_dict, temp_filepath)
            actual = load_from_file(temp_filepath)
        self.assertEqual(actual, json_dict)

    def test_incorrect_extension(self):
        with tempfile.TemporaryDirectory() as tempdir, self.assertRaisesRegex(
                ValueError, r'should end with "\.yaml"\.'):
            temp_filepath = os.path.join(tempdir, 'output.txt')
            ir_utils._write_ir_to_file(json_dict, temp_filepath)


if __name__ == '__main__':
    unittest.main()
