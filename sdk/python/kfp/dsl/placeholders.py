# Copyright 2022 The Kubeflow Authors
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
"""Contains data structures and functions for handling input and output
placeholders."""

import abc
import json
from typing import Any, Dict, List, Optional, Union

from kfp import dsl
from kfp.dsl import utils
from kfp.dsl.types import type_utils


class Placeholder(abc.ABC):

    @abc.abstractmethod
    def _to_string(self) -> str:
        raise NotImplementedError

    def __str__(self) -> str:
        """Enables use of placeholders in f-strings.

        To be overridden by container placeholders ConcatPlaceholder and
        IfPresentPlaceholder, which cannot be used in an f-string.
        """
        return self._to_string()

    def __eq__(self, other: Any) -> bool:
        """Used for comparing placeholders in tests."""
        return isinstance(other,
                          self.__class__) and self.__dict__ == other.__dict__


class InputValuePlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.input_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.inputs.parameters['{self.input_name}']}}}}"


class InputListOfArtifactsPlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.input_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.inputs.artifacts['{self.input_name}']}}}}"

    def __getattribute__(self, name: str) -> Any:
        if name in {'name', 'uri', 'metadata', 'path'}:
            raise AttributeError(
                f'Cannot access an attribute on a list of artifacts in a Custom Container Component. Found reference to attribute {name!r} on {self.input_name!r}. Please pass the whole list of artifacts only.'
            )
        else:
            return object.__getattribute__(self, name)

    def __getitem__(self, k: int) -> None:
        raise KeyError(
            f'Cannot access individual artifacts in a list of artifacts. Found access to element {k} on {self.input_name!r}. Please pass the whole list of artifacts only.'
        )


class OutputListOfArtifactsPlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.output_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.outputs.artifacts['{self.output_name}']}}}}"

    def __getattribute__(self, name: str) -> Any:
        if name in {'name', 'uri', 'metadata', 'path'}:
            raise AttributeError(
                f'Cannot access an attribute on a list of artifacts in a Custom Container Component. Found reference to attribute {name!r} on {self.output_name!r}. Please pass the whole list of artifacts only.'
            )
        else:
            return object.__getattribute__(self, name)

    def __getitem__(self, k: int) -> None:
        raise KeyError(
            f'Cannot access individual artifacts in a list of artifacts. Found access to element {k} on {self.output_name!r}. Please pass the whole list of artifacts only.'
        )


class InputPathPlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.input_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.inputs.artifacts['{self.input_name}'].path}}}}"


class InputUriPlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.input_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.inputs.artifacts['{self.input_name}'].uri}}}}"


class InputMetadataPlaceholder(Placeholder):

    def __init__(self, input_name: str) -> None:
        self.input_name = input_name

    def _to_string(self) -> str:
        return f"{{{{$.inputs.artifacts['{self.input_name}'].metadata}}}}"

    def __getitem__(self, key: str) -> str:
        return f"{{{{$.inputs.artifacts['{self.input_name}'].metadata['{key}']}}}}"


class OutputParameterPlaceholder(Placeholder):

    def __init__(self, output_name: str) -> None:
        self.output_name = output_name

    def _to_string(self) -> str:
        return f"{{{{$.outputs.parameters['{self.output_name}'].output_file}}}}"


class OutputPathPlaceholder(Placeholder):

    def __init__(self, output_name: str) -> None:
        self.output_name = output_name

    def _to_string(self) -> str:
        return f"{{{{$.outputs.artifacts['{self.output_name}'].path}}}}"


class OutputUriPlaceholder(Placeholder):

    def __init__(self, output_name: str) -> None:
        self.output_name = output_name

    def _to_string(self) -> str:
        return f"{{{{$.outputs.artifacts['{self.output_name}'].uri}}}}"


class OutputMetadataPlaceholder(Placeholder):

    def __init__(self, output_name: str) -> None:
        self.output_name = output_name

    def _to_string(self) -> str:
        return f"{{{{$.outputs.artifacts['{self.output_name}'].metadata}}}}"

    def __getitem__(self, key: str) -> str:
        return f"{{{{$.outputs.artifacts['{self.output_name}'].metadata['{key}']}}}}"


class ConcatPlaceholder(Placeholder):
    """Placeholder for concatenating multiple strings. May contain other
    placeholders.

    Args:
        items: Elements to concatenate.

    Examples:
      ::

        @container_component
        def container_with_concat_placeholder(text1: str, text2: Output[Dataset],
                                              output_path: OutputPath(str)):
            return ContainerSpec(
                image='python:3.9',
                command=[
                    'my_program',
                    ConcatPlaceholder(['prefix-', text1, text2.uri])
                ],
                args=['--output_path', output_path]
            )
    """

    def __init__(self, items: List['CommandLineElement']) -> None:
        for item in items:
            if isinstance(item, IfPresentPlaceholder):
                item._validate_then_and_else_are_only_single_element()
        self.items = items

    def _to_dict(self) -> Dict[str, Any]:
        return {
            'Concat': [
                convert_command_line_element_to_string_or_struct(item)
                for item in self.items
            ]
        }

    def _to_string(self) -> str:
        return json.dumps(self._to_dict())

    def __str__(self) -> str:
        raise ValueError(
            f'Cannot use {self.__class__.__name__} in an f-string.')


class IfPresentPlaceholder(Placeholder):
    """Placeholder for handling cases where an input may or may not be passed.
    May contain other placeholders.

    Args:
        input_name: Name of the input/output.
        then: If the input/output specified in name is present, the command-line argument will be replaced at run-time by the value of then.
        else_: If the input/output specified in name is not present, the command-line argument will be replaced at run-time by the value of else_.

    Examples:
      ::

        @container_component
        def container_with_if_placeholder(output_path: OutputPath(str),
                                          dataset: Output[Dataset],
                                          optional_input: str = 'default'):
            return ContainerSpec(
                    image='python:3.9',
                    command=[
                        'my_program',
                        IfPresentPlaceholder(
                            input_name='optional_input',
                            then=[optional_input],
                            else_=['no_input']), '--dataset',
                        IfPresentPlaceholder(
                            input_name='optional_input', then=[dataset.uri], else_=['no_dataset'])
                    ],
                    args=['--output_path', output_path]
                )
    """

    def __init__(
        self,
        input_name: str,
        then: Union['CommandLineElement', List['CommandLineElement']],
        else_: Optional[Union['CommandLineElement',
                              List['CommandLineElement']]] = None,
    ) -> None:
        self.input_name = input_name
        self.then = then
        self.else_ = else_

    def _validate_then_and_else_are_only_single_element(self) -> None:
        """Rercursively validate that then and else contain only a single
        element.

        This method should only be called by a ConcatPlaceholder, which
        cannot have an IfPresentPlaceholder with a list in either 'then'
        or 'else_'.
        """

        # the illegal state
        if isinstance(self.then, list) or isinstance(self.else_, list):
            raise ValueError(
                f'Cannot use {IfPresentPlaceholder.__name__} within {ConcatPlaceholder.__name__} when `then` and `else_` arguments to {IfPresentPlaceholder.__name__} are lists. Please use a single element for `then` and `else_` only.'
            )

        # check that there is no illegal state found recursively
        if isinstance(self.then, ConcatPlaceholder):
            for item in self.then.items:
                if isinstance(item, IfPresentPlaceholder):
                    item._validate_then_and_else_are_only_single_element()
        elif isinstance(self.then, IfPresentPlaceholder):
            self.then._validate_then_and_else_are_only_single_element()

        if isinstance(self.else_, ConcatPlaceholder):
            for item in self.else_.items:
                if isinstance(item, IfPresentPlaceholder):
                    item._validate_then_and_else_are_only_single_element()
        elif isinstance(self.else_, IfPresentPlaceholder):
            self.else_._validate_then_and_else_are_only_single_element()

    def _to_dict(self) -> Dict[str, Any]:
        struct = {
            'IfPresent': {
                'InputName':
                    self.input_name,
                'Then': [
                    convert_command_line_element_to_string_or_struct(e)
                    for e in self.then
                ] if isinstance(self.then, list) else
                        convert_command_line_element_to_string_or_struct(
                            self.then)
            }
        }
        if self.else_:
            struct['IfPresent']['Else'] = [
                convert_command_line_element_to_string_or_struct(e)
                for e in self.else_
            ] if isinstance(
                self.else_,
                list) else convert_command_line_element_to_string_or_struct(
                    self.else_)
        return struct

    def _to_string(self) -> str:
        return json.dumps(self._to_dict())

    def __str__(self) -> str:
        raise ValueError(
            f'Cannot use {self.__class__.__name__} in an f-string.')


_CONTAINER_PLACEHOLDERS = (IfPresentPlaceholder, ConcatPlaceholder)
PRIMITIVE_INPUT_PLACEHOLDERS = (InputValuePlaceholder, InputPathPlaceholder,
                                InputUriPlaceholder, InputMetadataPlaceholder,
                                InputListOfArtifactsPlaceholder)
PRIMITIVE_OUTPUT_PLACEHOLDERS = (OutputParameterPlaceholder,
                                 OutputPathPlaceholder, OutputUriPlaceholder,
                                 OutputMetadataPlaceholder,
                                 OutputListOfArtifactsPlaceholder)

CommandLineElement = Union[str, Placeholder]


def convert_command_line_element_to_string(
        element: Union[str, Placeholder]) -> str:
    return element._to_string() if isinstance(element, Placeholder) else element


def convert_command_line_element_to_string_or_struct(
        element: Union[Placeholder, Any]) -> Any:
    if isinstance(element, Placeholder):
        return element._to_dict() if isinstance(
            element, _CONTAINER_PLACEHOLDERS) else element._to_string()

    return element


def maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
        arg: Dict[str, Any],
        component_dict: Dict[str, Any]) -> Union[CommandLineElement, Any]:
    if isinstance(arg, str):
        return arg

    if not isinstance(arg, dict):
        raise ValueError

    has_one_entry = len(arg) == 1

    if not has_one_entry:
        raise ValueError(
            f'Got unexpected dictionary {arg}. Expected a dictionary with one entry.'
        )

    first_key = list(arg.keys())[0]
    first_value = list(arg.values())[0]
    if first_key == 'inputValue':
        return InputValuePlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'inputPath':
        return InputPathPlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'inputUri':
        return InputUriPlaceholder(
            input_name=utils.sanitize_input_name(first_value))

    elif first_key == 'outputPath':
        outputs = component_dict['outputs']
        for output in outputs:
            if output['name'] == first_value:
                type_ = output.get('type')
                is_parameter = type_utils.is_parameter_type(type_)
                if is_parameter:
                    return OutputParameterPlaceholder(
                        output_name=utils.sanitize_input_name(first_value))
                else:
                    return OutputPathPlaceholder(
                        output_name=utils.sanitize_input_name(first_value))
        raise ValueError(
            f'{first_value} not found in component outputs. Could not process placeholders. Component spec: {component_dict}.'
        )

    elif first_key == 'outputUri':
        return OutputUriPlaceholder(
            output_name=utils.sanitize_input_name(first_value))

    elif first_key == 'ifPresent':
        structure_kwargs = arg['ifPresent']
        structure_kwargs['input_name'] = structure_kwargs.pop('inputName')
        structure_kwargs['otherwise'] = structure_kwargs.pop('else')
        structure_kwargs['then'] = [
            maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                e, component_dict=component_dict)
            for e in structure_kwargs['then']
        ]
        structure_kwargs['otherwise'] = [
            maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                e, component_dict=component_dict)
            for e in structure_kwargs['otherwise']
        ]
        return IfPresentPlaceholder(**structure_kwargs)

    elif first_key == 'concat':
        return ConcatPlaceholder(items=[
            maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                e, component_dict=component_dict) for e in arg['concat']
        ])

    elif first_key == 'executorInput':
        return dsl.PIPELINE_TASK_EXECUTOR_INPUT_PLACEHOLDER

    elif 'if' in arg:
        if_ = arg['if']
        input_name = utils.sanitize_input_name(if_['cond']['isPresent'])
        then = if_['then']
        else_ = if_.get('else')

        if isinstance(then, list):
            then = [
                maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                    val, component_dict=component_dict) for val in then
            ]
        else:
            then = maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                then, component_dict=component_dict)

        if else_ is None:
            pass
        elif isinstance(else_, list):
            else_ = [
                maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                    val, component_dict=component_dict) for val in else_
            ]
        else:
            maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                else_, component_dict=component_dict)

        return IfPresentPlaceholder(
            input_name=input_name, then=then, else_=else_)

    elif 'concat' in arg:

        return ConcatPlaceholder(items=[
            maybe_convert_v1_yaml_placeholder_to_v2_placeholder(
                val, component_dict=component_dict) for val in arg['concat']
        ])
    else:
        raise TypeError(f'Unexpected argument {arg} of type {type(arg)}.')
