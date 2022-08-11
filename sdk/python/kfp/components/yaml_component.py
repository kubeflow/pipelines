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
"""Functions for loading component from yaml."""

__all__ = [
    'load_component_from_text',
    'load_component_from_url',
    'load_component_from_file',
]

from kfp.components import base_component
from kfp.components import structures


class YamlComponent(base_component.BaseComponent):
    """Component defined YAML component spec."""

    def execute(self, *args, **kwargs):
        pass


def load_component_from_text(text: str) -> base_component.BaseComponent:
    """Loads component from text."""
    return YamlComponent(
        structures.ComponentSpec.load_from_component_yaml(text))


def load_component_from_file(file_path: str) -> base_component.BaseComponent:
    """Loads component from file.

    Args:
        file_path: A string containing path to the YAML file.
    """
    with open(file_path, 'rb') as component_stream:
        return load_component_from_text(component_stream.read())


def load_component_from_url(url: str,
                            auth=None) -> base_component.BaseComponent:
    """Loads component from url.

    Args:
        url: A string containing path to the url containing YAML file.
        auth: The authentication credentials necessary for url access.
    """

    if url is None:
        raise TypeError

    if url.startswith('gs://'):
        #Replacing the gs:// URI with https:// URI (works for public objects)
        url = 'https://storage.googleapis.com/' + url[len('gs://'):]

    import requests
    resp = requests.get(url, auth=auth)
    resp.raise_for_status()

    return load_component_from_text(resp.content)
