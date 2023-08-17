"""The `kfp.components` module contains functions for loading components from
compiled YAML."""
# Copyright 2021 The Kubeflow Authors
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

__all__ = [
    'load_component_from_file',
    'load_component_from_url',
    'load_component_from_text',
    'PythonComponent',
    'BaseComponent',
    'ContainerComponent',
    'YamlComponent',
]

from kfp.components.load_yaml_utilities import load_component_from_file
from kfp.components.load_yaml_utilities import load_component_from_text
from kfp.components.load_yaml_utilities import load_component_from_url
# keep this for backward compatibility with user code "from kfp.components import placholders" and similar
from kfp.dsl import base_component  # noqa: keep unused import
from kfp.dsl import placeholders  # noqa: keep unused import
from kfp.dsl.base_component import BaseComponent
from kfp.dsl.container_component_class import ContainerComponent
from kfp.dsl.python_component import PythonComponent
from kfp.dsl.yaml_component import YamlComponent
