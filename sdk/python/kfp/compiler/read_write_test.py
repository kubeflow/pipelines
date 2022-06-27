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
import types
from typing import Any, Callable, Dict, List, Optional, Union
import unittest

from absl.testing import parameterized
from kfp import compiler
from kfp import components
from kfp.compiler._read_write_test_config import CONFIG
from kfp.components import pipeline_context
from kfp.components import python_component
from kfp.components import structures
import yaml

PROJECT_ROOT = os.path.abspath(os.path.join(__file__, *([os.path.pardir] * 5)))


def expand_config(config: dict) -> List[Dict[str, Any]]:
    parameters: List[Dict[str, Any]] = []
    for name, test_group in config.items():
        test_data_dir = os.path.join(PROJECT_ROOT, test_group['test_data_dir'])
        config = test_group['config']
        parameters.extend({
            'name': name + '-' + test_case,
            'test_case': test_case,
            'test_data_dir': test_data_dir,
            'read': config['read'],
            'write': config['write'],
            'function': None if name == 'pipelines' else test_case,
        } for test_case in test_group['test_cases'])

    return parameters


def collect_pipeline_from_module(
    target_module: types.ModuleType
) -> Union[Callable[..., Any], python_component.PythonComponent]:
    pipelines = []
    module_attrs = dir(target_module)
    for attr in module_attrs:
        obj = getattr(target_module, attr)
        if pipeline_context.Pipeline.is_pipeline_func(obj):
            pipelines.append(obj)
    if len(pipelines) == 1:
        return pipelines[0]
    else:
        raise ValueError(
            f'Expect one pipeline function in module {target_module}, got {len(pipelines)}: {pipelines}. Please specify the pipeline function name with --function.'
        )


def collect_pipeline_func(
    python_file: str,
    function_name: Optional[str] = None
) -> Union[Callable[..., Any], python_component.PythonComponent]:
    sys.path.insert(0, os.path.dirname(python_file))
    try:
        filename = os.path.basename(python_file)
        module_name = os.path.splitext(filename)[0]
        if function_name is None:
            return collect_pipeline_from_module(
                target_module=__import__(module_name))

        module = __import__(module_name, fromlist=[function_name])
        if not hasattr(module, function_name):
            raise ValueError(
                f'Pipeline function or component "{function_name}" not found in module {filename}.'
            )

        return getattr(module, function_name)

    finally:
        del sys.path[0]


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


def set_description_in_component_spec_to_none(
        component_spec: structures.ComponentSpec) -> structures.ComponentSpec:
    """Sets the description field of a ComponentSpec to None."""
    # Ignore description when comparing components specs read in from v1 component YAML and from IR YAML, because non lightweight Python components defined in v1 YAML can have a description field, but IR YAML does not preserve this field unless the component is a lightweight Python function-based component
    component_spec.description = None
    return component_spec


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
            set_description_in_component_spec_to_none(
                original_component.component_spec),
            set_description_in_component_spec_to_none(
                reloaded_component.component_spec))

    def _test_serialization_correctness(self,
                                        python_file: str,
                                        yaml_file: str,
                                        function_name: Optional[str] = None):
        """Tests serialization correctness."""
        pipeline = collect_pipeline_func(
            python_file, function_name=function_name)
        compiled_result = self._compile_and_read_yaml(pipeline)
        golden_result = load_compiled_file(yaml_file)
        self.assertEqual(compiled_result, golden_result)

    @parameterized.parameters(expand_config((CONFIG)))
    def test(
        self,
        name: str,
        test_case: str,
        test_data_dir: str,
        function: Optional[str],
        read: bool,
        write: bool,
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
