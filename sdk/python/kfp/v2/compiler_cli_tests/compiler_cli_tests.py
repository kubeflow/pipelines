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

import json
import os
import shutil
import subprocess
import tempfile
import unittest
import kfp


class CompilerCliTests(unittest.TestCase):

  def _test_compile_py_to_json(self, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'test_data')
    py_file = os.path.join(test_data_dir, '{}.py'.format(file_base_name))
    tmpdir = tempfile.mkdtemp()
    try:
      target_json = os.path.join(tmpdir, file_base_name + '-pipeline.json')
      subprocess.check_call(
          ['dsl-compile-v2', '--py', py_file, '--output', target_json])
      with open(os.path.join(test_data_dir, file_base_name + '.json'),
                'r') as f:
        golden = json.load(f)
        # Correct the sdkVersion
        golden['sdkVersion'] = 'kfp-{}'.format(kfp.__version__)
        # Need to sort the list items before comparison
        golden['tasks'].sort(key=lambda x: x['executorLabel'])

      with open(os.path.join(test_data_dir, target_json), 'r') as f:
        compiled = json.load(f)
        # Need to sort the list items before comparison
        compiled['tasks'].sort(key=lambda x: x['executorLabel'])

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def test_two_step_pipeline_with_importer(self):
    self._test_compile_py_to_json('two_step_pipeline_with_importer')

  def test_simple_pipeline_without_importer(self):
    self._test_compile_py_to_json('simple_pipeline_without_importer')

  def test_pipeline_with_ontology(self):
    self._test_compile_py_to_json('pipeline_with_ontology')

  def test_pipeline_with_if_placeholder(self):
    self._test_compile_py_to_json('pipeline_with_if_placeholder')

  def test_pipeline_with_concat_placeholder(self):
    self._test_compile_py_to_json('pipeline_with_concat_placeholder')

  def test_pipeline_with_resource_spec(self):
    self._test_compile_py_to_json('pipeline_with_resource_spec')

  def test_pipeline_with_various_io_types(self):
    self._test_compile_py_to_json('pipeline_with_various_io_types')


if __name__ == '__main__':
  unittest.main()
