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
"""Functions for loading components from compiled YAML."""

import hashlib
from typing import Optional, Tuple, Union
import warnings

from kfp.dsl import structures
from kfp.dsl import v1_structures
from kfp.dsl import yaml_component
import requests
import yaml


def load_component_from_text(text: str) -> yaml_component.YamlComponent:
    """Loads a component from text.

    Args:
        text (str): Component YAML text.

    Returns:
        Component loaded from YAML.
    """
    return yaml_component.YamlComponent(
        component_spec=_load_component_spec_from_yaml_documents(text),
        component_yaml=text)


def load_component_from_file(file_path: str) -> yaml_component.YamlComponent:
    """Loads a component from a file.

    Args:
        file_path (str): Filepath to a YAML component.

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
    """Loads a component from a URL.

    Args:
        url (str): URL to a YAML component.
        auth (Tuple[str, str], optional): A ``('<username>', '<password>')`` tuple of authentication credentials necessary for URL access. See `Requests Authorization <https://requests.readthedocs.io/en/latest/user/authentication/#authentication>`_ for more information.

    Returns:
        Component loaded from YAML.

    Example:
      ::

        from kfp import components

        components.load_component_from_url('https://raw.githubusercontent.com/kubeflow/pipelines/7b49eadf621a9054e1f1315c86f95fb8cf8c17c3/sdk/python/kfp/compiler/test_data/components/identity.yaml')

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


def _load_documents_from_yaml(component_yaml: str) -> Tuple[dict, dict]:
    """Loads up to two YAML documents from a YAML string.

    First document must always be present. If second document is
    present, it is returned as a dict, else an empty dict.
    """
    documents = list(yaml.safe_load_all(component_yaml))
    num_docs = len(documents)
    if num_docs == 1:
        pipeline_spec_dict = documents[0]
        platform_spec_dict = {}
    elif num_docs == 2:
        pipeline_spec_dict = documents[0]
        platform_spec_dict = documents[1]
    else:
        raise ValueError(
            f'Expected one or two YAML documents in the IR YAML file. Got: {num_docs}.'
        )
    return pipeline_spec_dict, platform_spec_dict


def _load_component_spec_from_yaml_documents(
        component_yaml: str) -> structures.ComponentSpec:
    """Loads V1 or V2 component YAML into a ComponentSpec.

    Args:
        component_yaml: PipelineSpec and optionally PlatformSpec YAML documents as a single string.

    Returns:
        ComponentSpec: The ComponentSpec object.
    """

    def extract_description(component_yaml: str) -> Union[str, None]:
        heading = '# Description: '
        multi_line_description_prefix = '#             '
        index_of_heading = 2
        if heading in component_yaml:
            description = component_yaml.splitlines()[index_of_heading]

            # Multi line
            comments = component_yaml.splitlines()
            index = index_of_heading + 1
            while comments[index][:len(multi_line_description_prefix
                                      )] == multi_line_description_prefix:
                description += '\n' + comments[index][
                    len(multi_line_description_prefix) + 1:]
                index += 1

            return description[len(heading):]
        else:
            return None

    pipeline_spec_dict, platform_spec_dict = _load_documents_from_yaml(
        component_yaml)

    is_v1 = 'implementation' in set(pipeline_spec_dict.keys())
    if is_v1:
        v1_component = load_v1_component_spec_from_component_text(
            component_yaml)
        return structures.ComponentSpec.from_v1_component_spec(v1_component)
    else:
        component_spec = structures.ComponentSpec.from_ir_dicts(
            pipeline_spec_dict, platform_spec_dict)
        if not component_spec.description:
            component_spec.description = extract_description(
                component_yaml=component_yaml)
        return component_spec


def load_v1_component_spec_from_component_text(
        text) -> v1_structures.ComponentSpec:
    component_dict = yaml.safe_load(text)
    component_spec = v1_structures.ComponentSpec.from_dict(component_dict)

    if isinstance(component_spec.implementation,
                  v1_structures.ContainerImplementation) and (
                      component_spec.implementation.container.command is None):
        warnings.warn(
            'Container component must specify command to be compatible with KFP '
            'v2 compatible mode and emissary executor, which will be the default'
            ' executor for KFP v2.'
            'https://www.kubeflow.org/docs/components/pipelines/installation/choose-executor/',
            category=FutureWarning,
        )

    # Calculating hash digest for the component
    data = text if isinstance(text, bytes) else text.encode('utf-8')
    data = data.replace(b'\r\n', b'\n')  # Normalizing line endings
    digest = hashlib.sha256(data).hexdigest()
    component_spec._digest = digest

    return component_spec
