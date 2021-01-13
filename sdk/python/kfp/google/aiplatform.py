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
"""Connector components of Google AI Platform (Unified) services."""
from typing import Any, Dict, List, Optional, Type

from kfp import dsl
from kfp.dsl import artifact


def custom_job(
    input_artifacts: Optional[Dict[str, Any]] = None,
    input_parameters: Optional[Dict[str, Any]] = None,
    output_artifacts: Optional[Dict[str, Type[artifact.Artifact]]] = None,
    output_parameters: Optional[Dict[str, Any]] = None,
    # Custom container training specs.
    image_uri: Optional[str] = None,
    commands: Optional[List[str]] = None,
    # Custom Python training spec.
    executor_image_uri: Optional[str] = None,
    package_uris: Optional[List[str]] = None,
    python_module: Optional[str] = None,
    # Command line args of the user program.
    args: Optional[List[Any]] = None,
    machine_type: Optional[str] = None,
    # Full-fledged custom job API spec. For details please see:
    # https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/CustomJobSpec
    additional_job_spec: Optional[Dict[str, Any]] = None
) -> dsl.ContainerOp:
  """DSL representation of a AI Platform (Unified) custom training job.

  For detailed doc of the service, please refer to
  https://cloud.google.com/ai-platform-unified/docs/training/create-custom-job

  Args:
    input_artifacts: The input artifact specification. Should be a mapping from
      input name to output from upstream tasks.
    input_parameters: The input parameter specification. Should be a mapping
      from input name to one of the following three:
      - output from upstream tasks, or
      - pipeline parameter, or
      - constant value
    output_artifacts: The output artifact declaration. Should be a mapping from
      output name to a type subclassing artifact.Artifact.
    output_parameters: The output parameter declaration. Should be a mapping
      from output name to one of 1) str, 2) float, or 3) int.
    image_uri: The URI of the container image containing the user training
      program. Applicable for custom container training.
    commands: The container command/entrypoint. Applicable for custom container
      training.
    executor_image_uri: The URI of the container image containing the
      dependencies of user training program. Applicable for custom Python
      training.
    package_uris: The Python packages that are expected to be running on the
      executor container. Applicable for custom Python training.
    python_module: The entrypoint of user training program. Applicable for
      custom Python training.
    args: The command line arguments of user training program. This is expected
      to be a list of either 1) constant string, or 2) KFP DSL placeholders, to
      connect the user program with the declared component I/O.
    machine_type: The machine type used to run the training program. The value
      of this field will be propagated to all worker pools if not specified
      otherwise in additional_job_spec.
    additional_job_spec: Full-fledged custom job API spec. The value specified
      in this field will override the defaults provided through other function
      parameters.

      For details please see:
      https://cloud.google.com/ai-platform-unified/docs/reference/rest/v1beta1/CustomJobSpec

  Returns:
    A KFP ContainerOp object represents the launcher container job, from which
    the user training program will be submitted to AI Platform (Unified) Custom
    Job service.

  Raises:
    TBD
  """
  pass
