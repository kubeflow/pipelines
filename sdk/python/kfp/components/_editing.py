# Copyright 2020 The Kubeflow Pipelines authors
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
import warnings
from .structures import ComponentReference
from ._component_store import ComponentStore


def edit_component_yaml(component_ref: ComponentReference):
    '''Generates code to edit the component YAML text.

    This function can work with components loaded from URLs, files or text.
    This function is used indirectly by calling `op.edit()` method.
    If used in a notebook, adds cell with the component editing code.
    Otherwise returns the code as string.

    Example::

        my_op = components.load_component_from_*(...)
        my_op.edit()
    '''
    editing_code = generate_code_to_edit_component_yaml(component_ref=component_ref)
    # If we're in a notebook, add a new cell. Otherwise return the editing code as string
    try:
        _ipython_create_new_cell(editing_code)
    except:
        return editing_code


def generate_code_to_edit_component_yaml(
    component_ref: ComponentReference, component_store: ComponentStore = None,
):
    '''Generates code to edit the component YAML text.

    This function can work with components loaded from URLs, files or text.
    '''
    component_spec = component_ref.spec
    if not component_spec:
        if not component_store:
            component_store = ComponentStore.default_store
        component_ref = component_store._load_component_spec_in_component_ref(component_ref)
        component_spec = component_ref.spec

    component_data = component_spec._data
    local_path = getattr(component_spec, '_local_path', None)
    if not local_path:
        if component_ref.url:
            # generating local path from the URL
            # Removing schema
            local_path = component_ref.url[component_ref.url.index('://') + 3:]
            # Cleaning path parts
            path_parts = [
                part for part in local_path.split('/') if part not in ['.', '..', '']
            ]
            # Joining path parts together
            local_path = os.path.join(
                '.', *[re.sub(r'[^-\w\s.]', '_', part) for part in path_parts]
            )
        else:
            local_path = os.path.join(component_spec._digest, 'component.yaml')
    if not local_path.endswith('.yaml'):
        warnings.warn(
            'The component file does not have the ".yaml" extension: "{}".'.format(local_path)
        )

    yaml = component_data.decode('utf-8')
    quotes = "'''"
    if quotes in yaml:
        quotes = '"""'
    if quotes in yaml:
        raise NotImplementedError('Editing components that use both triple single quotes and triple double quotes is not supported.')

    editing_code = r"""# ! This code writes to a local file. Inspect the path carefully before running.
# This code writes the edited component to a file and loads it back.
# When satisfied with the edited component, upload it online to share.
import kfp
from pathlib import Path
component_path = '{component_path}'
Path(component_path).parent.mkdir(parents=True, exist_ok=True)
Path(component_path).write_text(r{quotes}
{yaml}{quotes}.lstrip('\n'))
my_op = kfp.components.load_component_from_file('{component_path}')
""".format(
        yaml=yaml,
        component_path=local_path,
        quotes=quotes,
    )
    return editing_code


def _ipython_create_new_cell(contents):
    from IPython.core.getipython import get_ipython
    shell = get_ipython()

    payload = dict(
        source='set_next_input',
        text=contents,
        replace=False,
    )
    shell.payload_manager.write_payload(payload, single=False)
