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
from typing import List, Dict, Union


# TODO: Move this to a separate class
# For now, this identifies a condition with only "==" operator supported.
ConditionOperator = namedtuple('ConditionOperator', 'operator operand1 operand2')
PipelineParamTuple = namedtuple('PipelineParamTuple', 'name op pattern')


def sanitize_k8s_name(name):
    """From _make_kubernetes_name
      sanitize_k8s_name cleans and converts the names in the workflow.
    """
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).lstrip('-').rstrip('-')


def match_serialized_pipelineparam(payload: str):
  """match_serialized_pipelineparam matches the serialized pipelineparam.
  Args:
    payloads (str): a string that contains the serialized pipelineparam.

  Returns:
    PipelineParamTuple
  """
  matches = re.findall(r'{{pipelineparam:op=([\w\s_-]*);name=([\w\s_-]+)}}', payload)
  param_tuples = []
  for match in matches:
      pattern = '{{pipelineparam:op=%s;name=%s}}' % (match[0], match[1])
      param_tuples.append(PipelineParamTuple(
                          name=sanitize_k8s_name(match[1]), 
                          op=sanitize_k8s_name(match[0]), 
                          pattern=pattern))
  return param_tuples


def _extract_pipelineparams(payloads: str or List[str]):
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
    param_tuples += match_serialized_pipelineparam(payload)
  pipeline_params = []
  for param_tuple in list(set(param_tuples)):
    pipeline_params.append(PipelineParam(param_tuple.name,
                                         param_tuple.op, 
                                         pattern=param_tuple.pattern))
  return pipeline_params


def extract_pipelineparams_from_any(payload) -> List['PipelineParam']:
  """Recursively extract PipelineParam instances or serialized string from any object or list of objects.

  Args:
    payload (str or k8_obj or list[str or k8_obj]): a string/a list 
        of strings that contains serialized pipelineparams or a k8 definition 
        object.
  Return:
    List[PipelineParam]
  """
  if not payload:
    return []

  # PipelineParam
  if isinstance(payload, PipelineParam):
    return [payload]
 
  # str
  if isinstance(payload, str):
    return list(set(_extract_pipelineparams(payload)))
  
  # list or tuple
  if isinstance(payload, list) or isinstance(payload, tuple):
    pipeline_params = []
    for item in payload:
      pipeline_params += extract_pipelineparams_from_any(item)
    return list(set(pipeline_params))

  # dict
  if isinstance(payload, dict):
    pipeline_params = []
    for item in payload.values():
      pipeline_params += extract_pipelineparams_from_any(item)
    return list(set(pipeline_params))

  # k8s swagger object
  if hasattr(payload, 'swagger_types') and isinstance(payload.swagger_types, dict):
    pipeline_params = []
    for key in payload.swagger_types.keys():
      pipeline_params += extract_pipelineparams_from_any(getattr(payload, key))

    return list(set(pipeline_params))

  # k8s openapi object
  if hasattr(payload, 'openapi_types') and isinstance(payload.openapi_types, dict):
    pipeline_params = []
    for key in payload.openapi_types.keys():
      pipeline_params += extract_pipelineparams_from_any(getattr(payload, key))

    return list(set(pipeline_params))

  # return empty list  
  return []


class PipelineParam(object):
  """Representing a future value that is passed between pipeline components.

  A PipelineParam object can be used as a pipeline function argument so that it will be a
  pipeline parameter that shows up in ML Pipelines system UI. It can also represent an intermediate
  value passed between components.
  """
  
  def __init__(self, name: str, op_name: str=None, value: str=None, param_type : Union[str, Dict] = None, pattern: str=None):
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
      pattern: the serialized string regex pattern this pipeline parameter created from. 
    Raises: ValueError in name or op_name contains invalid characters, or both op_name
            and value are set.
    """

    valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
    if not re.match(valid_name_regex, name):
      raise ValueError('Only letters, numbers, spaces, "_", and "-" are allowed in name. Must begin with a letter.  '
                       'Got name: {}'.format(name))

    if op_name and value:
      raise ValueError('op_name and value cannot be both set.')

    self.name = name
    # ensure value is None even if empty string or empty list
    # so that serialization and unserialization remain consistent
    # (i.e. None => '' => None)
    self.op_name = op_name if op_name else None
    self.value = value if value else None
    self.param_type = param_type
    self.pattern = pattern or str(self)

  @property
  def full_name(self):
    """Unique name in the argo yaml for the PipelineParam"""
    if self.op_name:
        return self.op_name + '-' + self.name
    return self.name

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
    return '{{pipelineparam:op=%s;name=%s}}' % (op_name, self.name)
  
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
    self.param_type = None
    return self

