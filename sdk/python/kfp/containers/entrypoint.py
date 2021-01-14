# Copyright 2021 Google LLC
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

from typing import Optional, Union

from absl import logging
import fire
from google.protobuf import json_format
import os

from kfp.containers import _gcs_helper
from kfp.containers import entrypoint_utils
from kfp.dsl import artifact

_FN_SOURCE = 'ml/main.py'
_FN_NAME_ARG = 'function_name'

_PARAM_METADATA_SUFFIX = '_input_param_metadata_file'
_ARTIFACT_METADATA_SUFFIX = '_input_artifact_metadata_file'
_FIELD_NAME_SUFFIX = '_input_field_name'
_ARGO_PARAM_SUFFIX = '_input_argo_param'
_INPUT_PATH_SUFFIX = '_input_path'
_OUTPUT_NAME_SUFFIX = '_input_output_name'

_OUTPUT_PARAM_PATH_SUFFIX = '_parameter_output_path'
_OUTPUT_ARTIFACT_PATH_SUFFIX = '_artifact_output_path'

_METADATA_FILE_ARG = 'executor_metadata_json_file'


class InputParam(object):
  """POD that holds an input parameter."""

  def __init__(self,
      value: Optional[Union[str, float, int]] = None,
      metadata_file: Optional[str] = None,
      field_name: Optional[str] = None):
    """Instantiates an InputParam object.

    Args:
      value: The actual value of the parameter.
      metadata_file: The location of the metadata JSON file output by the
        producer step.
      field_name: The output name of the producer.

    Raises:
      ValueError: when neither of the following is true:
        1) value is provided, and metadata_file and field_name are not; or
        2) both metadata_file and field_name are provided, and value is not.
    """
    if not (value is not None and not (metadata_file or field_name) or (
        metadata_file and field_name and value is None)):
      raise ValueError('Either value or both metadata_file and field_name '
                       'needs to be provided. Got value={value}, field_name='
                       '{field_name}, metadata_file={metadata_file}'.format(
          value=value,
          field_name=field_name,
          metadata_file=metadata_file
      ))
    if value is not None:
      self._value = value
    else:
      # Parse the value by inspecting the producer's metadata JSON file.
      self._value = entrypoint_utils.get_parameter_from_output(
          metadata_file, field_name)

    self._metadata_file = metadata_file
    self._field_name = field_name

  # Following properties are read-only
  @property
  def value(self) -> Union[float, str, int]:
    return self._value

  @property
  def metadata_file(self) -> str:
    return self._metadata_file

  @property
  def field_name(self) -> str:
    return self._field_name


class InputArtifact(object):
  """POD that holds an input artifact."""

  def __init__(self,
      uri: Optional[str] = None,
      metadata_file: Optional[str] = None,
      output_name: Optional[str] = None
  ):
    """Instantiates an InputParam object.

    Args:
      uri: The uri holds the input artifact.
      metadata_file: The location of the metadata JSON file output by the
        producer step.
      output_name: The output name of the artifact in producer step.

    Raises:
      ValueError: when neither of the following is true:
        1) uri is provided, and metadata_file and output_name are not; or
        2) both metadata_file and output_name are provided, and uri is not.
    """
    if not ((uri and not (metadata_file or output_name) or (
        metadata_file and output_name and not uri))):
      raise ValueError('Either uri or both metadata_file and output_name '
                       'needs to be provided. Got uri={uri}, output_name='
                       '{output_name}, metadata_file={metadata_file}'.format(
          uri=uri,
          output_name=output_name,
          metadata_file=metadata_file
      ))

    self._metadata_file = metadata_file
    self._output_name = output_name
    if uri:
      self._uri = uri
    else:
      self._uri = self.get_artifact().uri

  # Following properties are read-only.
  @property
  def uri(self) -> str:
    return self._uri

  @property
  def metadata_file(self) -> str:
    return self._metadata_file

  @property
  def output_name(self) -> str:
    return self._output_name

  def get_artifact(self) -> artifact.Artifact:
    """Gets an artifact object by parsing metadata or creating one from uri."""
    if self.metadata_file and self.output_name:
      return entrypoint_utils.get_artifact_from_output(
          self.metadata_file, self.output_name)
    else:
      # Provide an empty schema when returning a raw Artifact.
      result = artifact.Artifact(
          instance_schema=artifact.DEFAULT_ARTIFACT_SCHEMA)
      result.uri = self.uri
      return result


def main(**kwargs):
  """Container entrypoint used by KFP Python function based component.

  This function has a dynamic signature, which will be interpreted according to
  the I/O and data-passing contract of KFP Python function components. The
  parameter will be received from command line interface.

  For each declared parameter input of the user function, three command line
  arguments will be recognized:
  1. {name of the parameter}_input_param_metadata_file: The metadata JSON file
     path output by the producer.
  2. {name of the parameter}_input_field_name: The output name of the parameter,
     by which the parameter can be found in the producer metadata JSON file.
  3. {name of the parameter}_input_argo_param: The actual runtime value of the
     input parameter.
  When the producer is a new-styled KFP Python component, 1 and 2 will be
  populated, and when it's a conventional KFP Python component, 3 will be in
  use.

  For each declared artifact input of the user function, three command line args
  will be recognized:
  1. {name of the artifact}_input_path: The actual path, or uri, of the input
     artifact.
  2. {name of the artifact}_input_artifact_metadata_file: The metadata JSON file
     path output by the producer.
  3. {name of the artifact}_input_output_name: The output name of the artifact,
     by which the artifact can be found in the producer metadata JSON file.
  If the producer is a new-styled KFP Python component, 2+3 will be used to give
  user code access to MLMD (custom) properties associated with this artifact;
  if the producer is a conventional KFP Python component, 1 will be used to
  construct an Artifact with only the URI populated.

  For each declared artifact or parameter output of the user function, a command
  line arg, namely, `{name of the artifact|parameter}_(artifact|parameter)_output_path`,
  will be passed to specify the location where the output content is written to.

  In addition, `executor_metadata_json_file` specifies the location where the
  output metadata JSON file will be written.
  """
  if _METADATA_FILE_ARG not in kwargs:
    raise RuntimeError('Must specify executor_metadata_json_file')

  # Group arguments according to suffixes.
  input_params_metadata = {}
  input_params_field_name = {}
  input_params_value = {}
  input_artifacts_metadata = {}
  input_artifacts_uri = {}
  input_artifacts_output_name = {}
  output_artifacts_uri = {}
  output_params_path = {}
  for k, v in kwargs.items():
    if k.endswith(_PARAM_METADATA_SUFFIX):
      param_name = k[:-len(_PARAM_METADATA_SUFFIX)]
      input_params_metadata[param_name] = v
    elif k.endswith(_FIELD_NAME_SUFFIX):
      param_name = k[:-len(_FIELD_NAME_SUFFIX)]
      input_params_field_name[param_name] = v
    elif k.endswith(_ARGO_PARAM_SUFFIX):
      param_name = k[:-len(_ARGO_PARAM_SUFFIX)]
      input_params_value[param_name] = v
    elif k.endswith(_ARTIFACT_METADATA_SUFFIX):
      artifact_name = k[:-len(_ARTIFACT_METADATA_SUFFIX)]
      input_artifacts_metadata[artifact_name] = v
    elif k.endswith(_INPUT_PATH_SUFFIX):
      artifact_name = k[:-len(_INPUT_PATH_SUFFIX)]
      input_artifacts_uri[artifact_name] = v
    elif k.endswith(_OUTPUT_NAME_SUFFIX):
      artifact_name = k[:-len(_OUTPUT_NAME_SUFFIX)]
      input_artifacts_output_name[artifact_name] = v
    elif k.endswith(_OUTPUT_PARAM_PATH_SUFFIX):
      param_name = k[:-len(_OUTPUT_PARAM_PATH_SUFFIX)]
      output_params_path[param_name] = v
    elif k.endswith(_OUTPUT_ARTIFACT_PATH_SUFFIX):
      artifact_name = k[:-len(_OUTPUT_ARTIFACT_PATH_SUFFIX)]
      output_artifacts_uri[artifact_name] = v
    elif k not in (_METADATA_FILE_ARG, _FN_NAME_ARG):
      logging.warning(
          'Got unexpected command line argument: %s=%s Ignoring', k, v)

  # Instantiate POD objects.
  input_params = {}
  for param_name in (
      input_params_value.keys() |
      input_params_field_name.keys() | input_params_metadata.keys()):
    input_param = InputParam(
        value=input_params_value.get(param_name),
        metadata_file=input_params_metadata.get(param_name),
        field_name=input_params_field_name.get(param_name))
    input_params[param_name] = input_param

  input_artifacts = {}
  for artifact_name in (
      input_artifacts_uri.keys() |
      input_artifacts_metadata.keys() |
      input_artifacts_output_name.keys()
  ):
    input_artifact = InputArtifact(
        uri=input_artifacts_uri.get(artifact_name),
        metadata_file=input_artifacts_metadata.get(artifact_name),
        output_name=input_artifacts_output_name.get(artifact_name))
    input_artifacts[artifact_name] = input_artifact

  # Import and invoke the user-provided function.
  # Currently the actual user code is built into container as /ml/main.py
  # which is specified in
  # kfp.containers._component_builder.build_python_component.

  # Also, determine a way to inspect the function signature to decide the type
  # of output artifacts.
  fn_name = kwargs[_FN_NAME_ARG]

  fn = entrypoint_utils.import_func_from_source(_FN_SOURCE, fn_name)
  # Get the output artifacts and combine them with the provided URIs.
  output_artifacts = entrypoint_utils.get_output_artifacts(
      fn, output_artifacts_uri)
  invoking_kwargs = {}
  for k, v in output_artifacts.items():
    invoking_kwargs[k] = v

  for k, v in input_params.items():
    invoking_kwargs[k] = v.value
  for k, v in input_artifacts.items():
    invoking_kwargs[k] = v.get_artifact()

  # Execute the user function. fn_res is expected to contain output parameters
  # only. It's either an namedtuple or a single primitive value.
  fn_res = fn(**invoking_kwargs)

  if isinstance(fn_res, (int, float, str)) and len(output_params_path) != 1:
    raise RuntimeError('For primitive output a single output param path is '
                       'expected. Got %s' % output_params_path)

  if isinstance(fn_res, (int, float, str)):
    output_name = list(output_params_path.keys())[0]
    # Write the output to the provided path.
    _gcs_helper.GCSHelper.write_to_gcs_path(
        path=output_params_path[output_name],
        content=str(fn_res))
  else:
    # When multiple outputs, we'll need to match each field to the output paths.
    for idx, output_name in enumerate(fn_res._fields):
      path = output_params_path[output_name]
      _gcs_helper.GCSHelper.write_to_gcs_path(
          path=path,
          content=str(fn_res[idx]))

  # Write output metadata JSON file.
  output_parameters = {}
  if isinstance(fn_res, (int, float, str)):
    output_parameters['output'] = fn_res
  else:
    for idx, output_name in enumerate(fn_res._fields):
      output_parameters[output_name] = fn_res[idx]

  executor_output = entrypoint_utils.get_executor_output(
      output_artifacts=output_artifacts,
      output_params=output_parameters)

  _gcs_helper.GCSHelper.write_to_gcs_path(
      path=kwargs[_METADATA_FILE_ARG],
      content=json_format.MessageToJson(executor_output))


if __name__ == '__main__':
  fire.Fire(main)