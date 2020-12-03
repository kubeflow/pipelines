# Copyright 2020 Google LLC
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

__all__ = [
    'ComponentSpec',
    'ConcatPlaceholder',
    'ContainerImplementation',
    'ContainerSpec',
    'IfPlaceholder',
    'IfPlaceholderStructure',
    'InputPathPlaceholder',
    'InputSpec',
    'InputUriPlaceholder',
    'InputValuePlaceholder',
    'IsPresentPlaceholder',
    'OutputPathPlaceholder',
    'OutputSpec',
    'OutputUriPlaceholder',
]

from typing import Any, Dict, List, Mapping, Optional, Union
import warnings

from kfp.components.modelbase import ModelBase

PrimitiveTypes = Union[str, int, float, bool]
PrimitiveTypesIncludingNone = Optional[PrimitiveTypes]

TypeSpecType = Union[str, Dict, List]


class InputSpec(ModelBase):
  """Describes the component input specification."""

  def __init__(
      self,
      name: str,
      type: Optional[TypeSpecType] = None,
      description: Optional[str] = None,
      default: Optional[PrimitiveTypes] = None,
      optional: Optional[bool] = False,
      annotations: Optional[Dict[str, Any]] = None,
  ):
    super().__init__(locals())


class OutputSpec(ModelBase):
  """Describes the component output specification."""

  def __init__(
      self,
      name: str,
      type: Optional[TypeSpecType] = None,
      description: Optional[str] = None,
      annotations: Optional[Dict[str, Any]] = None,
  ):
    super().__init__(locals())


class InputUriPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the URI of an input artifact.

  Represents the command-line argument placeholder that will be replaced at
  run-time by the URI of the input artifact argument.
  """
  _serialized_names = {
      'input_name': 'inputUri',
  }

  def __init__(
      self,
      input_name: str,
  ):
    super().__init__(locals())


class InputValuePlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the value of an input.

  Represents the command-line argument placeholder that will be replaced at
  run-time by the input argument value.
  """
  _serialized_names = {
      'input_name': 'inputValue',
  }

  def __init__(
      self,
      input_name: str,
  ):
    super().__init__(locals())


def _custom_formatwarning(message, category, filename, lineno, line=None):
  return '%s:%s: %s: %s\n' % (filename, lineno, category.__name__, message)


class InputPathPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the path of an input.

  Represents the command-line argument placeholder that will be replaced at
  run-time by a local file path pointing to a file containing the input argument
  value.
  """
  _serialized_names = {
      'input_name': 'inputPath',
  }

  def __init__(
      self,
      input_name: str,
  ):
    super().__init__(locals())
    formatwarning_orig = warnings.formatwarning
    warnings.formatwarning = _custom_formatwarning
    warnings.warn(
        'Local file paths are currently unsupported for I/O. Please ensure your '
        'component is able to read/write to Google Cloud Storage (using gsutil, '
        'tf.gfile, or other similar libraries).')
    warnings.formatwarning = formatwarning_orig


class OutputPathPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the path of an input.

  Represents the command-line argument placeholder that will be replaced at
  run-time by a local file path pointing to a file where the program should
  write its output data.
  """
  _serialized_names = {
      'output_name': 'outputPath',
  }

  def __init__(
      self,
      output_name: str,
  ):
    super().__init__(locals())
    formatwarning_orig = warnings.formatwarning
    warnings.formatwarning = _custom_formatwarning
    warnings.warn(
        'Local file paths are currently unsupported for I/O. Please ensure your '
        'component is able to read/write to Google Cloud Storage (using gsutil, '
        'tf.gfile, or other similar libraries).')
    warnings.formatwarning = formatwarning_orig


class OutputUriPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the URI of an output artifact.

  Represents the command-line argument placeholder that will be replaced at
  run-time by a URI of the output artifac where the program should write its
  output data.
  """
  _serialized_names = {
      'output_name': 'outputUri',
  }

  def __init__(
      self,
      output_name: str,
  ):
    super().__init__(locals())


CommandlineArgumentType = Union[str, InputUriPlaceholder, InputValuePlaceholder,
                                InputPathPlaceholder, OutputUriPlaceholder,
                                OutputPathPlaceholder, 'ConcatPlaceholder',
                                'IfPlaceholder',]


class ConcatPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for the concatenation of items.

  Represents the command-line argument placeholder that will be replaced at
  run-time by the concatenated values of its items.
  """
  _serialized_names = {
      'items': 'concat',
  }

  def __init__(
      self,
      items: List[CommandlineArgumentType],
  ):
    super().__init__(locals())


class IsPresentPlaceholder(ModelBase):  # Non-standard attr names
  """Represents a placeholder for if an argument presents.

  Represents the command-line argument placeholder that will be replaced at
  run-time by a boolean value specifying whether the caller has passed an
  argument for the specified optional input.
  """
  _serialized_names = {
      'input_name': 'isPresent',
  }

  def __init__(
      self,
      input_name: str,
  ):
    super().__init__(locals())


IfConditionArgumentType = Union[bool, str, IsPresentPlaceholder,
                                InputValuePlaceholder]


class IfPlaceholderStructure(ModelBase):  # Non-standard attr names
  """Represents the structure of IfPlaceholder.

  Used in by the IfPlaceholder - the command-line argument placeholder that will
  be replaced at run-time by the expanded value of either "then_value" or
  "else_value" depending on the submissio-time resolved value of the "cond"
  predicate.
  """
  _serialized_names = {
      'condition': 'cond',
      'then_value': 'then',
      'else_value': 'else',
  }

  def __init__(
      self,
      condition: IfConditionArgumentType,
      then_value: Union[CommandlineArgumentType, List[CommandlineArgumentType]],
      else_value: Optional[Union[CommandlineArgumentType,
                                 List[CommandlineArgumentType]]] = None,
  ):
    super().__init__(locals())


class IfPlaceholder(ModelBase):  # Non-standard attr names
  """Represents the placeholder for if statement.

  Represents the command-line argument placeholder that will be replaced at
  run-time by the expanded value of either "then_value" or "else_value"
  depending on the submissio-time resolved value of the "cond" predicate.
  """
  _serialized_names = {
      'if_structure': 'if',
  }

  def __init__(
      self,
      if_structure: IfPlaceholderStructure,
  ):
    super().__init__(locals())


class ContainerSpec(ModelBase):
  """Describes the container component implementation."""
  _serialized_names = {
      'file_outputs':
          'fileOutputs',  # TODO: rename to something like legacy_unconfigurable_output_paths
  }

  def __init__(
      self,
      image: str,
      command: Optional[List[CommandlineArgumentType]] = None,
      args: Optional[List[CommandlineArgumentType]] = None,
      env: Optional[Mapping[str, str]] = None,
      file_outputs: Optional[Mapping[
          str,
          str]] = None,  # TODO: rename to something like legacy_unconfigurable_output_paths
  ):
    super().__init__(locals())


class ContainerImplementation(ModelBase):
  """Represents the container component implementation."""

  def __init__(
      self,
      container: ContainerSpec,
  ):
    super().__init__(locals())


# ImplementationType doesn't support GraphImplementation in v2 yet.
ImplementationType = Union[ContainerImplementation]


class MetadataSpec(ModelBase):

  def __init__(
      self,
      annotations: Optional[Dict[str, str]] = None,
      labels: Optional[Dict[str, str]] = None,
  ):
    super().__init__(locals())


class ComponentSpec(ModelBase):
  """Component specification.

  Describes the metadata (name, description, annotations and labels), the
  interface (inputs and outputs) and the implementation of the component.
  """

  def __init__(
      self,
      name: Optional[str] = None,  # ? Move to metadata?
      description: Optional[str] = None,  # ? Move to metadata?
      metadata: Optional[MetadataSpec] = None,
      inputs: Optional[List[InputSpec]] = None,
      outputs: Optional[List[OutputSpec]] = None,
      implementation: Optional[ImplementationType] = None,
      version: Optional[str] = 'google.com/cloud/pipelines/component/v1',
      # tags: Optional[Set[str]] = None,
  ):
    super().__init__(locals())
    self._post_init()

  def _post_init(self):
    # Checking input names for uniqueness
    self._inputs_dict = {}
    if self.inputs:
      for input in self.inputs:
        if input.name in self._inputs_dict:
          raise ValueError('Non-unique input name "{}"'.format(input.name))
        self._inputs_dict[input.name] = input

    # Checking output names for uniqueness
    self._outputs_dict = {}
    if self.outputs:
      for output in self.outputs:
        if output.name in self._outputs_dict:
          raise ValueError('Non-unique output name "{}"'.format(output.name))
        self._outputs_dict[output.name] = output

    if isinstance(self.implementation, ContainerImplementation):
      container = self.implementation.container

      if container.file_outputs:
        for output_name, path in container.file_outputs.items():
          if output_name not in self._outputs_dict:
            raise TypeError(
                'Unconfigurable output entry "{}" references non-existing output.'
                .format({output_name: path}))

      def verify_arg(arg):
        if arg is None:
          pass
        elif isinstance(arg, (str, int, float, bool)):
          pass
        elif isinstance(arg, list):
          for arg2 in arg:
            verify_arg(arg2)
        elif isinstance(arg, (InputUriPlaceholder, InputValuePlaceholder,
                              InputPathPlaceholder, IsPresentPlaceholder)):
          if arg.input_name not in self._inputs_dict:
            raise TypeError(
                'Argument "{}" references non-existing input.'.format(arg))
        elif isinstance(arg, (OutputUriPlaceholder, OutputPathPlaceholder)):
          if arg.output_name not in self._outputs_dict:
            raise TypeError(
                'Argument "{}" references non-existing output.'.format(arg))
        elif isinstance(arg, ConcatPlaceholder):
          for arg2 in arg.items:
            verify_arg(arg2)
        elif isinstance(arg, IfPlaceholder):
          verify_arg(arg.if_structure.condition)
          verify_arg(arg.if_structure.then_value)
          verify_arg(arg.if_structure.else_value)
        else:
          raise TypeError('Unexpected argument "{}"'.format(arg))

      verify_arg(container.command)
      verify_arg(container.args)

  def save(self, file_path: str):
    """Saves the component definition to file.

    It can be shared online and later loaded using the load_component
    function.
    """
    from kfp.components._yaml_utils import dump_yaml
    component_yaml = dump_yaml(self.to_dict())
    with open(file_path, 'w') as f:
      f.write(component_yaml)
