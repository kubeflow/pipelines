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
"""Utility functions used to create custom Kubeflow components."""
import os
from typing import Any, Dict, List, Optional

from google_cloud_pipeline_components._implementation.llm import env
import kfp


def build_payload(
    *,
    display_name: str,
    machine_type: str,
    image_uri: str,
    args: List[str],
    accelerator_type: str = '',
    accelerator_count: int = 0,
    encryption_spec_key_name: str = '',
    labels: Optional[Dict[str, str]] = None,
    scheduling: Optional[Dict[str, Any]] = None,
    base_output_directory: Optional[str] = None,
    tensorboard: Optional[str] = None,
) -> Dict[str, Any]:
  """Generates payload for a custom training job.

  Args:
    display_name: Component display name. Can contain up to 128 UTF-8
      characters.
    machine_type: The type of the machine to provision for the custom job. Must
      be a valid GCE instance type and compatible with the accelerator type.
    image_uri: Docker image URI to use for the custom job.
    args: Arguments to pass to the Docker image.
    accelerator_type: Type of accelerator. By default no accelerator is
      requested.
    accelerator_count: Number of accelerators. By default no accelerators are
      requested.
    encryption_spec_key_name: Customer-managed encryption key. If this is set,
      then all resources created by the CustomJob will be encrypted with the
      provided encryption key. Note that this is not supported for TPU at the
      moment.
    labels: The labels with user-defined metadata to organize CustomJobs.
    scheduling: Scheduling options for a CustomJob.
    base_output_directory: Cloud Storage location to store the output of this
      CustomJob
    tensorboard: The name of a Vertex AI TensorBoard resource to which this
      CustomJob will upload TensorBoard logs. Format:
      ``projects/{project}/locations/{location}/tensorboards/{tensorboard}``

  Returns:
    Custom job payload.

  Raises:
    ValueError: if one of ``accelerator_count`` or ``accelerator_type`` is
      specified, but the corresponding field is not valid.
  """
  payload = {
      'display_name': display_name,
      'job_spec': {
          'worker_pool_specs': [{
              'replica_count': '1',
              'machine_spec': {'machine_type': str(machine_type)},
              'container_spec': {'image_uri': str(image_uri), 'args': args},
          }]
      },
  }

  if accelerator_type and accelerator_count:
    payload['job_spec']['worker_pool_specs'][0]['machine_spec'][
        'accelerator_type'
    ] = str(accelerator_type)
    payload['job_spec']['worker_pool_specs'][0]['machine_spec'][
        'accelerator_count'
    ] = accelerator_count
  elif accelerator_type and accelerator_count < 1:
    raise ValueError(
        'Accelerator count must be at least 1 if accelerator type '
        f'is specified. Received accelerator_count == {accelerator_count}'
    )
  elif accelerator_count and not accelerator_type:
    raise ValueError(
        'Accelerator type must be specified if accelerator count is not 0.'
        f'Received accelerator_type == {accelerator_type}.'
    )

  if encryption_spec_key_name:
    payload['encryption_spec'] = {'kms_key_name': encryption_spec_key_name}

  if labels:
    payload['labels'] = labels

  if scheduling:
    payload['job_spec']['scheduling'] = scheduling

  if base_output_directory:
    payload['job_spec']['base_output_directory'] = {
        'output_uri_prefix': base_output_directory
    }

  if tensorboard:
    payload['job_spec']['tensorboard'] = tensorboard

  return payload


def get_temp_location() -> str:
  """Gets a task-specific location to store temporary files."""
  return os.path.join(
      kfp.dsl.PIPELINE_ROOT_PLACEHOLDER,
      kfp.dsl.PIPELINE_JOB_ID_PLACEHOLDER,
      kfp.dsl.PIPELINE_TASK_ID_PLACEHOLDER,
      'temp',
  )


def get_default_image_uri(
    image_name: str,
    image_name_prefix: Optional[str] = None,
) -> str:
  """Gets the default image URI for a given image.

  The URI is resolved using environment variables that define the artifact
  registry, image name modifications and tag. This method only works for images
  that are not selected dynamically based on accelerator type. This is typically
  true for CPU-only images.

  Args:
    image_name: Name of the image to resolve.
    image_name_prefix: prefix to add to the image name when constructing the
      URI. If `None`, `env.PRIVATE_IMAGE_NAME_PREFIX'` is used.

  Returns:
    URI of the image.
  """
  if image_name.find('autosxs') != -1:
    image_tag = env.get_autosxs_image_tag()
  else:
    image_tag = env.get_private_image_tag()

  if image_name_prefix is None:
    image_name_prefix = env.PRIVATE_IMAGE_NAME_PREFIX

  return '/'.join([
      f'{env.PRIVATE_ARTIFACT_REGISTRY_LOCATION}-docker.pkg.dev',
      env.PRIVATE_ARTIFACT_REGISTRY_PROJECT,
      env.PRIVATE_ARTIFACT_REGISTRY,
      f'{image_name_prefix}{image_name}:{image_tag}',
  ])
