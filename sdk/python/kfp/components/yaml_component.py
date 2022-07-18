"""Functions for loading components from compiled YAML."""

# Copyright 2021-2022 The Kubeflow Authors
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

from typing import Optional, Tuple

from kfp.components import base_component
from kfp.components import structures
import requests


class YamlComponent(base_component.BaseComponent):
    """A component loaded from YAML."""

    def execute(self, *args, **kwargs):
        pass


def load_component_from_text(text: str) -> YamlComponent:
    """Loads a component from text.

    Args:
        text (str): The component YAML text.

    Returns:
        YamlComponent: In-memory representation of a component loaded from YAML.
    """
    return YamlComponent(
        structures.ComponentSpec.load_from_component_yaml(text))


def load_component_from_file(file_path: str) -> YamlComponent:
    """Loads a component from a file.

    Args:
        file_path (str): The file path to a YAML component.

    Returns:
        YamlComponent: In-memory representation of a component loaded from YAML.
    """
    with open(file_path, 'r') as component_stream:
        return load_component_from_text(component_stream.read())


def load_component_from_url(url: str,
                            auth: Optional[Tuple[str,
                                                 str]] = None) -> YamlComponent:
    """Loads a component from a url.

    Args:
        file_path (str): The url to a YAML component.
        auth (Tuple[str, str], optional): The a ('username', 'password') tuple of authentication credentials necessary for url access. See Requests Authorization for more information: https://requests.readthedocs.io/en/latest/user/authentication/#authentication

    Returns:
        YamlComponent: In-memory representation of a component loaded from YAML.
    """
    if url is None:
        raise ValueError('url must be a string.')

    if url.startswith('gs://'):
        #Replacing the gs:// URI with https:// URI (works for public objects)
        url = 'https://storage.googleapis.com/' + url[len('gs://'):]

    resp = requests.get(url, auth=auth)
    resp.raise_for_status()

    return load_component_from_text(resp.content.decode('utf-8'))
