# Copyright 2018-2022 The Kubeflow Authors
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

import hashlib
import warnings

from kfp.components import v1_structures
import yaml


def _load_component_spec_from_component_text(
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
