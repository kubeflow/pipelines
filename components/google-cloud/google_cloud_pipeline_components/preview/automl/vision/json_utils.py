# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Utility functions for constructing JSON payload."""

import json
from typing import Any, Dict, List, Optional, Sequence, Union
from kfp.dsl import placeholders


_InputType = Union[
    placeholders.Placeholder, str, int, bool, float, List[Any], Dict[Any, Any]
]
_OutputType = Union[placeholders.Placeholder, str]


class _SmartConcatList:
  """List for building ConcatPlaceholder without unnecessary nesting."""

  _items: List[_OutputType] = []

  def Append(self, element: _OutputType) -> None:
    """Appends element to the list.

    1. If element is another ConcatPlaceholder, it will be flattened.
    2. If seq's last element is string and element is also a string, they will
      be concatenated.
    3. Otherwise, append directly.

    Args:
      element: Element to append to the list.
    """
    if isinstance(element, placeholders.ConcatPlaceholder):
      for item in element.items:
        self.Append(item)
    elif isinstance(element, str):
      if self._items and isinstance(self._items[-1], str):
        self._items[-1] += element
      else:
        self._items.append(element)
    elif isinstance(element, placeholders.Placeholder):
      self._items.append(element)
    else:
      raise ValueError(
          f'Element {element} is not supported by _SmartConcatList.'
      )

  def Extend(self, *args: _OutputType) -> None:
    for item in args:
      self.Append(item)

  def __init__(self, items: Sequence[_OutputType] = ()):
    self._items = []
    for item in items:
      self.Append(item)

  def Build(self) -> placeholders.ConcatPlaceholder:
    return placeholders.ConcatPlaceholder(self._items)


class IfPresent(placeholders.IfPresentPlaceholder):
  """Same as IfPresentPlaceholder but automatically converts outputs to JSON.

  If an object of this type is used as a dictionary value, the entire entry will
  be removed in case the corresponding input is not present.

  DO NOT use this as the first dictionary entry or the first item in a list.

  Attributes:
    input_name: Name of the input/output.
    then: If the input/output specified in name is present, the command-line
      argument will be replaced at run-time by the value of then.
    else_: If the input/output specified in name is not present, the
      command-line argument will be replaced at run-time by the value of else_.
  """

  then: _InputType
  else_: Optional[_InputType]

  def __init__(
      self,
      input_name: str,
      then: _InputType,
      else_: _InputType = None,
  ) -> None:
    then_obj = Json(then)
    else_obj = None if else_ is None else Json(else_)
    super().__init__(input_name, then_obj, else_obj)


def _DictEntry(
    key: _InputType, value: _InputType, is_first_item: bool
) -> placeholders.Placeholder:
  """Converts key, value to a placeholder that will evaluate to the JSON entry."""
  if isinstance(key, placeholders.Placeholder):
    raise ValueError('Placeholders are not supported as dictionary key.')
  key_obj = json.dumps(key) if isinstance(key, str) else f'"{json.dumps(key)}"'
  value_obj = Json(value)
  ret = _SmartConcatList([] if is_first_item else [','])
  ret.Extend(key_obj, ':')
  if isinstance(value_obj, IfPresent) and not value_obj.else_:
    if is_first_item:
      raise ValueError('IfPresent should not be used as the first dict entry.')
    # Removes the entire entry if input is not present.
    ret.Append(value_obj.then)
    return placeholders.IfPresentPlaceholder(
        input_name=value_obj.input_name,
        then=ret.Build(),
    )
  else:
    ret.Append(value_obj)
    return ret.Build()


def Concat(*args: _InputType) -> placeholders.ConcatPlaceholder:
  """Concats args by comma and returns the constructed ConcatPlaceholder."""
  ret = _SmartConcatList()
  is_first_item = True
  for el in args:
    obj = Json(el)
    if is_first_item:
      is_first_item = False
    elif isinstance(obj, placeholders.IfPresentPlaceholder):
      obj.then = _SmartConcatList([',', obj.then]).Build()
      if obj.else_:
        obj.else_ = _SmartConcatList([',', obj.else_]).Build()
    else:
      ret.Append(',')
    ret.Append(obj)
  return ret.Build()


def Json(obj: _InputType) -> _OutputType:
  """Converts obj to a placeholder that will evaluate to the JSON string."""
  if (
      isinstance(obj, str)
      or isinstance(obj, int)
      or isinstance(obj, float)
      or isinstance(obj, bool)
  ):
    return json.dumps(obj)
  if isinstance(obj, placeholders.Placeholder):
    if isinstance(obj, placeholders.ConcatPlaceholder) or isinstance(
        obj, placeholders.IfPresentPlaceholder
    ):
      return obj
    if isinstance(obj, placeholders.InputValuePlaceholder):
      # pylint: disable-next=protected-access
      if obj._ir_type != 'STRING':
        return obj
    # For string placeholders.
    return _SmartConcatList(['"', obj, '"']).Build()
  if isinstance(obj, list):
    return _SmartConcatList(['[', Concat(*obj), ']']).Build()
  if isinstance(obj, dict):
    concat_list = _SmartConcatList(['{'])
    is_first_item = True
    for key, value in obj.items():
      concat_list.Append(_DictEntry(key, value, is_first_item))
      is_first_item = False
    concat_list.Append('}')
    return concat_list.Build()
  raise ValueError(
      f'{obj} of type {type(obj)} is not supported by Json. Only Python'
      ' primitive types, lists, dicts, and kfp placeholders are supported.'
  )
