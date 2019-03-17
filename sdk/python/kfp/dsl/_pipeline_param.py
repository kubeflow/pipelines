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


import re
from collections import namedtuple
from ._metadata import TypeMeta


# TODO: Move this to a separate class
# For now, this identifies a condition with only "==" operator supported.
ConditionOperator = namedtuple('ConditionOperator', 'operator operand1 operand2')
PipelineParamTuple = namedtuple('PipelineParamTuple', 'name op value type')

def _match_serialized_pipelineparam(payload: str):
  """_match_serialized_pipelineparam matches the serialized pipelineparam.
  Args:
    payloads (str): a string that contains the serialized pipelineparam.

  Returns:
    PipelineParamTuple
  """
  matches = re.findall(r'{{pipelineparam:op=([\w\s_-]*);name=([\w\s_-]+);value=(.*?);type=(.*?);}}', payload)
  if len(matches) == 0:
    matches = re.findall(r'{{pipelineparam:op=([\w\s_-]*);name=([\w\s_-]+);value=(.*?)}}', payload)
  param_tuples = []
  for match in matches:
    if len(match) == 3:
      param_tuples.append(PipelineParamTuple(name=match[1], op=match[0], value=match[2], type=''))
    elif len(match) == 4:
      param_tuples.append(PipelineParamTuple(name=match[1], op=match[0], value=match[2], type=match[3]))
  return param_tuples

def _extract_pipelineparams(payloads: str or list[str]):
  """_extract_pipelineparam extract a list of PipelineParam instances from the payload string.
  Note: this function removes all duplicate matches.

  Args:
    payload (str or list[str]): a string/a list of strings that contains serialized pipelineparams
  Return:
    List[PipelineParam]
  """
  if isinstance(payloads, str):
    payloads = [payloads]
  param_tuples = []
  for payload in payloads:
    param_tuples += _match_serialized_pipelineparam(payload)
  pipeline_params = []
  for param_tuple in list(set(param_tuples)):
    pipeline_params.append(PipelineParam(param_tuple.name, param_tuple.op, param_tuple.value, TypeMeta.deserialize(param_tuple.type)))
  return pipeline_params

class PipelineParam(object):
  """Representing a future value that is passed between pipeline components.

  A PipelineParam object can be used as a pipeline function argument so that it will be a
  pipeline parameter that shows up in ML Pipelines system UI. It can also represent an intermediate
  value passed between components.
  """
  
  def __init__(self, name: str, op_name: str=None, value: str=None, param_type: TypeMeta=TypeMeta()):
    """Create a new instance of PipelineParam.
    Args:
      name: name of the pipeline parameter.
      op_name: the name of the operation that produces the PipelineParam. None means
               it is not produced by any operator, so if None, either user constructs it
               directly (for providing an immediate value), or it is a pipeline function
               argument.
      value: The actual value of the PipelineParam. If provided, the PipelineParam is
             "resolved" immediately. For now, we support string only.
      param_type: the type of the PipelineParam.
    Raises: ValueError in name or op_name contains invalid characters, or both op_name
            and value are set.
    """

    valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
    if not re.match(valid_name_regex, name):
      raise ValueError('Only letters, numbers, spaces, "_", and "-" are allowed in name. Must begin with letter: %s' % (name))

    if op_name and value:
      raise ValueError('op_name and value cannot be both set.')

    self.op_name = op_name
    self.name = name
    self.value = value
    self.param_type = param_type

  def __str__(self):
    """String representation.

    The string representation is a string identifier so we can mix the PipelineParam inline
    with other strings such as arguments. For example, we can support:
    ['echo %s' % param] as the container command and later a compiler can replace
    the placeholder "{{pipelineparam:op=%s;name=%s}}" with its own parameter identifier.
    """

    #This is deleted because if users specify default values to PipelineParam,
    # The compiler can not detect it as the value is not NULL.
    #if self.value:
    #  return str(self.value)

    op_name = self.op_name if self.op_name else ''
    value = self.value if self.value else ''
    if self.param_type is None:
      return '{{pipelineparam:op=%s;name=%s;value=%s}}' % (op_name, self.name, value)
    else:
      return '{{pipelineparam:op=%s;name=%s;value=%s;type=%s;}}' % (op_name, self.name, value, self.param_type.serialize())
  
  def __repr__(self):
      return str({self.__class__.__name__: self.__dict__})

  def __eq__(self, other):
    return ConditionOperator('==', self, other)

  def __ne__(self, other):
    return ConditionOperator('!=', self, other)

  def __lt__(self, other):
    return ConditionOperator('<', self, other)

  def __le__(self, other):
    return ConditionOperator('<=', self, other)

  def __gt__(self, other):
    return ConditionOperator('>', self, other)

  def __ge__(self, other):
    return ConditionOperator('>=', self, other)

  def __hash__(self):
    return hash((self.op_name, self.name))

  def ignore_type(self):
    """ignore_type ignores the type information such that type checking would also pass"""
    self.param_type = TypeMeta()
    return self

