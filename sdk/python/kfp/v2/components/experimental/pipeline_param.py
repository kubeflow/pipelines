# Copyright 2021 The Kubeflow Authors
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
"""Definition of PipelineParam."""

import json
import re
from dataclasses import dataclass
from typing import Dict, List, Optional, Union

from kfp.v2.components import utils
from kfp.v2.components.types import type_utils


@dataclass
class ConditionOperator:
    """Represents a condition expression to be used in dsl.Condition().

    Attributes:
      operator: The operator of the condition.
      operand1: The left operand.
      operand2: The right operand.
    """
    operator: str
    operand1: Union['PipelineParam', type_utils.PARAMETER_TYPES]
    operand: Union['PipelineParam', type_utils.PARAMETER_TYPES]


class PipelineParam:
    """Represents a future value that is passed between pipeline components.

    A PipelineParam object can be used as a pipeline function argument so that
    it will be a pipeline parameter that shows up in ML Pipelines system UI. It
    can also represent an intermediate value passed between components.

    Attributes:
      name: The name of the pipeline parameter.
      op_name: The name of the operation that produces the PipelineParam. None
        means it is not produced by any operator, so if None, either user
        constructs it directly (for providing an immediate value), or it is a
        pipeline function argument.
      param_type: The type of the PipelineParam.
      value: The actual value of the PipelineParam. If provided, the
        PipelineParam is "resolved" immediately. For now, we support string
        only.
      pattern: The serialized string regex pattern this pipeline parameter
        created from.
    """

    def __init__(
        self,
        name: str,
        op_name: Optional[str] = None,
        param_type: Optional[Union[str, Dict]] = None,
        value: Optional[type_utils.PARAMETER_TYPES] = None,
    ):
        """Inits a PipelineParam instance.

        Args:
          name: The name of the pipeline parameter.
          op_name: Optional; The name of the operation that produces the
            PipelineParam.
          param_type: Optional; The type of the PipelineParam.
          value: Optional; The actual value of the PipelineParam.

        Raises:
          ValueError in name or op_name contains invalid characters, or both
          op_name and value are set.
        """
        valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
        if not re.match(valid_name_regex, name):
            raise ValueError(
                'Only letters, numbers, spaces, "_", and "-" are allowed in the '
                'name. Must begin with a letter. Got name: {}'.format(name))

        if op_name and value:
            raise ValueError('op_name and value cannot be both set.')

        self.name = name
        self.value = value
        # ensure value is None even if empty string or empty list/dict
        # so that serialization and unserialization remain consistent
        # (i.e. None => '' => None)
        self.op_name = op_name or None
        self.param_type = param_type or None

    @property
    def full_name(self) -> str:
        """Unique name for the PipelineParam."""
        return f'{self.op_name}-{self.name}' if self.op_name else self.name

    @property
    def pattern(self) -> str:
        """Unique pattern for the PipelineParam."""
        return str(self)

    def __str__(self) -> str:
        """String representation of the PipelineParam.

        The string representation is a string identifier so we can mix the
        PipelineParam inline with other strings such as arguments. For example,
        we can support: ['echo %s' % param] as the container command and later a
        compiler can replace the placeholder
        '{{pipelineparam:op=%s;name=%s;type=%s}}' with its own parameter
        identifier.
        """
        op_name = self.op_name or ''
        name = self.name
        param_type = self.param_type or ''
        if isinstance(param_type, dict):
            param_type = json.dumps(param_type)
        return f'{{{{pipelineparam:op={op_name};name={name};type={param_type};}}}}'

    def __repr__(self) -> str:
        """Representation of the PipelineParam.

        We make repr return the placeholder string so that if someone uses
        str()-based serialization of complex objects containing `PipelineParam`,
        it works properly.
        (e.g. str([1, 2, 3, kfp.dsl.PipelineParam("aaa"), 4, 5, 6,]))
        """
        return str(self)

    def __hash__(self) -> int:
        """Returns the hash of a PipelineParam."""
        return hash(self.pattern)

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


def extract_pipelineparams(payload: str) -> List['PipelineParam']:
    """Extracts a list of PipelineParam instances from the payload string.

    Note: this function removes all duplicate matches.

    Args:
      payload: A string that may contain serialized PipelineParams.

    Returns:
      A list of PipelineParams found from the payload.
    """
    matches = re.findall(
        r'{{pipelineparam:op=([\w\s_-]*);name=([\w\s_-]+);type=([\w\s{}":_-]*);}}',
        payload)
    deduped_params = set()
    for match in matches:
        op_name, name, param_type = match
        try:
            param_type = json.loads(param_type)
        except json.JSONDecodeError:
            pass

        deduped_params.add(
            PipelineParam(
                name=utils.sanitize_k8s_name(name),
                op_name=utils.sanitize_k8s_name(op_name),
                param_type=param_type,
            ))
    return list(deduped_params)


def extract_pipelineparams_from_any(
    payload: Union['PipelineParam', str, list, tuple, dict]
) -> List['PipelineParam']:
    """Recursively extract PipelineParam instances any object or list of objects.

    Args:
      payload: An object that contains serialized PipelineParams or k8
        definition objects.

    Returns:
      A list of PipelineParams found from the payload.
    """
    if not payload:
        return []

    # PipelineParam
    if isinstance(payload, PipelineParam):
        return [payload]

    # str
    if isinstance(payload, str):
        return list(set(extract_pipelineparams(payload)))

    # list or tuple
    if isinstance(payload, list) or isinstance(payload, tuple):
        pipeline_params = []
        for item in payload:
            pipeline_params += extract_pipelineparams_from_any(item)
        return list(set(pipeline_params))

    # dict
    if isinstance(payload, dict):
        pipeline_params = []
        for key, value in payload.items():
            pipeline_params += extract_pipelineparams_from_any(key)
            pipeline_params += extract_pipelineparams_from_any(value)
        return list(set(pipeline_params))

    # TODO(chensun): extract PipelineParam from v2 container spec?

    return []
