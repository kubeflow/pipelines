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
"""Functions for creating IR PipelineContainerSpec instance."""

from typing import List

from kfp.pipeline_spec import pipeline_spec_pb2

# TODO: dedupe container spec.


def build_container_spec(
    image: str,
    command: List[str],
    arguments: List[str],
) -> pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec:
  """Builds an IR PipelineContainerSpec instance from structures.ComponentSpec.

  Args:
    image: The path to the container image.
    command: The command to run in the container.
    arguments: The arguments of the command.

  Returns:
    An instance of IR PipelineContainerSpec.
  """
  result = pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec()
  result.image = image
  result.command.extend(command)
  result.args.extend(arguments)

  return result
