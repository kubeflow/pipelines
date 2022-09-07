# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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
"""Module for importing a model evaluation to an existing Vertex model resource."""
import argparse
import os
import re
import sys

from google.api_core import exceptions
from google.cloud.aiplatform import Tensorboard
from google.cloud.aiplatform import TensorboardExperiment
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google.protobuf import json_format

_RESOURCE_TYPE = 'TensorboardExperiment'
_TENSORBOARD_RESOURCE_NAME_REGEX = re.compile(
    r'^projects\/[0-9]+\/locations\/[a-z0-9-]+\/tensorboards\/[0-9]+$')


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path

def main(argv):
  parser = argparse.ArgumentParser(
      prog='Vertex Tensorboard Experiment creator', description='')
  parser.add_argument(
      '--tensorboard_resource_name',
      dest='tensorboard_resource_name',
      type=str,
      default=None)
  parser.add_argument(
      '--tensorboard_experiment_id',
      dest='tensorboard_experiment_id',
      type=str,
      default=None)
  parser.add_argument(
      '--tensorboard_experiment_display_name',
      dest='tensorboard_experiment_display_name',
      type=str,
      default=None)
  parser.add_argument(
      '--tensorboard_experiment_description',
      dest='tensorboard_experiment_description',
      type=str,
      default=None)
  parser.add_argument(
      '--tensorboard_experiment_labels',
      dest='tensorboard_experiment_labels',
      type=dict,
      default=None)
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS)
  parsed_args, _ = parser.parse_known_args(argv)

  tensorboard_resource_name = parsed_args.tensorboard_resource_name
  tensorboard_experiment_id = parsed_args.tensorboard_experiment_id
  tensorboard_experiment_display_name = parsed_args.tensorboard_experiment_display_name
  tensorboard_experiment_description = parsed_args.tensorboard_experiment_description
  tensorboard_experiment_labels = parsed_args.tensorboard_experiment_labels

  if tensorboard_resource_name is None:
    raise RuntimeError('Tensorboard Resouce Name cannot be empty.\n')

  if not _TENSORBOARD_RESOURCE_NAME_REGEX.match(tensorboard_resource_name):
    raise ValueError(
        r'Invalid Tensorboard Resource Name: %s. Tensorboard Resource Name must be like projects/\{project_number\}/locations/\{location}\/tensorboards/\{tensorboard_ID\}'
        % tensorboard_resource_name)
  try:
    tensorboard_instance = Tensorboard(tensorboard_resource_name)
  except exceptions.NotFound as err:
    raise RuntimeError(
        "Tensorboard Insance: {tensorboard_instance} doesn't exist. Please create a Tensorboard or using a existing one."
        .format(tensorboard_instance=tensorboard_resource_name)) from err
  tensorboard_experiment_resource_name = tensorboard_resource_name + '/experiments/' + tensorboard_experiment_id

  try:
    tensorboard_experiment = TensorboardExperiment.create(
        tensorboard_experiment_id=tensorboard_experiment_id,
        tensorboard_name=tensorboard_resource_name,
        display_name=tensorboard_experiment_display_name,
        description=tensorboard_experiment_description,
        labels=tensorboard_experiment_labels)
  except exceptions.AlreadyExists:
    tensorboard_experiment = TensorboardExperiment(
        tensorboard_experiment_resource_name)

  resources = GcpResources()
  tensorboard_experiment_resource = resources.resources.add()
  tensorboard_experiment_resource.resource_type = _RESOURCE_TYPE
  tensorboard_experiment_resource.resource_uri = tensorboard_experiment_resource_name
  with open(parsed_args.gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(resources))


if __name__ == '__main__':
  print(sys.argv)
  main(sys.argv)
