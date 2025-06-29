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
"""Classes and methods that supports argument for ParallelFor."""

import re
from typing import Any, Dict, List, Optional, Union

from kfp.dsl import pipeline_channel
from kfp.dsl.types import type_annotations
from kfp.dsl.types import type_utils

ItemList = List[Union[int, float, str, Dict[str, Any]]]

LOOP_ITEM_NAME_BASE = 'loop-item'
LOOP_ITEM_PARAM_NAME_BASE = 'loop-item-param'


def _get_loop_item_type(type_name: str) -> Optional[str]:
    """Extracts the loop item type.

    This method is used for extract the item type from a collection type.
    For example:

        List[str] -> str
        typing.List[int] -> int
        typing.Sequence[str] -> str
        List -> None
        str -> None

    Args:
        type_name: The collection type name, like `List`, Sequence`, etc.

    Returns:
        The collection item type or None if no match found.
    """
    match = re.match(r'(typing\.)?(?:\w+)(?:\[(?P<item_type>.+)\])', type_name)
    return match['item_type'].lstrip().rstrip() if match else None


def _get_subvar_type(type_name: str) -> Optional[str]:
    """Extracts the subvar type.

    This method is used for extract the value type from a dictionary type.
    For example:

        Dict[str, int] -> int
        typing.Mapping[str, float] -> float

    Args:
        type_name: The dictionary type.

    Returns:
        The dictionary value type or None if no match found.
    """
    match = re.match(
        r'(typing\.)?(?:\w+)(?:\[\s*(?:\w+)\s*,\s*(?P<value_type>.+)\])',
        type_name)
    return match['value_type'].lstrip().rstrip() if match else None


def _get_first_element_type(item_list: ItemList) -> str:
    """Returns the type of the first element of ItemList.

    Args:
        item_list: List of items to loop over. If a list of dicts then, all dicts must have the same keys.
    Returns:
        A string representing the type of the first element (e.g., "int", "Dict[str, int]").
    """
    first_element = item_list[0]
    if isinstance(first_element, dict):
        key_type = type(list(
            first_element.keys())[0]).__name__  # Get type of first key
        value_type = type(list(
            first_element.values())[0]).__name__  # Get type of first value
        return f'Dict[{key_type}, {value_type}]'
    else:
        return type(first_element).__name__


def _make_name(code: str) -> str:
    """Makes a name for a loop argument from a unique code."""
    return f'{LOOP_ITEM_PARAM_NAME_BASE}-{code}'


class LoopParameterArgument(pipeline_channel.PipelineParameterChannel):
    """Represents the parameter arguments that are looped over in a ParallelFor
    loop.

    The class shouldn't be instantiated by the end user, rather it is
    created automatically by a ParallelFor ops group.

    To create a LoopParameterArgument instance, use one of its factory methods::

        LoopParameterArgument.from_pipeline_channel(...)
        LoopParameterArgument.from_raw_items(...)


    Attributes:
        items_or_pipeline_channel: The raw items or the PipelineParameterChannel object
        this LoopParameterArgument is associated to.
    """

    def __init__(
        self,
        items: Union[ItemList, pipeline_channel.PipelineParameterChannel],
        name_code: Optional[str] = None,
        name_override: Optional[str] = None,
        **kwargs,
    ):
        """Initializes a LoopParameterArgument object.

        Args:
            items: List of items to loop over.  If a list of dicts then, all
                dicts must have the same keys and every key must be a legal
                Python variable name.
            name_code: A unique code used to identify these loop arguments.
                Should match the code for the ParallelFor ops_group which created
                these LoopParameterArguments. This prevents parameter name collisions.
            name_override: The override name for PipelineParameterChannel.
            **kwargs: Any other keyword arguments passed down to PipelineParameterChannel.
        """
        if (name_code is None) == (name_override is None):
            raise ValueError(
                'Expect one and only one of `name_code` and `name_override` to '
                'be specified.')

        if name_override is None:
            super().__init__(name=_make_name(name_code), **kwargs)
        else:
            super().__init__(name=name_override, **kwargs)

        if not isinstance(
                items,
            (list, tuple, pipeline_channel.PipelineParameterChannel)):
            raise TypeError(
                f'Expected list, tuple, or PipelineParameterChannel, got {items}.'
            )

        if isinstance(items, tuple):
            items = list(items)

        self.items_or_pipeline_channel = items
        self.is_with_items_loop_argument = not isinstance(
            items, pipeline_channel.PipelineParameterChannel)
        self._referenced_subvars: Dict[str, LoopArgumentVariable] = {}

        if isinstance(items, list) and isinstance(items[0], dict):
            subvar_names = set(items[0].keys())
            # then this block creates loop_arg.variable_a and loop_arg.variable_b
            for subvar_name in subvar_names:
                loop_arg_var = LoopArgumentVariable(
                    loop_argument=self,
                    subvar_name=subvar_name,
                )
                self._referenced_subvars[subvar_name] = loop_arg_var
                setattr(self, subvar_name, loop_arg_var)

    def __getattr__(self, name: str):
        # this is being overridden so that we can access subvariables of the
        # LoopParameterArgument (i.e.: item.a) without knowing the subvariable names ahead
        # of time.

        return self._referenced_subvars.setdefault(
            name, LoopArgumentVariable(
                loop_argument=self,
                subvar_name=name,
            ))

    @classmethod
    def from_pipeline_channel(
        cls,
        channel: pipeline_channel.PipelineParameterChannel,
    ) -> 'LoopParameterArgument':
        """Creates a LoopParameterArgument object from a
        PipelineParameterChannel object.

        Provide a flexible default channel_type ('String') if extraction
        from PipelineParameterChannel is unsuccessful. This maintains
        compilation progress in cases of unknown or missing type
        information.
        """
        # if channel is a LoopArgumentVariable, current system cannot check if
        # nested items are lists.
        if not isinstance(channel, LoopArgumentVariable):
            type_name = type_annotations.get_short_type_name(
                channel.channel_type)
            parameter_type = type_utils.PARAMETER_TYPES_MAPPING[
                type_name.lower()]
            if parameter_type != type_utils.LIST:
                raise ValueError(
                    'Cannot iterate over a single parameter using `dsl.ParallelFor`. Expected a list of parameters as argument to `items`.'
                )
        return LoopParameterArgument(
            items=channel,
            name_override=channel.name + '-' + LOOP_ITEM_NAME_BASE,
            task_name=channel.task_name,
            channel_type=_get_loop_item_type(channel.channel_type) or 'String',
        )

    @classmethod
    def from_raw_items(
        cls,
        raw_items: ItemList,
        name_code: str,
    ) -> 'LoopParameterArgument':
        """Creates a LoopParameterArgument object from raw item list."""
        if len(raw_items) == 0:
            raise ValueError('Got an empty item list for loop argument.')

        return LoopParameterArgument(
            items=raw_items,
            name_code=name_code,
            channel_type=_get_first_element_type(raw_items),
        )


class LoopArtifactArgument(pipeline_channel.PipelineArtifactChannel):
    """Represents the artifact arguments that are looped over in a ParallelFor
    loop.

    The class shouldn't be instantiated by the end user, rather it is
    created automatically by a ParallelFor ops group.

    To create a LoopArtifactArgument instance, use the factory method::

        LoopArtifactArgument.from_pipeline_channel(...)


    Attributes:
        pipeline_channel: The PipelineArtifactChannel object this
            LoopArtifactArgument is associated to.
    """

    def __init__(
        self,
        items: pipeline_channel.PipelineArtifactChannel,
        name_code: Optional[str] = None,
        name_override: Optional[str] = None,
        **kwargs,
    ):
        """Initializes a LoopArtifactArgument object.

        Args:
            items: The PipelineArtifactChannel object this LoopArtifactArgument is
                associated to.
            name_code: A unique code used to identify these loop arguments.
                Should match the code for the ParallelFor ops_group which created
                these LoopArtifactArguments. This prevents parameter name collisions.
            name_override: The override name for PipelineArtifactChannel.
            **kwargs: Any other keyword arguments passed down to PipelineArtifactChannel.
        """
        if (name_code is None) == (name_override is None):
            raise ValueError(
                'Expect one and only one of `name_code` and `name_override` to '
                'be specified.')

        # We don't support nested lists so `is_artifact_list` is always False.
        if name_override is None:
            super().__init__(
                name=_make_name(name_code), is_artifact_list=False, **kwargs)
        else:
            super().__init__(
                name=name_override, is_artifact_list=False, **kwargs)

        self.items_or_pipeline_channel = items
        self.is_with_items_loop_argument = not isinstance(
            items, pipeline_channel.PipelineArtifactChannel)

    @classmethod
    def from_pipeline_channel(
        cls,
        channel: pipeline_channel.PipelineArtifactChannel,
    ) -> 'LoopArtifactArgument':
        """Creates a LoopArtifactArgument object from a PipelineArtifactChannel
        object."""
        if not channel.is_artifact_list:
            raise ValueError(
                'Cannot iterate over a single artifact using `dsl.ParallelFor`. Expected a list of artifacts as argument to `items`.'
            )
        return LoopArtifactArgument(
            items=channel,
            name_override=channel.name + '-' + LOOP_ITEM_NAME_BASE,
            task_name=channel.task_name,
            channel_type=channel.channel_type,
        )

    # TODO: support artifact constants here.


class LoopArgumentVariable(pipeline_channel.PipelineParameterChannel):
    """Represents a subvariable for a loop argument.

    This is used for cases where we're looping over maps, each of which contains
    several variables. If the user ran:

        with dsl.ParallelFor([{'a': 1, 'b': 2}, {'a': 3, 'b': 4}]) as item:
            ...

    Then there's one LoopArgumentVariable for 'a' and another for 'b'.

    Attributes:
        loop_argument: The original LoopParameterArgument object this subvariable is
          attached to.
        subvar_name: The subvariable name.
    """
    SUBVAR_NAME_DELIMITER = '-subvar-'
    LEGAL_SUBVAR_NAME_REGEX = re.compile(r'^[a-zA-Z_][0-9a-zA-Z_]*$')

    def __init__(
        self,
        loop_argument: LoopParameterArgument,
        subvar_name: str,
    ):
        """Initializes a LoopArgumentVariable instance.

        Args:
            loop_argument: The LoopParameterArgument object this subvariable is based on
                a subvariable to.
            subvar_name: The name of this subvariable, which is the name of the
                dict key that spawned this subvariable.

        Raises:
            ValueError is subvar name is illegal.
        """
        if not self._subvar_name_is_legal(subvar_name):
            raise ValueError(
                f'Tried to create subvariable named {subvar_name}, but that is '
                'not a legal Python variable name.')

        self.subvar_name = subvar_name
        self.loop_argument = loop_argument
        # Handle potential channel_type extraction errors from LoopParameterArgument by defaulting to 'String'. This maintains compilation progress.
        super().__init__(
            name=self._get_name_override(
                loop_arg_name=loop_argument.name,
                subvar_name=subvar_name,
            ),
            task_name=loop_argument.task_name,
            channel_type=_get_subvar_type(loop_argument.channel_type) or
            'String',
        )

    @property
    def items_or_pipeline_channel(
            self) -> Union[ItemList, pipeline_channel.PipelineParameterChannel]:
        """Returns the loop argument items."""
        return self.loop_argument.items_or_pipeline_channel

    @property
    def is_with_items_loop_argument(self) -> bool:
        """Whether the loop argument is originated from raw items."""
        return self.loop_argument.is_with_items_loop_argument

    def _subvar_name_is_legal(self, proposed_variable_name: str) -> bool:
        """Returns True if the subvar name is legal."""
        return re.match(self.LEGAL_SUBVAR_NAME_REGEX,
                        proposed_variable_name) is not None

    def _get_name_override(self, loop_arg_name: str, subvar_name: str) -> str:
        """Gets the name.

        Args:
            loop_arg_name: the name of the loop argument parameter that this
              LoopArgumentVariable is attached to.
            subvar_name: The name of this subvariable.

        Returns:
            The name of this loop arg variable.
        """
        return f'{loop_arg_name}{self.SUBVAR_NAME_DELIMITER}{subvar_name}'


# TODO: migrate Collected to OneOfMixin style implementation
class Collected(pipeline_channel.PipelineChannel):
    """For collecting into a list the output from a task in dsl.ParallelFor
    loops.

    Args:
        output: The output of an upstream task within a dsl.ParallelFor loop.

    Example:
      ::

        @dsl.pipeline
        def math_pipeline() -> int:
            with dsl.ParallelFor([1, 2, 3]) as x:
                t = double(num=x)

        return add(nums=dsl.Collected(t.output)).output
    """

    def __init__(
        self,
        output: pipeline_channel.PipelineChannel,
    ) -> None:
        self.output = output
        # we know all dsl.Collected instances are lists, so set `is_artifact_list`
        # for type checking, which occurs before dsl.Collected is updated to
        # it's "correct" channel during compilation
        if isinstance(output, pipeline_channel.PipelineArtifactChannel):
            channel_type = output.channel_type
            self.is_artifact_channel = True
            self.is_artifact_list = True
        else:
            channel_type = 'LIST'
            self.is_artifact_channel = False
            self.is_artifact_list = False

        super().__init__(
            output.name,
            channel_type=channel_type,
            task_name=output.task_name,
        )
        self._validate_no_oneof_channel(self.output)

    def _validate_no_oneof_channel(
        self, channel: Union[pipeline_channel.PipelineParameterChannel,
                             pipeline_channel.PipelineArtifactChannel]
    ) -> None:
        if isinstance(channel, pipeline_channel.OneOfMixin):
            raise ValueError(
                f'dsl.{pipeline_channel.OneOf.__name__} cannot be used inside of dsl.{Collected.__name__}.'
            )
