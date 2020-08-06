"""Base class for all SageMaker components"""
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from abc import ABC, abstractmethod
from typing import Type

from .sagemaker_component_spec import SageMakerComponentSpec

def ComponentMetadata(name: str, description: str, spec: Type[SageMakerComponentSpec]):
    def _component_metadata(cls):
        cls.COMPONENT_NAME = name
        cls.COMPONENT_DESCRIPTION = description
        cls.COMPONENT_SPEC = spec
        return cls
    return _component_metadata

class SageMakerComponent(object):
    """Base class for a KFP SageMaker component.

    An instance of a subclass of this component represents an instantiation of the
    component within a pipeline run.

    Attributes:
        COMPONENT_NAME: The name of the component as displayed to the user.
        COMPONENT_DESCRIPTION: The description of the component as displayed to
            the user.
        COMPONENT_SPEC: The correspending spec associated with the component.
    """

    COMPONENT_NAME = ""
    COMPONENT_DESCRIPTION = ""
    COMPONENT_SPEC = SageMakerComponentSpec

    def __init__(self):
        """Initialize a new component."""
        pass

    @abstractmethod
    def Do(self, spec: COMPONENT_SPEC):
        pass
