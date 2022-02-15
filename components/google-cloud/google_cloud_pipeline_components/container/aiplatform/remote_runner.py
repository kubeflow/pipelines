# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Module for remote execution of AI Platform pipeline component."""

import argparse
import ast
from distutils import util as distutil
import inspect
import json
import os
import re
from typing import Any, Callable, Dict, Tuple, Type, TypeVar

from google.cloud import aiplatform
from google_cloud_pipeline_components.aiplatform import utils
from ..utils import execution_context

INIT_KEY = 'init'
METHOD_KEY = 'method'
AIPLATFORM_API_VERSION = 'v1'

RESOURCE_PREFIX = {
    'bigquery': 'bq://',
    'aiplatform': 'aiplatform://',
    'google_cloud_storage': 'gs://',
    'google_cloud_storage_gcs_fuse': '/gcs/',
}

RESOURCE_NAME_PATTERN = re.compile(
    r'^projects\/(?P<project>[\w-]+)\/locations\/(?P<location>[\w-]+)\/(?P<resource>[\w\-\/]+)\/(?P<id>[\w-]+)$'
)


def split_args(kwargs: Dict[str, Any]) -> Tuple[Dict[str, Any], Dict[str, Any]]:
  """Splits args into constructor and method args.

    Args:
        kwargs: kwargs with parameter names preprended with init or method

    Returns:
        constructor kwargs, method kwargs
    """
  init_args = {}
  method_args = {}

  for key, arg in kwargs.items():
    if key.startswith(INIT_KEY):
      init_args[key.split('.')[-1]] = arg
    elif key.startswith(METHOD_KEY):
      method_args[key.split('.')[-1]] = arg

  return init_args, method_args


def write_to_artifact(executor_input, text):
  """Write output to local artifact and metadata path (uses GCSFuse)."""

  output_artifacts = {}
  for name, artifacts in executor_input.get('outputs', {}).get('artifacts',
                                                               {}).items():
    artifacts_list = artifacts.get('artifacts')
    if artifacts_list:
      output_artifacts[name] = artifacts_list[0]

  executor_output = {}
  if output_artifacts:
    executor_output['artifacts'] = {}
    uri_with_prefix = ''

    # TODO - Support multiple outputs, current implementation.

    # sets all output uri's to text
    for name, artifact in output_artifacts.items():
      metadata = artifact.get('metadata', {})
      # Add URI Prefix
      # "https://[location]-aiplatform.googleapis.com/API_VERSION/": For AI Platform resource names, current version is defined in AIPLATFORM_API_VERSION.
      if RESOURCE_NAME_PATTERN.match(text):
        location = re.findall('locations/([\w\-]+)', text)[0]
        uri_with_prefix = f'https://{location}-aiplatform.googleapis.com/{AIPLATFORM_API_VERSION}/{text}'
        metadata.update({'resourceName': text})

      # "gcs://": For Google Cloud Storage resources.
      elif text.startswith(RESOURCE_PREFIX['google_cloud_storage_gcs_fuse']):
        uri_with_prefix = text.replace(
            RESOURCE_PREFIX['google_cloud_storage_gcs_fuse'],
            RESOURCE_PREFIX.get('google_cloud_storage'))

      # "bq://": For BigQuery resources.
      elif text.startswith(RESOURCE_PREFIX.get('bigquery')):
        uri_with_prefix = text
      else:
        uri_with_prefix = text

      runtime_artifact = {
          'name': artifact.get('name'),
          'uri': uri_with_prefix,
          'metadata': metadata
      }
      artifacts_list = {'artifacts': [runtime_artifact]}

      executor_output['artifacts'][name] = artifacts_list

  os.makedirs(
      os.path.dirname(executor_input['outputs']['outputFile']), exist_ok=True)
  with open(executor_input['outputs']['outputFile'], 'w') as f:
    f.write(json.dumps(executor_output))


def resolve_input_args(value, type_to_resolve):
  """If this is an input from Pipelines, read it directly from gcs."""
  if inspect.isclass(type_to_resolve) and issubclass(
      type_to_resolve, aiplatform.base.VertexAiResourceNoun):
    # Remove '/gcs/' prefix before attempting to remove `aiplatform` prefix
    if value.startswith(RESOURCE_PREFIX['google_cloud_storage_gcs_fuse']):
      value = value[len(RESOURCE_PREFIX['google_cloud_storage_gcs_fuse']):]
    # Remove `aiplatform` prefix from resource name
    if value.startswith(RESOURCE_PREFIX.get('aiplatform')):
      prefix_str = f"{RESOURCE_PREFIX['aiplatform']}{AIPLATFORM_API_VERSION}/"
      value = value[len(prefix_str):]

  # No action needed for Google Cloud Storage prefix.
  # No action needed for BigQuery resource names.
  return value


def resolve_init_args(key, value):
  """Resolves Metadata/InputPath parameters to resource names."""
  if key.endswith('_name'):
    # Remove '/gcs/' prefix before attempting to remove `aiplatform` prefix
    if value.startswith(RESOURCE_PREFIX['google_cloud_storage_gcs_fuse']):
      # not a resource noun, remove the /gcs/ prefix
      value = value[len(RESOURCE_PREFIX['google_cloud_storage_gcs_fuse']):]
    # Remove `aiplatform` prefix from resource name
    if value.startswith(RESOURCE_PREFIX.get('aiplatform')):
      prefix_str = f"{RESOURCE_PREFIX['aiplatform']}{AIPLATFORM_API_VERSION}/"
      value = value[len(prefix_str):]

  # No action needed for Google Cloud Storage prefix.
  # No action needed for BigQuery resource names.
  return value


def make_output(output_object: Any) -> str:
  if utils.is_mb_sdk_resource_noun_type(type(output_object)):
    return output_object.resource_name

  # TODO():handle more default cases
  # right now this is required for export data because proto Repeated
  # this should be expanded to handle multiple different types
  # or possibly export data should return a Dataset
  return json.dumps(list(output_object))


T = TypeVar('T')


def cast(value: str, annotation_type: Type[T]) -> T:
  """Casts a value to the annotation type.

  Includes special handling for bools passed as strings.

  Args:
      value (str): The value represented as a string.
      annotation_type (Type[T]): The type to cast the value to.

  Returns:
      An instance of annotation_type value.
  """
  if annotation_type is bool:
    return bool(distutil.strtobool(value))
  return annotation_type(value)


def prepare_parameters(kwargs: Dict[str, Any],
                       method: Callable,
                       is_init: bool = False):
  """Prepares parameters passed into components before calling SDK.

  1. Determines the annotation type that should used with the parameter
  2. Reads input values if needed
  3. Deserializes those values where appropriate
  4. Or casts to the correct type.

  Args:
      kwargs (Dict[str, Any]): The kwargs that will be passed into method.
        Mutates in place.
      method (Callable): The method the kwargs used to invoke the method.
      is_init (bool): Whether this method is a constructor
  """
  for key, param in inspect.signature(method).parameters.items():
    if key in kwargs:
      value = kwargs[key]
      param_type = utils.resolve_annotation(param.annotation)
      value = resolve_init_args(key, value) if is_init else resolve_input_args(
          value, param_type)
      deserializer = utils.get_deserializer(param_type)
      if deserializer:
        value = deserializer(value)
      else:
        value = cast(value, param_type)

      try:
        # Attempt at converting String to list:
        # Some parameters accept union[str, sequence[str]]
        # For these serialization with json is not possible as
        # component yaml conversion swaps double and single
        # quotes resulting in `JSON.Loads` loading such a list as
        # a String. Using ast.literal_eval to attempt to convert the
        # str back to a python List.
        value = ast.literal_eval(value)
        print(f'Conversion for value succeeded for value: {value}')
      except:
        # The input was actually a String and not a List,
        # no additional transformations are required.
        pass

      kwargs[key] = value


def runner(cls_name, method_name, executor_input, kwargs):
  cls = getattr(aiplatform, cls_name)

  init_args, method_args = split_args(kwargs)

  serialized_args = {INIT_KEY: init_args, METHOD_KEY: method_args}

  prepare_parameters(serialized_args[INIT_KEY], cls.__init__, is_init=True)
  obj = cls(**serialized_args[INIT_KEY]) if serialized_args[INIT_KEY] else cls

  method = getattr(obj, method_name)
  prepare_parameters(serialized_args[METHOD_KEY], method, is_init=False)

  with execution_context.ExecutionContext(
      on_cancel=getattr(obj, 'cancel', None)):
    print(
        f'method:{method} is being called with parameters {serialized_args[METHOD_KEY]}'
    )
    output = method(**serialized_args[METHOD_KEY])
    print('resource_name: %s', obj.resource_name)

    if output:
      write_to_artifact(executor_input, make_output(output))
      return output


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument('--cls_name', type=str)
  parser.add_argument('--method_name', type=str)
  parser.add_argument('--executor_input', type=str, default=None)

  args, unknown_args = parser.parse_known_args()
  kwargs = {}

  executor_input = json.loads(args.executor_input)

  key_value = None
  for arg in unknown_args:

    print(arg)

    # Remove whitespace from arg.
    arg = arg.strip()

    # Assumes all args are passed in the format of '--key value'
    if not key_value:
      key_value = arg[2:]
    else:
      kwargs[key_value] = arg
      key_value = None

  # Update user agent header for metrics reporting
  aiplatform.constants.USER_AGENT_PRODUCT = 'google-cloud-pipeline-components'

  print(runner(args.cls_name, args.method_name, executor_input, kwargs))


if __name__ == '__main__':
  main()
