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
from .types import BaseType, _check_valid_type_dict, _instance_to_dict

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

  def to_dict_or_str(self):
    if self.properties is None or len(self.properties) == 0:
      return self.name
    else:
      return {self.name: self.properties}

  @staticmethod
  def from_dict_or_str(payload):
    '''from_dict_or_str accepts a payload object and returns a TypeMeta instance
     Args:
       payload (str/dict): the payload could be a str or a dict
    '''

    type_meta = TypeMeta()
    if isinstance(payload, dict):
      if not _check_valid_type_dict(payload):
        raise ValueError(payload + ' is not a valid type string')
      type_meta.name, type_meta.properties = list(payload.items())[0]
      # Convert possible OrderedDict to dict
      type_meta.properties = dict(type_meta.properties)
    elif isinstance(payload, str):
      type_meta.name = payload
    else:
      raise ValueError('from_dict_or_str is expecting either dict or str.')
    return type_meta

  def serialize(self):
    return str(self.to_dict_or_str())

  @staticmethod
  def deserialize(payload):
    # If the payload is a string of a dict serialization, convert it back to a dict
    try:
      import ast
      payload = ast.literal_eval(payload)
    except:
      pass
    return TypeMeta.from_dict_or_str(payload)

class ParameterMeta(BaseMeta):
  def __init__(self,
      name: str,
      description: str = '',
      param_type: TypeMeta = None,
      default = None):
    self.name = name
    self.description = description
    self.param_type = TypeMeta() if param_type is None else param_type
    self.default = default

  def to_dict(self):
    return {'name': self.name,
            'description': self.description,
            'type': self.param_type.to_dict_or_str(),
            'default': self.default}

class ComponentMeta(BaseMeta):
  def __init__(
      self,
      name: str,
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
      name: str,
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

def _annotation_to_typemeta(annotation):
  '''_annotation_to_type_meta converts an annotation to an instance of TypeMeta
  Args:
    annotation(BaseType/str/dict): input/output annotations
  Returns:
    TypeMeta
    '''
  if isinstance(annotation, BaseType):
    arg_type = TypeMeta.deserialize(_instance_to_dict(annotation))
  elif isinstance(annotation, str):
    arg_type = TypeMeta.deserialize(annotation)
  elif isinstance(annotation, dict):
    if not _check_valid_type_dict(annotation):
      raise ValueError('Annotation ' + str(annotation) + ' is not a valid type dictionary.')
    arg_type = TypeMeta.deserialize(annotation)
  else:
    return TypeMeta()
  return arg_type


def _extract_component_metadata(func):
  '''Creates component metadata structure instance based on the function signature.'''

  import inspect
  fullargspec = inspect.getfullargspec(func)
  annotations = fullargspec.annotations

  # defaults
  arg_defaults = {}
  if fullargspec.defaults:
    for arg, default in zip(reversed(fullargspec.args), reversed(fullargspec.defaults)):
      arg_defaults[arg] = default

  # Inputs
  inputs = []
  for arg in fullargspec.args:
    arg_type = TypeMeta()
    arg_default = arg_defaults[arg] if arg in arg_defaults else None
    if arg in annotations:
      arg_type = _annotation_to_typemeta(annotations[arg])
    inputs.append(ParameterMeta(name=arg, description='', param_type=arg_type, default=arg_default))
  # Outputs
  outputs = []
  if 'return' in annotations:
    for output in annotations['return']:
      arg_type = _annotation_to_typemeta(annotations['return'][output])
      outputs.append(ParameterMeta(name=output, description='', param_type=arg_type))

  #TODO: add descriptions to the metadata
  #docstring parser:
  #  https://github.com/rr-/docstring_parser
  #  https://github.com/terrencepreilly/darglint/blob/master/darglint/parse.py

  # Construct the ComponentMeta
  return ComponentMeta(
    name=func.__name__,
    description='',
    inputs=inputs,
    outputs=outputs,
  )


def _extract_pipeline_metadata(func):
  '''Creates pipeline metadata structure instance based on the function signature.'''

  import inspect
  fullargspec = inspect.getfullargspec(func)
  args = fullargspec.args
  annotations = fullargspec.annotations

  # defaults
  arg_defaults = {}
  if fullargspec.defaults:
    for arg, default in zip(reversed(fullargspec.args), reversed(fullargspec.defaults)):
      arg_defaults[arg] = default

  # Construct the PipelineMeta
  pipeline_meta = PipelineMeta(
    name=getattr(func, '_pipeline_name', func.__name__),
    description=getattr(func, '_pipeline_description', func.__doc__)
  )
  # Inputs
  for arg in args:
    arg_type = TypeMeta()
    arg_default = arg_defaults[arg] if arg in arg_defaults else None
    if arg in annotations:
      arg_type = _annotation_to_typemeta(annotations[arg])
    pipeline_meta.inputs.append(ParameterMeta(name=arg, description='', param_type=arg_type, default=arg_default))

  #TODO: add descriptions to the metadata
  #docstring parser:
  #  https://github.com/rr-/docstring_parser
  #  https://github.com/terrencepreilly/darglint/blob/master/darglint/parse.py
  return pipeline_meta
