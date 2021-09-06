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
"""Definition of PipelineChannel."""

import dataclasses
import json
import re
from typing import Dict, List, Optional, Union

from kfp.v2.components import utils
from kfp.v2.components.types import type_utils


@dataclasses.dataclass
class ConditionOperator:
    """Represents a condition expression to be used in dsl.Condition().

    Attributes:
      operator: The operator of the condition.
      operand1: The left operand.
      operand2: The right operand.
    """
    operator: str
    operand1: Union['PipelineParameterChannel', type_utils.PARAMETER_TYPES]
    operand2: Union['PipelineParameterChannel', type_utils.PARAMETER_TYPES]


class PipelineChannel:
    """Represents a future value that is passed between pipeline components.

    A PipelineChannel object can be used as a pipeline function argument so that
    it will be a pipeline artifact or parameter that shows up in ML Pipelines
    system UI. It can also represent an intermediate value passed between
    components.

    Attributes:
      name: The name of the pipeline channel.
      channel_type: The type of the pipeline channel.
      op_name: The name of the operation that produces the pipeline channel.
        None means it is not produced by any operator, so if None, either user
        constructs it directly (for providing an immediate value), or it is a
        pipeline function argument.
      pattern: The serialized string regex pattern this pipeline channel created
        from.
    """

    def __init__(
        self,
        name: str,
        channel_type: Union[str, Dict],
        op_name: Optional[str] = None,
    ):
        """Inits a PipelineChannel instance.

        Args:
          name: The name of the pipeline channel.
          channel_type: The type of the pipeline channel.
          op_name: Optional; The name of the operation that produces the
            pipeline channel.

        Raises:
          ValueError: If name or op_name contains invalid characters.
          ValueError: If both op_name and value are set.
        """
        valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
        if not re.match(valid_name_regex, name):
            raise ValueError(
                'Only letters, numbers, spaces, "_", and "-" are allowed in the '
                'name. Must begin with a letter. Got name: {}'.format(name)
            )

        self.name = name
        self.channel_type = channel_type
        # ensure value is None even if empty string or empty list/dict
        # so that serialization and unserialization remain consistent
        # (i.e. None => '' => None)
        self.op_name = op_name or None

    @property
    def full_name(self) -> str:
        """Unique name for the PipelineChannel."""
        return f'{self.op_name}-{self.name}' if self.op_name else self.name

    @property
    def pattern(self) -> str:
        """Unique pattern for the PipelineChannel."""
        return str(self)

    def __str__(self) -> str:
        """String representation of the PipelineChannel.

        The string representation is a string identifier so we can mix
        the PipelineChannel inline with other strings such as arguments.
        For example, we can support: ['echo %s' % param] as the
        container command and later a compiler can replace the
        placeholder '{{pipeline_channel:op=%s;name=%s;type=%s}}' with
        its own parameter identifier.
        """
        op_name = self.op_name or ''
        name = self.name
        channel_type = self.channel_type or ''
        if isinstance(channel_type, dict):
            channel_type = json.dumps(channel_type)
        return f'{{{{channel:op={op_name};name={name};type={channel_type};}}}}'

    def __repr__(self) -> str:
        """Representation of the PipelineChannel.

        We make repr return the placeholder string so that if someone
        uses str()-based serialization of complex objects containing
        `PipelineChannel`, it works properly. (e.g. str([1, 2, 3,
        kfp.dsl.PipelineChannel("aaa"), 4, 5, 6,]))
        """
        return str(self)

    def __hash__(self) -> int:
        """Returns the hash of a PipelineChannel."""
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


class PipelineParameterChannel(PipelineChannel):
    """Represents a pipeline parameter channel.

    Attributes:
      name: The name of the pipeline channel.
      channel_type: The type of the pipeline channel.
      op_name: The name of the operation that produces the pipeline channel.
        None means it is not produced by any operator, so if None, either user
        constructs it directly (for providing an immediate value), or it is a
        pipeline function argument.
      pattern: The serialized string regex pattern this pipeline channel created
        from.
      value: The actual value of the pipeline channel. If provided, the
        pipeline channel is "resolved" immediately.
    """

    def __init__(
        self,
        name: str,
        channel_type: Union[str, Dict],
        op_name: Optional[str] = None,
        value: Optional[type_utils.PARAMETER_TYPES] = None,
    ):
        """Inits a PipelineArtifactChannel instance.

        Args:
          name: The name of the pipeline channel.
          channel_type: The type of the pipeline channel.
          op_name: Optional; The name of the operation that produces the
            pipeline channel.
          value: Optional; The actual value of the pipeline channel.

        Raises:
          ValueError: If name or op_name contains invalid characters.
          ValueError: If both op_name and value are set.
          TypeError: If the channel type is not a parameter type.
        """
        if op_name and value:
            raise ValueError('op_name and value cannot be both set.')

        if not type_utils.is_parameter_type(channel_type):
            raise TypeError(f'{channel_type} is not a parameter type.')

        self.value = value

        super(PipelineParameterChannel, self).__init__(
            name=name,
            channel_type=channel_type,
            op_name=op_name,
        )


class PipelineArtifactChannel(PipelineChannel):
    """Represents a pipeline artifact channel.

    Attributes:
      name: The name of the pipeline channel.
      channel_type: The type of the pipeline channel.
      op_name: The name of the operation that produces the pipeline channel.
        A pipeline artifact channel is always produced by some op.
      pattern: The serialized string regex pattern this pipeline channel created
        from.
    """

    def __init__(
        self,
        name: str,
        channel_type: Union[str, Dict],
        op_name: str,
    ):
        """Inits a PipelineArtifactChannel instance.

        Args:
          name: The name of the pipeline channel.
          channel_type: The type of the pipeline channel.
          op_name: The name of the operation that produces the pipeline channel.

        Raises:
          ValueError: If name or op_name contains invalid characters.
          TypeError: If the channel type is not an artifact type.
        """
        if type_utils.is_parameter_type(channel_type):
            raise TypeError(f'{channel_type} is not an artifact type.')

        super(PipelineArtifactChannel, self).__init__(
            name=name,
            channel_type=channel_type,
            op_name=op_name,
        )


def extract_pipeline_channels(payload: str) -> List['PipelineChannel']:
    """Extracts a list of PipelineChannel instances from the payload string.

    Note: this function removes all duplicate matches.

    Args:
      payload: A string that may contain serialized PipelineChannels.

    Returns:
      A list of PipelineChannels found from the payload.
    """
    matches = re.findall(
        r'{{channel:op=([\w\s_-]*);name=([\w\s_-]+);type=([\w\s{}":_-]*);}}',
        payload
    )
    deduped_channels = set()
    for match in matches:
        op_name, name, channel_type = match
        try:
            channel_type = json.loads(channel_type)
        except json.JSONDecodeError:
            pass

        if type_utils.is_parameter_type(channel_type):
            pipeline_channel = PipelineParameterChannel(
                name=utils.sanitize_k8s_name(name),
                channel_type=channel_type,
                op_name=utils.sanitize_k8s_name(op_name),
            )
        else:
            pipeline_channel = PipelineArtifactChannel(
                name=utils.sanitize_k8s_name(name),
                channel_type=channel_type,
                op_name=utils.sanitize_k8s_name(op_name),
            )
        deduped_channels.add(pipeline_channel)

    return list(deduped_channels)


def extract_pipeline_channels_from_any(
    payload: Union['PipelineChannel', str, list, tuple, dict]
) -> List['PipelineChannel']:
    """Recursively extract PipelineChannels from any object or list of objects.

    Args:
      payload: An object that contains serialized PipelineChannels or k8
        definition objects.

    Returns:
      A list of PipelineChannels found from the payload.
    """
    if not payload:
        return []

    # PipelineChannel
    if isinstance(payload, PipelineChannel):
        return [payload]

    # str
    if isinstance(payload, str):
        return list(set(extract_pipeline_channels(payload)))

    # list or tuple
    if isinstance(payload, list) or isinstance(payload, tuple):
        pipeline_channels = []
        for item in payload:
            pipeline_channels += extract_pipeline_channels_from_any(item)
        return list(set(pipeline_channels))

    # dict
    if isinstance(payload, dict):
        pipeline_channels = []
        for key, value in payload.items():
            pipeline_channels += extract_pipeline_channels_from_any(key)
            pipeline_channels += extract_pipeline_channels_from_any(value)
        return list(set(pipeline_channels))

    # TODO(chensun): extract PipelineChannel from v2 container spec?

    return []
