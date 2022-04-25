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

import os
import re
import shutil
import subprocess
import tempfile
import unittest

import yaml


def _ignore_kfp_version_helper(spec):
    """Ignores kfp sdk versioning in command.

    Takes in a YAML input and ignores the kfp sdk versioning in command
    for comparison between compiled file and goldens.
    """
    pipeline_spec = spec['pipelineSpec'] if 'pipelineSpec' in spec else spec

    if 'executors' in pipeline_spec['deploymentSpec']:
        for executor in pipeline_spec['deploymentSpec']['executors']:
            pipeline_spec['deploymentSpec']['executors'][executor] = yaml.load(
                re.sub(
                    "'kfp==(\d+).(\d+).(\d+)(-[a-z]+.\d+)?'", 'kfp',
                    yaml.dump(
                        pipeline_spec['deploymentSpec']['executors'][executor],
                        sort_keys=True)))
    return spec


class CompilerCliTests(unittest.TestCase):

    def setUp(self) -> None:
        self.maxDiff = None
        return super().setUp()

    def _test_compile_py_to_yaml(
        self,
        file_base_name,
        additional_arguments=None,
    ):
        test_data_dir = os.path.join(os.path.dirname(__file__), 'test_data')
        py_file = os.path.join(test_data_dir, '{}.py'.format(file_base_name))
        golden_compiled_file = os.path.join(test_data_dir,
                                            file_base_name + '.yaml')

        if additional_arguments is None:
            additional_arguments = []

        def _compile(target_output_file: str):
            try:
                subprocess.check_output(
                    ['dsl-compile', '--py', py_file, '--output', compiled_file]
                    + additional_arguments,
                    stderr=subprocess.STDOUT)
            except subprocess.CalledProcessError as exc:
                raise Exception(exc.output) from exc

        def _load_compiled_file(filename: str):
            with open(filename, 'r') as f:
                contents = yaml.safe_load(f)
                # Correct the sdkVersion
                pipeline_spec = contents[
                    'pipelineSpec'] if 'pipelineSpec' in contents else contents
                del pipeline_spec['sdkVersion']
                return _ignore_kfp_version_helper(contents)

        with tempfile.TemporaryDirectory() as tmpdir:
            compiled_file = os.path.join(tmpdir,
                                         file_base_name + '-pipeline.yaml')
            _compile(target_output_file=compiled_file)

            golden = _load_compiled_file(golden_compiled_file)
            compiled = _load_compiled_file(compiled_file)

            # Devs can run the following command to update golden files:
            # UPDATE_GOLDENS=True python3 -m unittest kfp/v2/compiler_cli_tests/compiler_cli_tests.py
            # If UPDATE_GOLDENS=True, and the diff is
            # different, update the golden file and reload it.
            update_goldens = os.environ.get('UPDATE_GOLDENS', False)
            if golden != compiled and update_goldens:
                _compile(target_output_file=golden_compiled_file)
                golden = _load_compiled_file(golden_compiled_file)

            self.assertEqual(golden, compiled)

    def test_two_step_pipeline(self):
        self._test_compile_py_to_yaml(
            'two_step_pipeline',
            ['--pipeline-parameters', '{"text":"Hello KFP!"}'])

    def test_two_step_pipeline_failure_parameter_parse(self):
        with self.assertRaisesRegex(
                Exception, r"Failed to parse --pipeline-parameters argument:"):
            self._test_compile_py_to_yaml(
                'two_step_pipeline',
                ['--pipeline-parameters', '{"text":"Hello KFP!}'])

    def test_pipeline_with_importer(self):
        self._test_compile_py_to_yaml('pipeline_with_importer')

    def test_pipeline_with_ontology(self):
        self._test_compile_py_to_yaml('pipeline_with_ontology')

    def test_pipeline_with_if_placeholder(self):
        self._test_compile_py_to_yaml('pipeline_with_if_placeholder')

    def test_pipeline_with_concat_placeholder(self):
        self._test_compile_py_to_yaml('pipeline_with_concat_placeholder')

    def test_pipeline_with_resource_spec(self):
        self._test_compile_py_to_yaml('pipeline_with_resource_spec')

    def test_pipeline_with_various_io_types(self):
        self._test_compile_py_to_yaml('pipeline_with_various_io_types')

    def test_pipeline_with_reused_component(self):
        self._test_compile_py_to_yaml('pipeline_with_reused_component')

    def test_pipeline_with_after(self):
        self._test_compile_py_to_yaml('pipeline_with_after')

    def test_pipeline_with_condition(self):
        self._test_compile_py_to_yaml('pipeline_with_condition')

    def test_pipeline_with_nested_conditions(self):
        self._test_compile_py_to_yaml('pipeline_with_nested_conditions')

    def test_pipeline_with_nested_conditions_yaml(self):
        self._test_compile_py_to_yaml('pipeline_with_nested_conditions_yaml')

    def test_pipeline_with_loops(self):
        self._test_compile_py_to_yaml('pipeline_with_loops')

    def test_pipeline_with_nested_loops(self):
        self._test_compile_py_to_yaml('pipeline_with_nested_loops')

    def test_pipeline_with_loops_and_conditions(self):
        self._test_compile_py_to_yaml('pipeline_with_loops_and_conditions')

    def test_pipeline_with_params_containing_format(self):
        self._test_compile_py_to_yaml('pipeline_with_params_containing_format')

    def test_lightweight_python_functions_v2_pipeline(self):
        self._test_compile_py_to_yaml(
            'lightweight_python_functions_v2_pipeline')

    def test_lightweight_python_functions_v2_with_outputs(self):
        self._test_compile_py_to_yaml(
            'lightweight_python_functions_v2_with_outputs')

    def test_xgboost_sample_pipeline(self):
        self._test_compile_py_to_yaml('xgboost_sample_pipeline')

    def test_pipeline_with_metrics_outputs(self):
        self._test_compile_py_to_yaml('pipeline_with_metrics_outputs')

    def test_pipeline_with_exit_handler(self):
        self._test_compile_py_to_yaml('pipeline_with_exit_handler')

    def test_pipeline_with_env(self):
        self._test_compile_py_to_yaml('pipeline_with_env')

    def test_v2_component_with_optional_inputs(self):
        self._test_compile_py_to_yaml('v2_component_with_optional_inputs')

    def test_pipeline_with_gcpc_types(self):
        self._test_compile_py_to_yaml('pipeline_with_gcpc_types')

    def test_pipeline_with_placeholders(self):
        self._test_compile_py_to_yaml('pipeline_with_placeholders')

    def test_pipeline_with_task_final_status(self):
        self._test_compile_py_to_yaml('pipeline_with_task_final_status')

    def test_pipeline_with_task_final_status_yaml(self):
        self._test_compile_py_to_yaml('pipeline_with_task_final_status_yaml')

    def test_v2_component_with_pip_index_urls(self):
        self._test_compile_py_to_yaml('v2_component_with_pip_index_urls')


if __name__ == '__main__':
    unittest.main()
