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
# limitations under the License.

import os
import re
import sys
import tempfile
from typing import Any, Callable, Dict, List, Union
import unittest

from absl.testing import parameterized
from kfp import compiler
from kfp import components
from kfp.dsl import placeholders
from kfp.dsl import python_component
from kfp.dsl import structures
import yaml

_PROJECT_ROOT = os.path.abspath(os.path.join(__file__, *([os.path.pardir] * 5)))


def create_test_cases() -> List[Dict[str, Any]]:
    parameters: List[Dict[str, Any]] = []
    config_path = os.path.join(_PROJECT_ROOT, 'sdk', 'python', 'test_data',
                               'test_data_config.yaml')
    with open(config_path) as f:
        config = yaml.safe_load(f)
    for name, test_group in config.items():
        test_data_dir = os.path.join(_PROJECT_ROOT, test_group['test_data_dir'])

        parameters.extend({
            'name': name + '-' + test_case['module'],
            'test_case': test_case['module'],
            'test_data_dir': test_data_dir,
            'read': test_group['read'],
            'write': test_group['write'],
            'function': test_case['name']
        } for test_case in test_group['test_cases'])

    return parameters


def import_obj_from_file(python_path: str, obj_name: str) -> Any:
    sys.path.insert(0, os.path.dirname(python_path))
    module_name = os.path.splitext(os.path.split(python_path)[1])[0]
    module = __import__(module_name, fromlist=[obj_name])
    if not hasattr(module, obj_name):
        raise ValueError(
            f'Object "{obj_name}" not found in module {python_path}.')
    return getattr(module, obj_name)


def ignore_kfp_version_helper(spec: Dict[str, Any]) -> Dict[str, Any]:
    """Ignores kfp sdk versioning in command.

    Takes in a YAML input and ignores the kfp sdk versioning in command
    for comparison between compiled file and goldens.
    """
    pipeline_spec = spec.get('pipelineSpec', spec)

    if 'executors' in pipeline_spec['deploymentSpec']:
        for executor in pipeline_spec['deploymentSpec']['executors']:
            pipeline_spec['deploymentSpec']['executors'][
                executor] = yaml.safe_load(
                    re.sub(
                        r"'kfp==(\d+).(\d+).(\d+)(-[a-z]+.\d+)?'", 'kfp',
                        yaml.dump(
                            pipeline_spec['deploymentSpec']['executors']
                            [executor],
                            sort_keys=True)))
    return spec


def load_compiled_file(filename: str) -> Dict[str, Any]:
    with open(filename) as f:
        contents = yaml.safe_load(f)
        pipeline_spec = contents[
            'pipelineSpec'] if 'pipelineSpec' in contents else contents
        # ignore the sdkVersion
        del pipeline_spec['sdkVersion']
        return ignore_kfp_version_helper(contents)


def handle_placeholders(
        component_spec: structures.ComponentSpec) -> structures.ComponentSpec:
    if component_spec.implementation.container is not None:
        if component_spec.implementation.container.command is not None:
            component_spec.implementation.container.command = [
                placeholders.convert_command_line_element_to_string(c)
                for c in component_spec.implementation.container.command
            ]
        if component_spec.implementation.container.args is not None:
            component_spec.implementation.container.args = [
                placeholders.convert_command_line_element_to_string(a)
                for a in component_spec.implementation.container.args
            ]
    return component_spec


def handle_expected_diffs(
        component_spec: structures.ComponentSpec) -> structures.ComponentSpec:
    """Strips some component spec fields that should be ignored when comparing
    with golden result."""
    # Ignore description when comparing components specs read in from v1 component YAML and from IR YAML, because non lightweight Python components defined in v1 YAML can have a description field, but IR YAML does not preserve this field unless the component is a lightweight Python function-based component
    component_spec.description = None
    # ignore SDK version so that golden snapshots don't need to be updated between SDK version bump
    if component_spec.implementation.graph is not None:
        component_spec.implementation.graph.sdk_version = ''

    return handle_placeholders(component_spec)


class ReadWriteTest(parameterized.TestCase):

    def _compile_and_load_component(
        self, compilable: Union[Callable[..., Any],
                                python_component.PythonComponent]):
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_file = os.path.join(tmp_dir, 're_compiled_output.yaml')
            compiler.Compiler().compile(compilable, tmp_file)
            return components.load_component_from_file(tmp_file)

    def _compile_and_read_yaml(
        self, compilable: Union[Callable[..., Any],
                                python_component.PythonComponent]):
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_file = os.path.join(tmp_dir, 're_compiled_output.yaml')
            compiler.Compiler().compile(compilable, tmp_file)
            return load_compiled_file(tmp_file)

    def _test_serialization_deserialization_consistency(self, yaml_file: str):
        """Tests serialization and deserialization consistency."""
        original_component = components.load_component_from_file(yaml_file)
        reloaded_component = self._compile_and_load_component(
            original_component)
        self.assertEqual(
            handle_expected_diffs(original_component.component_spec),
            handle_expected_diffs(reloaded_component.component_spec),
            f'\n\n\nError with (de)serialization consistency of: {yaml_file}')

    def _test_serialization_correctness(
        self,
        python_file: str,
        yaml_file: str,
        function_name: str,
    ):
        """Tests serialization correctness."""
        pipeline = import_obj_from_file(python_file, function_name)
        compiled_result = self._compile_and_read_yaml(pipeline)
        golden_result = load_compiled_file(yaml_file)
        self.assertEqual(compiled_result, golden_result,
                         f'\n\n\nError with compiling: {python_file}')

    @parameterized.parameters(create_test_cases())
    def test(
        self,
        name: str,
        test_case: str,
        test_data_dir: str,
        function: str,
        read: bool,
        write: bool,
    ):
        """Tests serialization and deserialization consistency and correctness.

        Args:
            name: '{test_group_name}-{test_case_name}'. Useful for print statements/debugging.
            test_case: Test case name (without file extension).
            test_data_dir: The directory containing the test case files.
            function: The function name to compile.
            read: Whether the pipeline/component supports deserialization from YAML (IR, except for V1 component YAML back compatability tests).
            write: Whether the pipeline/component supports compilation from a Python file.
        """
        yaml_file = os.path.join(test_data_dir, f'{test_case}.yaml')
        py_file = os.path.join(test_data_dir, f'{test_case}.py')
        if write:
            self._test_serialization_correctness(
                python_file=py_file,
                yaml_file=yaml_file,
                function_name=function)

        if read:
            self._test_serialization_deserialization_consistency(
                yaml_file=yaml_file)


if __name__ == '__main__':
    unittest.main()
