"""A convenience class for referencing evaluation data."""

from typing import Any, Dict, List, Text, Union

from lib.proto import configuration_pb2
from google.protobuf import json_format

SUBKEY_SEPARATOR = '__'


class ColumnSpec:
  """Wrapper around configuration_pb2.ColumnSpec.

  Conceptually, this class represents a list of subkeys required to access a
  particular field in a dictionary, and methods of working woth this list.
  For example, if a dictionary is {'car': {'make': 'Ariel'}}, the list of
  subkeys needed to access the make of a car is ['car', 'make'].

  Attributes:
    _subkeys: A list of subkeys identifying a field in a dictionary.
  """

  def __init__(self, name_spec: Union[Text, List[Text],
                                      configuration_pb2.ColumnSpec]):
    """Initializes object from different representations.

    Args:
      name_spec: one of the three representations:
        * string representation in the same format as the string returned by
          as_string() method of this class.
        * list representation: a non-empty list of subkeys.
        * proto representation: and instance of configuration_pb2.ColumnSpec
          proto.
    """
    if isinstance(name_spec, configuration_pb2.ColumnSpec):
      # Column spec is a ColumnSpec proto.
      self._subkeys = self._column_spec_to_subkeys(
          column_spec=name_spec, current_subkeys=[])
    elif isinstance(name_spec, str):
      # Column spec is a string containing concatenated subkeys. The string
      # should be in the same format as as_string() method of this class
      # produces.
      self._subkeys = name_spec.split(SUBKEY_SEPARATOR)
    else:
      # Column spec is already a list of subkeys.
      self._subkeys = name_spec

    if not self._subkeys:
      raise ValueError('subkeys list must be provided.')

    if not all(self._subkeys):
      raise ValueError(f'subkeys list {self._subkeys} contains an empty key. ')

  def _column_spec_to_subkeys(self, column_spec: configuration_pb2.ColumnSpec,
                              current_subkeys: List[Text]) -> List[Text]:
    """Recursively converts ColumnSpec from proto to a list of keys."""
    if not column_spec.name:
      raise ValueError(f'name is missing from ColumnSpec {column_spec}.')

    current_subkeys.append(column_spec.name)

    if not column_spec.HasField('sub_column_spec'):
      return current_subkeys

    return self._column_spec_to_subkeys(
        column_spec=column_spec.sub_column_spec,
        current_subkeys=current_subkeys)

  def as_string(self) -> Text:
    """Returns string representation of ColumnSpec."""
    return SUBKEY_SEPARATOR.join(self._subkeys)

  def as_proto(self) -> configuration_pb2.ColumnSpec:
    """Returns proto representation of ColumnSpec."""
    spec = configuration_pb2.ColumnSpec()
    current_spec = spec
    current_spec.name = self._subkeys[0]
    for key in self._subkeys[1:]:
      current_spec = current_spec.sub_column_spec
      current_spec.name = key

    return spec

  def as_dict(self) -> Dict[str, Any]:
    """Returns dictionary representation of ColumnSpec."""
    return json_format.MessageToDict(self.as_proto())

  def get_value_from_dict(self, json_dict: Dict[Text, Any]) -> Any:
    """Extracts column value from a dictionary."""
    return self._get_value_from_dict(
        json_dict,
        subkeys=self._subkeys,
        processed_subkeys=[],
        should_pop_value=False)

  def pop_value_from_dict(self, json_dict: Dict[Text, Any]) -> Any:
    """Pops column value from a dictionary."""
    return self._get_value_from_dict(
        json_dict,
        subkeys=self._subkeys,
        processed_subkeys=[],
        should_pop_value=True)

  def _get_value_from_dict(self, json_dict: Dict[Text, Any],
                           subkeys: List[Text], processed_subkeys: List[Text],
                           should_pop_value: bool) -> Any:
    """Gets or pops a column value from a dictionary."""
    subkey = subkeys[0]
    if subkey not in json_dict:
      key_string = self._list_to_key_string(processed_subkeys)
      raise KeyError(
          f'key {subkey} is not in json_dict{key_string}. json_dict is {json_dict}'
      )

    if len(subkeys) == 1:
      if should_pop_value:
        return json_dict.pop(subkey)

      return json_dict[subkey]

    return self._get_value_from_dict(
        json_dict[subkey],
        subkeys=subkeys[1:],
        processed_subkeys=processed_subkeys + [subkey],
        should_pop_value=should_pop_value)

  def _list_to_key_string(self, key_list: List[Text]) -> Text:
    """Creates a string demonstrating how the key list was used on a dictionary.

    For example, if the key_list is ['a', 'b'], then the retuned string will be
    ['a']['b']. Used to create more informative error messages.

    Args:
      key_list: list of keys used to access values in a dictionary.

    Returns:
      A string demonstrating how the key list was used on a dictionary.
    """
    return ''.join('[{}]'.format(key) for key in key_list)

  def exists_in_dict(self, json_dict: Dict[Text, Any]) -> bool:
    """Checks if the key exists in a dictionary."""
    return self._exists_in_dict(json_dict, subkeys=self._subkeys)

  def _exists_in_dict(self, json_dict: Dict[Text, Any],
                      subkeys: List[Text]) -> bool:
    """Implements the checks of whether the key exists in a dictionary."""
    subkey = subkeys[0]
    if subkey not in json_dict:
      # Current subkey is not in the dictionary.
      return False

    item = json_dict[subkey]
    n_subkeys = len(subkeys)
    if not isinstance(item, dict) and n_subkeys > 1:
      # We still have subkeys to process but there is no dictionary to descend
      # into
      return False

    if n_subkeys == 1:
      # We've processed all the subkeys.
      return True

    return self._exists_in_dict(json_dict=item, subkeys=subkeys[1:])
