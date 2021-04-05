# Copyright 2021 Google LLC. All Rights Reserved.
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
"""Module for remote execution of AI Platform component."""

import argparse
import inspect
import os
from google.cloud import aiplatform
from google.cloud import storage

INIT_KEY = 'init'
METHOD_KEY = 'method'


# TODO() Add type-hinting the functions
# TODO() Add explanation / exmaples and validation for kwargs
def split_args(kwargs):
  """Split args to match MB SDK input format for init, method and config."""
  init_args = {}
  method_args = {}
  config_args = {}

  for key, arg in kwargs.items():
    if key.startswith(INIT_KEY):
      init_args[key.split('.')[-1]] = arg
    elif key.startswith(METHOD_KEY):
      method_args[key.split('.')[-1]] = arg
    else:
      config_args[key.split('.')[-1]] = arg

  return init_args, method_args, config_args

# TODO() Add docstring and typ annotation. 
def write_to_gcs(project, gcs_uri, text):
  """Helper method to write resouce names to gcs for downstream components."""

  gcs_uri = gcs_uri[len('gs://'):]
  gcs_bucket, gcs_blob = gcs_uri.split('/', 1)

  with open('resource_name.txt', 'w') as f:
    f.write(text)

  client = storage.Client(project=project)
  bucket = client.get_bucket(gcs_bucket)
  blob = bucket.blob(os.path.join(gcs_blob, 'resource_name.txt'))
  blob.upload_from_filename('resource_name.txt')


def read_from_gcs(project, gcs_uri):
  gcs_uri = gcs_uri[len('gs://'):]
  gcs_bucket, gcs_blob = gcs_uri.split('/', 1)
  client = storage.Client(project=project)
  bucket = client.get_bucket(gcs_bucket)
  blob = bucket.get_blob(os.path.join(gcs_blob, 'resource_name.txt'))
  resource_name = blob.download_as_string().decode('utf-8')
  return resource_name


def resolve_project(serialized_args):
  return serialized_args[INIT_KEY].get(
      'project', serialized_args[METHOD_KEY].get('project'))


def resolve_input_args(value, inpu_type, project):
  """If this is an input from Pipelines, read it directly from gcs."""
  if issubclass(inpu_type, aiplatform.base.AiPlatformResourceNoun):
    if value.startswith('gs://'):  # not a resource noun:
      value = read_from_gcs(project, value)
  return value


def resolve_init_args(key, value, project):
  if key.endswith('_name'):
    if value.startswith('gs://'):  # not a resource noun
      value = read_from_gcs(project, value)
  return value


def runner(cls_name, method_name, resource_name_output_uri, kwargs):
  """Main method for initializing and executing MBD SDK component."""
  cls = getattr(aiplatform, cls_name)

  init_args, method_args, _ = split_args(kwargs)

  serialized_args = {INIT_KEY: init_args, METHOD_KEY: method_args}

  project = resolve_project(serialized_args)

  for key, param in inspect.signature(cls.__init__).parameters.items():
    if key in serialized_args[INIT_KEY]:
      serialized_args[INIT_KEY][key] = resolve_init_args(
          key, serialized_args[INIT_KEY][key], project)

  print(serialized_args[INIT_KEY])
  obj = cls(**serialized_args[INIT_KEY]) if serialized_args[INIT_KEY] else cls

  method = getattr(obj, method_name)

  for key, param in inspect.signature(method).parameters.items():
    if key in serialized_args[METHOD_KEY]:
      type_args = getattr(param.annotation, '__args__', param.annotation)
      cast = type_args[0] if isinstance(type_args, tuple) else type_args
      serialized_args[METHOD_KEY][key] = resolve_input_args(
          serialized_args[METHOD_KEY][key], cast, project)
      serialized_args[METHOD_KEY][key] = cast(serialized_args[METHOD_KEY][key])

  print(serialized_args[METHOD_KEY])
  output = method(**serialized_args[METHOD_KEY])

  if output:
    write_to_gcs(project, resource_name_output_uri, output.resource_name)
    return output


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument('--cls_name', type=str)
  parser.add_argument('--method_name', type=str)
  parser.add_argument('--resource_name_output_uri', type=str)

  args, unknown_args = parser.parse_known_args()
  kwargs = {}

  key_value = None
  for arg in unknown_args:
    print(arg)
    if '=' in arg:
      key, value = arg[2:].split('=')
      kwargs[key] = value
    else:
      if not key_value:
        key_value = arg[2:]
      else:
        kwargs[key_value] = arg
        key_value = None

  print(
      runner(args.cls_name, args.method_name, args.resource_name_output_uri,
             kwargs))


if __name__ == '__main__':
  main()
