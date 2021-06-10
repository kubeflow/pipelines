# Copyright 2020 The Kubeflow Authors
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

  def _test_compile_py_to_json(self,
                               file_base_name,
                               additional_arguments=[],
                               update_golden: bool = False):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'test_data')
    py_file = os.path.join(test_data_dir, '{}.py'.format(file_base_name))
    tmpdir = tempfile.mkdtemp()
    golden_compiled_file = os.path.join(test_data_dir, file_base_name + '.json')

    try:
      if update_golden:
        target_json = golden_compiled_file
      else:
        target_json = os.path.join(tmpdir, file_base_name + '-pipeline.json')
      subprocess.check_call([
          'dsl-compile-v2', '--py', py_file, '--output', target_json
      ] + additional_arguments)

      with open(golden_compiled_file, 'r') as f:
        golden = json.load(f)
        # Correct the sdkVersion
        del golden['pipelineSpec']['sdkVersion']

      with open(target_json, 'r') as f:
        compiled = json.load(f)
        del compiled['pipelineSpec']['sdkVersion']

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)

  def test_two_step_pipeline(self):
    self._test_compile_py_to_json(
        'two_step_pipeline',
        ['--pipeline-parameters', '{"text":"Hello KFP!"}'])

  def test_pipeline_with_importer(self):
    self._test_compile_py_to_json('pipeline_with_importer')

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

  def test_pipeline_with_reused_component(self):
    self._test_compile_py_to_json('pipeline_with_reused_component')

  def test_pipeline_with_after(self):
    self._test_compile_py_to_json('pipeline_with_after')

  def test_pipeline_with_condition(self):
    self._test_compile_py_to_json('pipeline_with_condition')

  def test_pipeline_with_nested_conditions(self):
    self._test_compile_py_to_json('pipeline_with_nested_conditions')

  def test_pipeline_with_nested_conditions_yaml(self):
    self._test_compile_py_to_json('pipeline_with_nested_conditions_yaml')

  def test_pipeline_with_loop_static(self):
    self._test_compile_py_to_json('pipeline_with_loop_static')

  def test_pipeline_with_loop_parameter(self):
    self._test_compile_py_to_json('pipeline_with_loop_parameter')

  def test_pipeline_with_loop_output(self):
    self._test_compile_py_to_json('pipeline_with_loop_output')

  def test_pipeline_with_nested_loops(self):
    self._test_compile_py_to_json('pipeline_with_nested_loops')

  def test_pipeline_with_loops_and_conditions(self):
    self._test_compile_py_to_json('pipeline_with_loops_and_conditions')

  def test_pipeline_with_params_containing_format(self):
    self._test_compile_py_to_json('pipeline_with_params_containing_format')

  def test_lightweight_python_functions_v2_pipeline(self):
    self._test_compile_py_to_json('lightweight_python_functions_v2_pipeline')

  def test_lightweight_python_functions_v2_with_outputs(self):
    self._test_compile_py_to_json(
        'lightweight_python_functions_v2_with_outputs')

  def test_xgboost_sample_pipeline(self):
    self._test_compile_py_to_json('xgboost_sample_pipeline')

  def test_pipeline_with_custom_job_spec(self):
    self._test_compile_py_to_json('pipeline_with_custom_job_spec')

  def test_pipeline_with_metrics_outputs(self):
    self._test_compile_py_to_json('pipeline_with_metrics_outputs')

  def test_pipeline_with_exit_handler(self):
    self._test_compile_py_to_json('pipeline_with_exit_handler')


if __name__ == '__main__':
  unittest.main()
