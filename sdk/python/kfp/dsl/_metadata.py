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
import warnings
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


class ParameterMeta(BaseMeta):
  def __init__(self,
      name: str,
      description: str = '',
      param_type = None,
      default = None):
    self.name = name
    self.description = description
    self.param_type = param_type
    self.default = default

  def to_dict(self):
    result = {}
    if self.name:
      result['name'] = self.name
    if self.description:
      result['description'] = self.description
    if self.param_type:
      result['type'] = self.param_type
    if self.default:
      result['default'] = self.default
    return result


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
    result = {}
    if self.name:
      result['name'] = self.name
    if self.description:
      result['description'] = self.description
    if self.inputs:
      result['inputs'] = [input.to_dict() for input in self.inputs]
    if self.outputs:
      result['outputs'] = [output.to_dict() for output in self.outputs]
    return result


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
    result = {}
    if self.name:
      result['name'] = self.name
    if self.description:
      result['description'] = self.description
    if self.inputs:
      result['inputs'] = [input.to_dict() for input in self.inputs]
    return result

def _annotation_to_typemeta(annotation):
  '''_annotation_to_type_meta converts an annotation to a type structure
  Args:
    annotation(BaseType/str/dict): input/output annotations
      BaseType: registered in kfp.dsl.types
      str: either a string of a dict serialization or a string of the type name
      dict: type name and properties. note that the properties values can be dict.
  Returns:
    dict or string representing the type
    '''
  if isinstance(annotation, BaseType):
    arg_type = _instance_to_dict(annotation)
  elif isinstance(annotation, str):
    arg_type = annotation
  elif isinstance(annotation, dict):
    if not _check_valid_type_dict(annotation):
      raise ValueError('Annotation ' + str(annotation) + ' is not a valid type dictionary.')
    arg_type = annotation
  else:
    return None
  return arg_type


def _extract_component_metadata(func):
  '''Creates component metadata structure instance based on the function signature.'''

  # Importing here to prevent circular import failures
  #TODO: Change _pipeline_param to stop importing _metadata
  from ._pipeline_param import PipelineParam

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
    arg_type = None
    arg_default = arg_defaults[arg] if arg in arg_defaults else None
    if isinstance(arg_default, PipelineParam):
      warnings.warn('Explicit creation of `kfp.dsl.PipelineParam`s by the users is deprecated. The users should define the parameter type and default values using standard pythonic constructs: def my_func(a: int = 1, b: str = "default"):')
      arg_default = arg_default.value
    if arg in annotations:
      arg_type = _annotation_to_typemeta(annotations[arg])
    inputs.append(ParameterMeta(name=arg, param_type=arg_type, default=arg_default))
  # Outputs
  outputs = []
  if 'return' in annotations:
    for output in annotations['return']:
      arg_type = _annotation_to_typemeta(annotations['return'][output])
      outputs.append(ParameterMeta(name=output, param_type=arg_type))

  #TODO: add descriptions to the metadata
  #docstring parser:
  #  https://github.com/rr-/docstring_parser
  #  https://github.com/terrencepreilly/darglint/blob/master/darglint/parse.py

  # Construct the ComponentMeta
  return ComponentMeta(
    name=func.__name__,
    inputs=inputs if inputs else None,
    outputs=outputs if outputs else None,
  )


def _extract_pipeline_metadata(func):
  '''Creates pipeline metadata structure instance based on the function signature.'''

  # Importing here to prevent circular import failures
  #TODO: Change _pipeline_param to stop importing _metadata
  from ._pipeline_param import PipelineParam

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
    arg_type = None
    arg_default = arg_defaults[arg] if arg in arg_defaults else None
    if isinstance(arg_default, PipelineParam):
      warnings.warn('Explicit creation of `kfp.dsl.PipelineParam`s by the users is deprecated. The users should define the parameter type and default values using standard pythonic constructs: def my_func(a: int = 1, b: str = "default"):')
      arg_default = arg_default.value
    if arg in annotations:
      arg_type = _annotation_to_typemeta(annotations[arg])
    arg_type_properties = list(arg_type.values())[0] if isinstance(arg_type, dict) else {}
    if 'openapi_schema_validator' in arg_type_properties and arg_default is not None:
      from jsonschema import validate
      import json
      schema_object = arg_type_properties['openapi_schema_validator']
      if isinstance(schema_object, str):
        # In case the property value for the schema validator is a string instead of a dict.
        schema_object = json.loads(schema_object)
      validate(instance=arg_default, schema=schema_object)
    pipeline_meta.inputs.append(ParameterMeta(name=arg, param_type=arg_type, default=arg_default))

  #TODO: add descriptions to the metadata
  #docstring parser:
  #  https://github.com/rr-/docstring_parser
  #  https://github.com/terrencepreilly/darglint/blob/master/darglint/parse.py
  return pipeline_meta
