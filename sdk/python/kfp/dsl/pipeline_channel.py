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

import abc
import contextlib
import dataclasses
import json
import re
from typing import Dict, List, Optional, Union

from kfp.dsl.types import type_utils


@dataclasses.dataclass
class ConditionOperation:
    """Represents a condition expression to be used in condition control flow
    group.

    Attributes:
      operator: The operator of the condition.
      left_operand: The left operand.
      right_operand: The right operand.
      negate: Whether to negate the result of the binary operation.
    """
    operator: str
    left_operand: Union['PipelineParameterChannel', type_utils.PARAMETER_TYPES]
    right_operand: Union['PipelineParameterChannel', type_utils.PARAMETER_TYPES]
    negate: bool = False


# The string template used to generate the placeholder of a PipelineChannel.
_PIPELINE_CHANNEL_PLACEHOLDER_TEMPLATE = (
    '{{channel:task=%s;name=%s;type=%s;}}')
# The regex for parsing PipelineChannel placeholders from a string.
_PIPELINE_CHANNEL_PLACEHOLDER_REGEX = (
    r'{{channel:task=([\w\s_-]*);name=([\w\s_-]+);type=([\w\s{}":_-]*);}}')


class PipelineChannel(abc.ABC):
    """Represents a future value that is passed between pipeline components.

    A PipelineChannel object can be used as a pipeline function argument so that
    it will be a pipeline artifact or parameter that shows up in ML Pipelines
    system UI. It can also represent an intermediate value passed between
    components.

    Attributes:
        name: The name of the pipeline channel.
        channel_type: The type of the pipeline channel.
        task_name: The name of the task that produces the pipeline channel.
            None means it is not produced by any task, so if None, either user
            constructs it directly (for providing an immediate value), or it is
            a pipeline function argument.
        pattern: The serialized string regex pattern this pipeline channel
            created from.
    """

    @abc.abstractmethod
    def __init__(
        self,
        name: str,
        channel_type: Union[str, Dict],
        task_name: Optional[str] = None,
    ):
        """Initializes a PipelineChannel instance.

        Args:
            name: The name of the pipeline channel. The name will be sanitized
                to be k8s compatible.
            channel_type: The type of the pipeline channel.
            task_name: Optional; The name of the task that produces the pipeline
                channel. If provided, the task name will be sanitized to be k8s
                compatible.

        Raises:
            ValueError: If name or task_name contains invalid characters.
            ValueError: If both task_name and value are set.
        """
        valid_name_regex = r'^[A-Za-z][A-Za-z0-9\s_-]*$'
        if not re.match(valid_name_regex, name):
            raise ValueError(
                f'Only letters, numbers, spaces, "_", and "-" are allowed in the name. Must begin with a letter. Got name: {name}'
            )

        self.name = name
        self.channel_type = channel_type
        # ensure value is None even if empty string or empty list/dict
        # so that serialization and unserialization remain consistent
        # (i.e. None => '' => None)
        self.task_name = task_name or None
        from kfp.dsl import pipeline_context

        self.pipeline = pipeline_context.Pipeline.get_default_pipeline()

    @property
    def task(self) -> Union['PipelineTask', 'TasksGroup']:
        # TODO: migrate Collected to OneOfMixin style implementation,
        # then move this out of a property
        if self.task_name is None or self.pipeline is None:
            return None

        if self.task_name in self.pipeline.tasks:
            return self.pipeline.tasks[self.task_name]

        from kfp.compiler import compiler_utils
        all_groups = compiler_utils.get_all_groups(self.pipeline.groups[0])
        # pipeline hasn't exited, so it doesn't have a name
        all_groups_no_pipeline = all_groups[1:]
        group_name_to_group = {
            group.name: group for group in all_groups_no_pipeline
        }
        if self.task_name in group_name_to_group:
            return group_name_to_group[self.task_name]

        raise ValueError(
            f"PipelineChannel task name '{self.task_name}' not found in pipeline."
        )

    @property
    def full_name(self) -> str:
        """Unique name for the PipelineChannel."""
        return f'{self.task_name}-{self.name}' if self.task_name else self.name

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
        placeholder '{{pipeline_channel:task=%s;name=%s;type=%s}}' with
        its own parameter identifier.
        """
        task_name = self.task_name or ''
        name = self.name
        channel_type = self.channel_type or ''
        if isinstance(channel_type, dict):
            channel_type = json.dumps(channel_type)
        return _PIPELINE_CHANNEL_PLACEHOLDER_TEMPLATE % (task_name, name,
                                                         channel_type)

    def __repr__(self) -> str:
        """Representation of the PipelineChannel.

        We make repr return the placeholder string so that if someone
        uses str()-based serialization of complex objects containing
        `PipelineChannel`, it works properly. (e.g. str([1, 2, 3,
        kfp.pipeline_channel.PipelineParameterChannel("aaa"), 4, 5, 6,]))
        """
        return str(self)

    def __hash__(self) -> int:
        """Returns the hash of a PipelineChannel."""
        return hash(self.pattern)

    def __eq__(self, other):
        return ConditionOperation('==', self, other)

    def __ne__(self, other):
        return ConditionOperation('!=', self, other)

    def __lt__(self, other):
        return ConditionOperation('<', self, other)

    def __le__(self, other):
        return ConditionOperation('<=', self, other)

    def __gt__(self, other):
        return ConditionOperation('>', self, other)

    def __ge__(self, other):
        return ConditionOperation('>=', self, other)


class PipelineParameterChannel(PipelineChannel):
    """Represents a pipeline parameter channel.

    Attributes:
      name: The name of the pipeline channel.
      channel_type: The type of the pipeline channel.
      task_name: The name of the task that produces the pipeline channel.
        None means it is not produced by any task, so if None, either user
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
        task_name: Optional[str] = None,
        value: Optional[type_utils.PARAMETER_TYPES] = None,
    ):
        """Initializes a PipelineArtifactChannel instance.

        Args:
          name: The name of the pipeline channel.
          channel_type: The type of the pipeline channel.
          task_name: Optional; The name of the task that produces the pipeline
            channel.
          value: Optional; The actual value of the pipeline channel.

        Raises:
          ValueError: If name or task_name contains invalid characters.
          ValueError: If both task_name and value are set.
          TypeError: If the channel type is not a parameter type.
        """
        if task_name and value:
            raise ValueError('task_name and value cannot be both set.')

        if not type_utils.is_parameter_type(channel_type):
            raise TypeError(f'{channel_type} is not a parameter type.')

        self.value = value

        super(PipelineParameterChannel, self).__init__(
            name=name,
            channel_type=channel_type,
            task_name=task_name,
        )


class PipelineArtifactChannel(PipelineChannel):
    """Represents a pipeline artifact channel.

    Attributes:
      name: The name of the pipeline channel.
      channel_type: The type of the pipeline channel.
      task_name: The name of the task that produces the pipeline channel.
        A pipeline artifact channel is always produced by some task.
      pattern: The serialized string regex pattern this pipeline channel created
        from.
    """

    def __init__(
        self,
        name: str,
        channel_type: Union[str, Dict],
        task_name: Optional[str],
        is_artifact_list: bool,
    ):
        """Initializes a PipelineArtifactChannel instance.

        Args:
            name: The name of the pipeline channel.
            channel_type: The type of the pipeline channel.
            task_name: Optional; the name of the task that produces the pipeline
                channel.
            is_artifact_list: True if `channel_type` represents a list of the artifact type.

        Raises:
            ValueError: If name or task_name contains invalid characters.
            TypeError: If the channel type is not an artifact type.
        """
        if type_utils.is_parameter_type(channel_type):
            raise TypeError(f'{channel_type} is not an artifact type.')

        self.is_artifact_list = is_artifact_list

        super(PipelineArtifactChannel, self).__init__(
            name=name,
            channel_type=channel_type,
            task_name=task_name,
        )


class OneOfMixin(PipelineChannel):
    """Shared functionality for OneOfParameter and OneOfAritfact."""

    def _set_condition_branches_group(
        self, channels: List[Union[PipelineParameterChannel,
                                   PipelineArtifactChannel]]
    ) -> None:
        # avoid circular import
        from kfp.dsl import tasks_group

        # .condition_branches_group could really be collapsed into just .task,
        # but we prefer keeping both for clarity in the rest of the compiler
        # code. When the code is logically related to a
        # condition_branches_group, it aids understanding to reference this
        # attribute name. When the code is trying to treat the OneOfMixin like
        # a typical PipelineChannel, it aids to reference task.
        self.condition_branches_group: tasks_group.ConditionBranches = channels[
            0].task.parent_task_group.parent_task_group

    def _make_oneof_name(self) -> str:
        # avoid circular imports
        from kfp.compiler import compiler_utils

        # This is a different type of "injected channel".
        # We know that this output will _always_ be a pipeline channel, so we
        # set the pipeline-channel-- prefix immediately (here).
        # In the downstream compiler logic, we get to treat this output like a
        # normal task output.
        return compiler_utils.additional_input_name_for_pipeline_channel(
            f'{self.condition_branches_group.name}-oneof-{self.condition_branches_group.get_oneof_id()}'
        )

    def _validate_channels(
        self,
        channels: List[Union[PipelineParameterChannel,
                             PipelineArtifactChannel]],
    ):
        self._validate_no_collected_channel(channels)
        self._validate_no_oneof_channel(channels)
        self._validate_no_mix_of_parameters_and_artifacts(channels)
        self._validate_has_else_group(self.condition_branches_group)

    def _validate_no_collected_channel(
        self, channels: List[Union[PipelineParameterChannel,
                                   PipelineArtifactChannel]]
    ) -> None:
        # avoid circular imports
        from kfp.dsl import for_loop
        if any(isinstance(channel, for_loop.Collected) for channel in channels):
            raise ValueError(
                f'dsl.{for_loop.Collected.__name__} cannot be used inside of dsl.{OneOf.__name__}.'
            )

    def _validate_no_oneof_channel(
        self, channels: List[Union[PipelineParameterChannel,
                                   PipelineArtifactChannel]]
    ) -> None:
        if any(isinstance(channel, OneOfMixin) for channel in channels):
            raise ValueError(
                f'dsl.{OneOf.__name__} cannot be used inside of another dsl.{OneOf.__name__}.'
            )

    def _validate_no_mix_of_parameters_and_artifacts(
        self, channels: List[Union[PipelineParameterChannel,
                                   PipelineArtifactChannel]]
    ) -> None:

        first_channel = channels[0]
        if isinstance(first_channel, PipelineParameterChannel):
            first_channel_type = PipelineParameterChannel
        else:
            first_channel_type = PipelineArtifactChannel

        for channel in channels:
            # if not all channels match the first channel's type, then there
            # is a mix of parameter and artifact channels
            if not isinstance(channel, first_channel_type):
                raise TypeError(
                    f'Task outputs passed to dsl.{OneOf.__name__} must be the same type. Found a mix of parameters and artifacts passed to dsl.{OneOf.__name__}.'
                )

    def _validate_has_else_group(
        self,
        parent_group: 'tasks_group.ConditionBranches',
    ) -> None:
        # avoid circular imports
        from kfp.dsl import tasks_group
        if not isinstance(parent_group.groups[-1], tasks_group.Else):
            raise ValueError(
                f'dsl.{OneOf.__name__} must include an output from a task in a dsl.{tasks_group.Else.__name__} group to ensure at least one output is available at runtime.'
            )

    def __str__(self):
        # supporting oneof in f-strings is technically feasible, but would
        # require somehow encoding all of the oneof channels into the
        # f-string
        # another way to do this would be to maintain a pipeline-level
        # map of PipelineChannels and encode a lookup key in the f-string
        # the combination of OneOf and an f-string is not common, so prefer
        # deferring implementation
        raise NotImplementedError(
            f'dsl.{OneOf.__name__} does not support string interpolation.')

    @property
    def pattern(self) -> str:
        # override self.pattern to avoid calling __str__, allowing us to block f-strings for now
        # this makes it OneOfMixin hashable for use in sets/dicts
        task_name = self.task_name or ''
        name = self.name
        channel_type = self.channel_type or ''
        if isinstance(channel_type, dict):
            channel_type = json.dumps(channel_type)
        return _PIPELINE_CHANNEL_PLACEHOLDER_TEMPLATE % (task_name, name,
                                                         channel_type)


# splitting out OneOf into subclasses significantly decreases the amount of
# branching in downstream compiler logic, since the
# isinstance(<some channel>, PipelineParameterChannel/PipelineArtifactChannel)
# checks continue to behave in desirable ways
class OneOfParameter(PipelineParameterChannel, OneOfMixin):
    """OneOf that results in a parameter channel for all downstream tasks."""

    def __init__(self, channels: List[PipelineParameterChannel]) -> None:
        self.channels = channels
        self._set_condition_branches_group(channels)
        super().__init__(
            name=self._make_oneof_name(),
            channel_type=channels[0].channel_type,
            task_name=None,
        )
        self.task_name = self.condition_branches_group.name
        self.channels = channels
        self._validate_channels(channels)
        self._validate_same_kfp_type(channels)

    def _validate_same_kfp_type(
            self, channels: List[PipelineParameterChannel]) -> None:
        expected_type = channels[0].channel_type
        for i, channel in enumerate(channels[1:], start=1):
            if channel.channel_type != expected_type:
                raise TypeError(
                    f'Task outputs passed to dsl.{OneOf.__name__} must be the same type. Got two channels with different types: {expected_type} at index 0 and {channel.channel_type} at index {i}.'
                )


class OneOfArtifact(PipelineArtifactChannel, OneOfMixin):
    """OneOf that results in an artifact channel for all downstream tasks."""

    def __init__(self, channels: List[PipelineArtifactChannel]) -> None:
        self.channels = channels
        self._set_condition_branches_group(channels)
        super().__init__(
            name=self._make_oneof_name(),
            channel_type=channels[0].channel_type,
            task_name=None,
            is_artifact_list=channels[0].is_artifact_list,
        )
        self.task_name = self.condition_branches_group.name
        self._validate_channels(channels)
        self._validate_same_kfp_type(channels)

    def _validate_same_kfp_type(
            self, channels: List[PipelineArtifactChannel]) -> None:
        # Unlike for component interface type checking where anything is
        # passable to Artifact, we should require the output artifacts for a
        # OneOf to be the same. This reduces the complexity/ambiguity for the
        # user of the actual type checking logic. What should the type checking
        # behavior be if the OneOf surfaces an Artifact and a Dataset? We can
        # always loosen backward compatibly in the future, so prefer starting
        # conservatively.
        expected_type = channels[0].channel_type
        expected_is_list = channels[0].is_artifact_list
        for i, channel in enumerate(channels[1:], start=1):
            if channel.channel_type != expected_type or channel.is_artifact_list != expected_is_list:
                raise TypeError(
                    f'Task outputs passed to dsl.{OneOf.__name__} must be the same type. Got two channels with different types: {expected_type} at index 0 and {channel.channel_type} at index {i}.'
                )


class OneOf:
    """For collecting mutually exclusive outputs from conditional branches into
    a single pipeline channel.

    Args:
        channels: The channels to collect into a OneOf. Must be of the same type.

    Example:
      ::

        @dsl.pipeline
        def flip_coin_pipeline() -> str:
            flip_coin_task = flip_coin()
            with dsl.If(flip_coin_task.output == 'heads'):
                print_task_1 = print_and_return(text='Got heads!')
            with dsl.Else():
                print_task_2 = print_and_return(text='Got tails!')

            # use the output from the branch that gets executed
            oneof = dsl.OneOf(print_task_1.output, print_task_2.output)

            # consume it
            print_and_return(text=oneof)

            # return it
            return oneof
    """

    def __new__(
        cls, *channels: Union[PipelineParameterChannel, PipelineArtifactChannel]
    ) -> Union[OneOfParameter, OneOfArtifact]:
        first_channel = channels[0]
        if isinstance(first_channel, PipelineParameterChannel):
            return OneOfParameter(channels=list(channels))
        elif isinstance(first_channel, PipelineArtifactChannel):
            return OneOfArtifact(channels=list(channels))
        else:
            raise ValueError(
                f'Got unknown input to dsl.{OneOf.__name__} with type {type(first_channel)}.'
            )


def create_pipeline_channel(
    name: str,
    channel_type: Union[str, Dict],
    task_name: Optional[str] = None,
    value: Optional[type_utils.PARAMETER_TYPES] = None,
    is_artifact_list: bool = False,
) -> PipelineChannel:
    """Creates a PipelineChannel object.

    Args:
        name: The name of the channel.
        channel_type: The type of the channel, which decides whether it is an
            PipelineParameterChannel or PipelineArtifactChannel
        task_name: Optional; the task that produced the channel.
        value: Optional; the realized value for a channel.

    Returns:
        A PipelineParameterChannel or PipelineArtifactChannel object.
    """
    if type_utils.is_parameter_type(channel_type):
        return PipelineParameterChannel(
            name=name,
            channel_type=channel_type,
            task_name=task_name,
            value=value,
        )
    else:
        return PipelineArtifactChannel(
            name=name,
            channel_type=channel_type,
            task_name=task_name,
            is_artifact_list=is_artifact_list,
        )


def extract_pipeline_channels_from_string(
        payload: str) -> List[PipelineChannel]:
    """Extracts a list of PipelineChannel instances from the payload string.

    Note: this function removes all duplicate matches.

    Args:
      payload: A string that may contain serialized PipelineChannels.

    Returns:
      A list of PipelineChannels found from the payload.
    """
    matches = re.findall(_PIPELINE_CHANNEL_PLACEHOLDER_REGEX, payload)
    unique_channels = set()
    for match in matches:
        task_name, name, channel_type = match

        # channel_type could be either a string (e.g. "Integer") or a dictionary
        # (e.g.: {"custom_type": {"custom_property": "some_value"}}).
        # Try loading it into dictionary, if failed, it means channel_type is a
        # string.
        with contextlib.suppress(json.JSONDecodeError):
            channel_type = json.loads(channel_type)

        if type_utils.is_parameter_type(channel_type):
            pipeline_channel = PipelineParameterChannel(
                name=name,
                channel_type=channel_type,
                task_name=task_name,
            )
        else:
            pipeline_channel = PipelineArtifactChannel(
                name=name,
                channel_type=channel_type,
                task_name=task_name,
                # currently no support for getting the index from a list of artifacts (e.g., my_datasets[0].uri), so this will always be False until accessing a single artifact element is supported
                is_artifact_list=False,
            )
        unique_channels.add(pipeline_channel)

    return list(unique_channels)


def extract_pipeline_channels_from_any(
    payload: Union[PipelineChannel, str, int, float, bool, list, tuple, dict]
) -> List[PipelineChannel]:
    """Recursively extract PipelineChannels from any object or list of objects.

    Args:
      payload: An object that contains serialized PipelineChannels or k8
        definition objects.

    Returns:
      A list of PipelineChannels found from the payload.
    """
    if not payload:
        return []

    if isinstance(payload, PipelineChannel):
        return [payload]

    if isinstance(payload, str):
        return list(set(extract_pipeline_channels_from_string(payload)))

    if isinstance(payload, (list, tuple)):
        pipeline_channels = []
        for item in payload:
            pipeline_channels += extract_pipeline_channels_from_any(item)
        return list(set(pipeline_channels))

    if isinstance(payload, dict):
        pipeline_channels = []
        for key, value in payload.items():
            pipeline_channels += extract_pipeline_channels_from_any(key)
            pipeline_channels += extract_pipeline_channels_from_any(value)
        return list(set(pipeline_channels))

    # TODO(chensun): extract PipelineChannel from v2 container spec?

    return []
