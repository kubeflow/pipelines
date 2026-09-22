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
"""Functions for loading IR and legacy container component YAML."""

from typing import Optional, Tuple

from kfp.dsl import structures
from kfp.dsl import yaml_component
import requests


def load_component_from_text(text: str) -> yaml_component.YamlComponent:
    """Loads a component from IR or legacy container component YAML.

    IR YAML may include a second document containing a PlatformSpec.
    Legacy ``implementation: container:`` YAML is converted to a native v2
    component at load time; it does not require v1 backend support.

    Args:
        text (str): Component YAML text.

    Returns:
        Component loaded from YAML.
    """
    return yaml_component.YamlComponent(
        component_spec=structures.ComponentSpec.from_yaml_documents(text),
        component_yaml=text)


def load_component_from_file(file_path: str) -> yaml_component.YamlComponent:
    """Loads a component from an IR or legacy container component YAML file.

    Args:
        file_path (str): Filepath to component YAML.

    Returns:
        Component loaded from YAML.

    Example:
      ::

        from kfp import components

        components.load_component_from_file('~/path/to/pipeline.yaml')
    """
    with open(file_path, 'r') as component_stream:
        return load_component_from_text(component_stream.read())


def load_component_from_url(
        url: str,
        auth: Optional[Tuple[str, str]] = None) -> yaml_component.YamlComponent:
    """Loads a component from a URL containing IR or legacy container YAML.

    Args:
        url (str): URL to component YAML.
        auth (Tuple[str, str], optional): A ``('<username>', '<password>')`` tuple of authentication credentials necessary for URL access. See `Requests Authorization <https://requests.readthedocs.io/en/latest/user/authentication/#authentication>`_ for more information.

    Returns:
        Component loaded from YAML.

    Example:
      ::

        from kfp import components

        components.load_component_from_url('https://example.com/compiled-component.yaml')

        components.load_component_from_url('gs://path/to/pipeline.yaml')
    """
    if url is None:
        raise ValueError('url must be a string.')

    if url.startswith('gs://'):
        #Replacing the gs:// URI with https:// URI (works for public objects)
        url = 'https://storage.googleapis.com/' + url[len('gs://'):]

    resp = requests.get(url, auth=auth)
    resp.raise_for_status()

    return load_component_from_text(resp.content.decode('utf-8'))
