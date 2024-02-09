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
"""A collection of constants shared across components and pipelines."""

import os

from google_cloud_pipeline_components._implementation.llm.generated import refined_image_versions


def get_private_image_tag() -> str:
  return os.getenv('PRIVATE_IMAGE_TAG') or refined_image_versions.IMAGE_TAG


def get_autosxs_image_tag() -> str:
  return os.getenv('PRIVATE_IMAGE_TAG') or refined_image_versions.IMAGE_TAG


def get_use_test_machine_spec() -> bool:
  str_value = os.getenv('USE_TEST_MACHINE_SPEC', 'False')
  return str_value.lower() in {'true', '1'}


# Variables associated with private images:
CLOUD_ML_REGION = os.getenv('CLOUD_ML_REGION', 'europe-west4')
PRIVATE_ARTIFACT_REGISTRY_PROJECT: str = (
    os.getenv(
        'PRIVATE_ARTIFACT_REGISTRY_PROJECT',
    )
    or 'vertex-ai-restricted'
)
PRIVATE_ARTIFACT_REGISTRY_LOCATION: str = (
    os.getenv(
        'PRIVATE_ARTIFACT_REGISTRY_LOCATION',
    )
    or 'us'
)
PRIVATE_ARTIFACT_REGISTRY: str = (
    os.getenv('PRIVATE_ARTIFACT_REGISTRY') or 'rlhf'
)
PRIVATE_IMAGE_NAME_PREFIX: str = (
    os.getenv('PRIVATE_IMAGE_NAME_PREFIX') or 'rlhf_'
)
PRIVATE_IMAGE_TAG: str = get_private_image_tag()
AUTOSXS_IMAGE_TAG: str = get_autosxs_image_tag()

# Dataset variables:
TRAIN_SPLIT: str = 'train'
