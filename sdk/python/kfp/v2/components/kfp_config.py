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
from typing import Dict, Optional, OrderedDict

import configparser
import dataclasses
import pathlib
import warnings

_KFP_CONFIG_FILE = 'kfp_config.ini'

_COMPONENTS_SECTION = 'Components'


@dataclasses.dataclass
class KFPConfigComponent():
    function_name: str
    module_path: pathlib.Path


class KFPConfig():

    def __init__(self, config_directory: Optional[pathlib.Path] = None):
        self._config_parser = configparser.ConfigParser()
        # Preserve case for keys.
        self._config_parser.optionxform = lambda x: x

        if config_directory is None:
            self._config_filepath = pathlib.Path(_KFP_CONFIG_FILE)
        else:
            self._config_filepath = config_directory / _KFP_CONFIG_FILE

        try:
            with open(str(self._config_filepath), 'r') as f:
                self._config_parser.read_file(f)
        except IOError:
            warnings.warn('No existing KFP Config file found')

        if not self._config_parser.has_section(_COMPONENTS_SECTION):
            self._config_parser.add_section(_COMPONENTS_SECTION)

        self._components = {}

    def add_component(self, function_name: str, path: pathlib.Path):
        self._components[function_name] = str(path)

    def save(self):
        # Always write out components in alphabetical order for determinism,
        # especially in tests.
        for function_name in sorted(self._components.keys()):
            self._config_parser[_COMPONENTS_SECTION][
                function_name] = self._components[function_name]

        with open(str(self._config_filepath), 'w') as f:
            self._config_parser.write(f)

    def get_components(self) -> Dict[str, pathlib.Path]:
        result = {
            function_name: pathlib.Path(module_path) for function_name,
            module_path in self._config_parser[_COMPONENTS_SECTION].items()
        }
        return result
