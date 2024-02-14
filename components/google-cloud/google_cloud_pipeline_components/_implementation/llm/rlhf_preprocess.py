# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Component for running LLM Batch Prediction jobs side-by-side."""

import os

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  """Determines the image URI to create a container from."""
  return os.environ.get(
      'RLHF_PREPROCESS_IMAGE_OVERRIDE'
  ) or utils.get_default_image_uri('rlhf_preprocess')


# pylint: disable=unused-argument,dangerous-default-value
@dsl.container_component
def rlhf_preprocess(
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    has_tensorboard_id: dsl.OutputPath(bool),  # pytype: disable=invalid-annotation
    has_inference_dataset: dsl.OutputPath(bool),  # pytype: disable=invalid-annotation
    evaluation_dataset: str,
    tensorboard_resource_id: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='rlhf_preprocess',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'rlhf_preprocess',
              f'--evaluation_dataset={evaluation_dataset}',
              f'--tensorboard_resource_id={tensorboard_resource_id}',
              f'--project={_placeholders.PROJECT_ID_PLACEHOLDER}',
              f'--location={_placeholders.LOCATION_PLACEHOLDER}',
              f'--has_tensorboard_id_path={has_tensorboard_id}',
              f'--has_inference_dataset_path={has_inference_dataset}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
