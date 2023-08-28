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
"""Component loaded from YAML."""

from google.protobuf import json_format
from kfp.dsl import base_component
from kfp.dsl import structures
from kfp.pipeline_spec import pipeline_spec_pb2


class YamlComponent(base_component.BaseComponent):
    """A component loaded from a YAML file.

    **Note:** ``YamlComponent`` is not intended to be used to construct components directly. Use ``kfp.components.load_component_from_*()`` instead.

    Attribute:
        component_spec: Component definition.
        component_yaml: The yaml string that this component is loaded from.
    """

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        component_yaml: str,
    ):
        super().__init__(component_spec=component_spec)
        self.component_yaml = component_yaml

    @property
    def pipeline_spec(self) -> pipeline_spec_pb2.PipelineSpec:
        """Returns the pipeline spec of the component."""
        component_dict = structures.load_documents_from_yaml(
            self.component_yaml)[0]
        is_v1 = 'implementation' in set(component_dict.keys())
        if is_v1:
            return self.component_spec.to_pipeline_spec()
        else:
            return json_format.ParseDict(component_dict,
                                         pipeline_spec_pb2.PipelineSpec())

    def execute(self, *args, **kwargs):
        """Not implemented."""
        raise NotImplementedError
