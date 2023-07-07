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

from typing import Union


class ContainerComponentArtifactChannel:
    """A class for passing in placeholders into container_component decorated
    function."""

    def __init__(self, io_type: str, var_name: str):
        self._io_type = io_type
        self._var_name = var_name

    def __getattr__(self, _name: str) -> Union['placeholders.Placeholder']:
        # aviod circular imports
        from kfp.dsl import placeholders

        attr_to_placeholder_dict = {
            'uri': {
                'input': placeholders.InputUriPlaceholder,
                'output': placeholders.OutputUriPlaceholder,
            },
            'path': {
                'input': placeholders.InputPathPlaceholder,
                'output': placeholders.OutputPathPlaceholder,
            },
            'metadata': {
                'input': placeholders.InputMetadataPlaceholder,
                'output': placeholders.OutputMetadataPlaceholder
            },
        }
        if _name not in ['uri', 'path', 'metadata']:
            raise AttributeError(f'Cannot access artifact attribute "{_name}".')
        return attr_to_placeholder_dict[_name][self._io_type](self._var_name)
