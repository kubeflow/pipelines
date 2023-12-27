# Copyright 2023 The Kubeflow Authors
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
from typing import Any, Callable, Dict, List, Tuple, Union
import unittest

from kfp import compiler
from kfp import components
from kfp.components import placeholders
import pytest
import yaml

_TEST_ROOT_DIR = os.path.dirname(__file__)
_CONFIG_PATH = os.path.join(_TEST_ROOT_DIR, 'test_data_config.yaml')
_TEST_DATA_DIR = os.path.join(_TEST_ROOT_DIR, 'data')


def create_test_cases() -> List[Dict[str, Any]]:
    with open(_CONFIG_PATH) as f:
        config = yaml.safe_load(f)
    return [(test_case['module'], test_case['name'])
            for test_case in config['test_cases']]


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


def load_pipeline_spec_and_platform_spec(
        filename: str) -> Tuple[Dict[str, Any], Dict[str, Any]]:
    with open(filename) as f:
        pipeline_spec, platform_spec = tuple(yaml.safe_load_all(f))
        # ignore the sdkVersion
        del pipeline_spec['sdkVersion']
        return ignore_kfp_version_helper(pipeline_spec), platform_spec


def handle_placeholders(
        component_spec: 'structures.ComponentSpec'
) -> 'structures.ComponentSpec':
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
        component_spec: 'structures.ComponentSpec'
) -> 'structures.ComponentSpec':
    """Strips some component spec fields that should be ignored when comparing
    with golden result."""
    # Ignore description when comparing components specs read in from v1 component YAML and from IR YAML, because non lightweight Python components defined in v1 YAML can have a description field, but IR YAML does not preserve this field unless the component is a lightweight Python function-based component
    component_spec.description = None
    # ignore SDK version so that golden snapshots don't need to be updated between SDK version bump
    if component_spec.implementation.graph is not None:
        component_spec.implementation.graph.sdk_version = ''

    return handle_placeholders(component_spec)


class TestReadWrite:

    def _compile_and_load_component(
        self, compilable: Union[Callable[..., Any],
                                'python_component.PythonComponent']):
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_file = os.path.join(tmp_dir, 're_compiled_output.yaml')
            compiler.Compiler().compile(compilable, tmp_file)
            return components.load_component_from_file(tmp_file)

    def _compile_and_read_yaml(
        self, compilable: Union[Callable[..., Any],
                                'python_component.PythonComponent']
    ) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        with tempfile.TemporaryDirectory() as tmp_dir:
            tmp_file = os.path.join(tmp_dir, 're_compiled_output.yaml')
            compiler.Compiler().compile(compilable, tmp_file)
            return load_pipeline_spec_and_platform_spec(tmp_file)

    def _test_serialization_deserialization_consistency(self, yaml_file: str):
        """Tests serialization and deserialization consistency."""
        original_component = components.load_component_from_file(yaml_file)
        reloaded_component = self._compile_and_load_component(
            original_component)
        assert handle_expected_diffs(
            original_component.component_spec) == handle_expected_diffs(
                reloaded_component.component_spec)

    def _test_serialization_correctness(
        self,
        python_file: str,
        yaml_file: str,
        function_name: str,
    ):
        """Tests serialization correctness."""
        pipeline = import_obj_from_file(python_file, function_name)
        compiled_pipeline_spec, compiled_platform_spec = self._compile_and_read_yaml(
            pipeline)
        golden_pipeline_spec, golden_platform_spec = load_pipeline_spec_and_platform_spec(
            yaml_file)
        assert compiled_pipeline_spec == golden_pipeline_spec
        assert compiled_platform_spec == golden_platform_spec

    @pytest.mark.parametrize('test_case,function', create_test_cases())
    def test(
        self,
        test_case: str,
        function: str,
    ):
        """Tests serialization and deserialization consistency and correctness.

        Args:
            name (str): '{test_group_name}-{test_case_name}'. Useful for print statements/debugging.
            test_case (str): Test case name (without file extension).
            test_data_dir (str): The directory containing the test case files.
            function (str, optional): The function name to compile.
            read (bool): Whether the pipeline/component supports deserialization from YAML (IR, except for V1 component YAML back compatability tests).
            write (bool): Whether the pipeline/component supports compilation from a Python file.
        """
        yaml_file = os.path.join(_TEST_DATA_DIR, f'{test_case}.yaml')
        py_file = os.path.join(_TEST_DATA_DIR, f'{test_case}.py')

        self._test_serialization_correctness(
            python_file=py_file, yaml_file=yaml_file, function_name=function)

        self._test_serialization_deserialization_consistency(
            yaml_file=yaml_file)


if __name__ == '__main__':
    unittest.main()
