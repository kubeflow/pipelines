from kfp import dsl
import yaml


class IfPresentPlaceholder(dsl.IfPresentPlaceholder):

    def __repr__(self) -> str:
        return f'dsl.IfPresentPlaceholder(input_name="{self.input_name}", then={self.then})'


class ConcatPlaceholder(dsl.ConcatPlaceholder):

    def __repr__(self) -> str:
        return f'dsl.ConcatPlaceholder(items={self.items})'


class ArtifactPrinter:

    def __init__(self, key: str, attr: str) -> None:
        self.key = key
        self.attr = attr

    def __repr__(self) -> str:
        return f'{self.key}.{self.attr}'


class OutputPathPrinter:

    def __init__(self, key: str) -> None:
        self.key = key

    def __repr__(self) -> str:
        return f'{self.key}'


class InputValuePrinter:

    def __init__(self, key: str) -> None:
        self.key = key

    def __repr__(self) -> str:
        return self.key


def parse_arg(arg):
    print(arg)
    if isinstance(arg, str):
        return arg
    if isinstance(arg, dict):
        key, value = arg.popitem()
        if key == 'if':
            cond = value['cond']['isPresent']
            then = [parse_arg(x) for x in value['then']]
            return IfPresentPlaceholder(cond, then=then)
        if key == 'concat':
            items = [parse_arg(x) for x in value]
            return ConcatPlaceholder(items=items)
        elif key == 'inputValue':
            return InputValuePrinter(value)
        elif key == 'inputUri':
            return ArtifactPrinter(value, 'uri')
        elif key == 'outputUri':
            return ArtifactPrinter(value, 'uri')
        elif key == 'outputPath':
            return OutputPathPrinter(value)
    raise Exception(f'Unknown arg:', key, value)


def parse_args(args):
    return [parse_arg(arg) for arg in args]


def make_containerspec(container):
    image = container['image']
    command = container.get('command', [])
    args = parse_args(container['args'])
    print()
    return f'return dsl.ContainerSpec(image="{image}", command={command}, args={args})'


def make_copyright() -> str:
    return """
# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""


def make_imports() -> str:
    return """
from kfp import dsl
from typing import Dict, Optional
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import _ForecastingMetrics
from google_cloud_pipeline_components.types.artifact_types import _ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import _RegressionMetrics
from kfp.dsl import Output
from kfp.dsl import Input
from kfp.dsl import Dataset
from kfp.dsl import Model
from kfp.dsl import Metrics
from kfp.dsl import Artifact
"""


def make_funcdef(parsed_yaml: dict) -> str:
    inputs = sort_strings(
        [parse_input(inp) for inp in parsed_yaml['inputs']] +
        [parse_output(out) for out in parsed_yaml.get('outputs', [])])
    return f"""
@dsl.container_component
def {'_'.join(parsed_yaml['name'].lower().split())}({', '.join(inputs)}):
"""


import textwrap

in_param_types = {
    'String': 'str',
    'JsonObject': 'dict',
    'JsonArray': 'list',
    'Integer': 'int',
    'google.VertexDataset': 'Input[VertexDataset]',
    'google.VertexModel': 'Input[VertexModel]',
    'Metrics': 'Input[Metrics]',
    'Boolean': 'bool',
    'Float': 'float',
    'Dict': 'dict',
    'JSONLines': 'Input[Artifact]',
    'Dataset': 'Input[Dataset]',
    'Model': 'Input[Model]',
    'system.Model': 'Input[Model]',
    'TransformOutput': 'Input[Artifact]',
    'TabularExampleGenMetadata': 'Input[Artifact]',
    'AutoMLTabularTuningResult': 'Input[Artifact]',
    'AutoMLTabularInstanceBaseline': 'Input[Artifact]',
    'DatasetSchema': 'Input[Artifact]',
    'AutoMLTabularModelArchitecture': 'Input[Artifact]',
    'google.UnmanagedContainerModel': 'Input[UnmanagedContainerModel]',
    'Artifact': 'Input[Artifact]',
    'system.Artifact': 'Input[Artifact]',
    'MaterializedSplit': 'Input[Artifact]',
    'google.BQTable': 'Input[Artifact]',
    'google.ForecastingMetrics': 'Input[_ForecastingMetrics]',
    'TabularFeatureRanking': 'Input[Artifact]',
    'SelectedFeatures': 'Input[Artifact]',
    'AutoMLTabularDatasetStats': 'Input[Artifact]',
    'TrainingSchema': 'Input[Artifact]',
    'google.ClassificationMetrics': 'Input[_ClassificationMetrics]',
    'google.RegressionMetrics': 'Input[_RegressionMetrics]',
}
out_param_types = {
    'Metrics': 'Output[Metrics]',
    'Dataset': 'Output[Dataset]',
    'system.Model': 'Output[Model]',
    'String': 'dsl.OutputPath(str)',
    'JsonObject': 'dsl.OutputPath(dict)',
    'JsonArray': 'dsl.OutputPath(list)',
    'Integer': 'dsl.OutputPath(int)',
    'Boolean': 'dsl.OutputPath(bool)',
    'Float': 'dsl.OutputPath(float)',
    'Dict': 'dsl.OutputPath(dict)',
    'google.VertexDataset': 'Output[VertexDataset]',
    'google.VertexModel': 'Output[VertexModel]',
    'JSONLines': 'Output[Artifact]',
    'TransformOutput': 'Output[Artifact]',
    'TabularExampleGenMetadata': 'Output[Artifact]',
    'AutoMLTabularTuningResult': 'Output[Artifact]',
    'AutoMLTabularInstanceBaseline': 'Output[Artifact]',
    'DatasetSchema': 'Output[Artifact]',
    'AutoMLTabularModelArchitecture': 'Output[Artifact]',
    'google.UnmanagedContainerModel': 'Output[UnmanagedContainerModel]',
    'Artifact': 'Output[Artifact]',
    'system.Artifact': 'Output[Artifact]',
    'MaterializedSplit': 'Output[Artifact]',
    'google.BQTable': 'Output[Artifact]',
    'google.ForecastingMetrics': 'Output[_ForecastingMetrics]',
    'TabularFeatureRanking': 'Output[Artifact]',
    'SelectedFeatures': 'Output[Artifact]',
    'AutoMLTabularDatasetStats': 'Output[Artifact]',
    'google.ClassificationMetrics': 'Output[_ClassificationMetrics]',
    'google.RegressionMetrics': 'Output[_RegressionMetrics]',
    'TrainingSchema': 'Output[Artifact]',
}


def sort_strings(lst):
    equal_strings = [s for s in lst if '=' in s]
    non_equal_strings = [s for s in lst if '=' not in s]
    sorted_strings = non_equal_strings + equal_strings
    return sorted_strings


def parse_input(inp):
    if inp['type'] not in in_param_types:
        raise ValueError('Got unknown type', inp['type'])
    optional = inp.get('optional', 'default' in inp)
    param = f"{inp['name']}: "
    if optional:
        param += 'Optional['
    param += in_param_types[inp['type']]
    if optional:
        param += ']'
        if 'default' in inp:
            default = inp['default']
            if default == 'true':
                default = 'True'
            if default == 'false':
                default = 'False'
            if in_param_types[inp['type']] == 'str':
                param += f' = "{default}"'
            else:
                param += f' = {default}'
        else:
            param += ' = None'
    return param


def parse_output(out):
    if out['type'] not in in_param_types:
        raise ValueError('Got unknown type', out['type'])
    optional = out.get('optional', False)
    param = f"{out['name']}: "
    param += out_param_types[out['type']]
    return param


def make_docstring(parsed_yaml: dict):
    return textwrap.indent(
        f'''
"""
{parsed_yaml['description']}
"""
# fmt: on
''', '  ')


def make_file_contents(parsed_yaml):
    final = []
    implementation = parsed_yaml['implementation']
    container = implementation['container']
    final.append(make_copyright())
    final.append(make_imports())
    final.append(make_funcdef(parsed_yaml))
    final.append(make_docstring(parsed_yaml))
    final.append(textwrap.indent(make_containerspec(container), '  '))
    file = '\n'.join(final)
    return file


with open('component.yaml', 'r') as f:
    parsed_yaml = yaml.safe_load(f)
file = make_file_contents(parsed_yaml)
import pyperclip

modified_file = file.replace(
    'google_cloud_pipeline_components.google_cloud_pipeline_components',
    'google_cloud_pipeline_components')
with open('test_component.py', 'w') as f:
    f.write(modified_file)
import subprocess

from test_component import *

subprocess.check_output([
    'autoflake',
    '--in-place',
    '--remove-all-unused-imports',
    'test_component.py',
])
subprocess.check_output([
    'isort',
    '--profile',
    'google',
    'test_component.py',
])
with open('test_component.py', 'r') as f:
    modified_file = f.read()
result = modified_file.replace(
    'google_cloud_pipeline_components',
    'google_cloud_pipeline_components.google_cloud_pipeline_components',
).strip()
pyperclip.copy(result)
print(result)
