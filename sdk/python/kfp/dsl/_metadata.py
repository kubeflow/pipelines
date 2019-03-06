# Copyright 2018 Google LLC
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

from typing import Dict, List
from abc import ABCMeta, abstractmethod
from ._types import _check_valid_type_dict

class BaseMeta(object):
  __metaclass__ = ABCMeta
  def __init__(self):
    pass

  @abstractmethod
  def to_dict(self):
    pass

  def serialize(self):
    import yaml
    return yaml.dump(self.to_dict())

  def __eq__(self, other):
    return self.__dict__ == other.__dict__

class TypeMeta(BaseMeta):
  def __init__(self,
      name: str = '',
      properties: Dict = None):
    self.name = name
    self.properties = {} if properties is None else properties

  def to_dict(self):
    return {self.name: self.properties}

  @staticmethod
  def from_dict(json_dict):
    if not _check_valid_type_dict(json_dict):
      raise ValueError(json_dict + ' is not a valid type string')
    type_meta = TypeMeta()
    type_meta.name, type_meta.properties = list(json_dict.items())[0]
    return type_meta

class ParameterMeta(BaseMeta):
  def __init__(self,
      name: str = '',
      description: str = '',
      param_type: TypeMeta = None,
      default = ''):
    self.name = name
    self.description = description
    self.param_type = TypeMeta() if param_type is None else param_type
    self.default = default

  def to_dict(self):
    return {'name': self.name,
            'description': self.description,
            'type': self.param_type.to_dict(),
            'default': self.default}

class ComponentMeta(BaseMeta):
  def __init__(
      self,
      name: str = '',
      description: str = '',
      inputs: List[ParameterMeta] = None,
      outputs: List[ParameterMeta] = None
  ):
    self.name = name
    self.description = description
    self.inputs = [] if inputs is None else inputs
    self.outputs = [] if outputs is None else outputs

  def to_dict(self):
    return {'name': self.name,
            'description': self.description,
            'inputs': [ input.to_dict() for input in self.inputs ],
            'outputs': [ output.to_dict() for output in self.outputs ]
            }

# Add a pipeline level metadata calss here.
# If one day we combine the component and pipeline yaml, ComponentMeta and PipelineMeta will become one, too.
class PipelineMeta(BaseMeta):
  def __init__(
      self,
      name: str = '',
      description: str = '',
      inputs: List[ParameterMeta] = None
  ):
    self.name = name
    self.description = description
    self.inputs = [] if inputs is None else inputs

  def to_dict(self):
    return {'name': self.name,
            'description': self.description,
            'inputs': [ input.to_dict() for input in self.inputs ]
            }