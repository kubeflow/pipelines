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
"""Component that preprocesses inputs for Reinforcement Learning from Human Feedback (RLHF)."""

import os

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


@dsl.container_component
def rlhf_preprocessor(
    large_model_reference: str,
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    has_tensorboard_id: dsl.OutputPath(bool),  # pytype: disable=invalid-annotation
    has_inference_dataset: dsl.OutputPath(bool),  # pytype: disable=invalid-annotation
    metadata_large_model_reference: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reference_model_path: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reward_model_reference: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reward_model_path: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    evaluation_dataset: str = '',
    tensorboard_resource_id: str = '',
    input_reference_model_path: str = '',
    image_uri: str = utils.get_default_image_uri('refined_cpu', ''),
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  # fmt: off
  """Preprocess RLHF pipeline inputs.

  Args:
    large_model_reference: The model for fine tuning.
    evaluation_dataset: Path to evaluation data.
    tensorboard_resource_id: TensorBoard resource id.
    metadata_large_model_reference: The base model for fine tuning. The name should be in capitalized snake case format.
    metadata_reference_model_path: The model checkpoint path for the reinforcer model
    metadata_reward_model_reference:  The base model for training reward model. The name should be in capitalized snake case format.
    metadata_reward_model_path: The model checkpoint path for the reward model.

  Returns:
    gcp_resources: GCP resources that can be used to track the custom job.
    has_tensorboard_id: Whether a tensorboard id is provided.
    has_inference_dataset: Whether inference data are provided.
  """
  # fmt: on
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='rlhf_preprocessor',
          machine_type='n1-standard-4',
          image_uri=image_uri,
          args=[
              '--app_name=rlhf_preprocessor',
              f'--evaluation_dataset={evaluation_dataset}',
              f'--tensorboard_resource_id={tensorboard_resource_id}',
              f'--large_model_reference={large_model_reference}',
              f'--input_reference_model_path={input_reference_model_path}',
              f'--has_tensorboard_id_path={has_tensorboard_id}',
              f'--has_inference_dataset_path={has_inference_dataset}',
              f'--metadata_large_model_reference_path={metadata_large_model_reference}',
              f'--metadata_reference_model_path_path={metadata_reference_model_path}',
              f'--metadata_reward_model_reference_path={metadata_reward_model_reference}',
              f'--metadata_reward_model_path_path={metadata_reward_model_path}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
