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
"""Component that preprocesses inputs for infer pipeline."""

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


@dsl.container_component
def infer_preprocessor(
    large_model_reference: str,
    accelerator_type: str,
    use_test_spec: bool,
    project: str,
    location: str,
    artifact_registry: str,
    tag: str,
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_large_model_reference: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reference_model_path: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reward_model_reference: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_reward_model_path: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_machine_type: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_tuning_location: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_accelerator_type: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_accelerator_count: dsl.OutputPath(int),  # pytype: disable=invalid-annotation
    metadata_instruction: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    metadata_refined_image_uri: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    use_experimental_image: bool = False,
    input_reference_model_path: str = '',
    instruction: str = '',
    image_uri: str = utils.get_default_image_uri('refined_cpu', ''),
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  # fmt: off
  """Preprocess infer pipeline inputs.

  Args:
    large_model_reference: The model for fine tuning.
    accelerator_type: Specific accelerator type for the job.
    use_test_spec: Whether to use a lower resource machine for testing.
    project: Project that contains the artifact registry.
    location: Region that contains the artifact registry.
    artifact_registry: Registry that contains Docker images.
    tag: Image tag.
    use_experimental_image:  Whether to use refined experimental image.
    input_reference_model_path: The model checkpoint path for the reference model
    instruction: The instruction to let the model know what task it needs to perform.
    image_uri: Docker image URI to use for the custom job.

  Returns:
    gcp_resources: GCP resources that can be used to track the custom job.
    metadata_large_model_reference: The base model for fine tuning. The name should be in capitalized snake case format.
    metadata_reference_model_path: The model checkpoint path for the reinforcer model
    metadata_reward_model_reference:  The base model for training reward model. The name should be in capitalized snake case format.
    metadata_reward_model_path: The model checkpoint path for the reward model.
    metadata_machine_type: The type of the machine to provision for the custom job.
    metadata_tuning_location: The GCP region to run the custom job.
    metadata_accelerator_type: Specific accelerator type for the custom job.
    metadata_accelerator_count: The number of accelerator.
    metadata_instruction: The instruction to let the model know what task it needs to perform.
    metadata_refined_image_uri: Docker image URI to use for the custom job.
  """
  # fmt: on
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='infer_preprocessor',
          machine_type='n1-standard-4',
          image_uri=image_uri,
          args=[
              '--app_name=infer_preprocessor',
              f'--large_model_reference={large_model_reference}',
              f'--input_reference_model_path={input_reference_model_path}',
              f'--accelerator_type={accelerator_type}',
              f'--use_test_spec={use_test_spec}',
              f'--project={project}',
              f'--location={location}',
              f'--artifact_registry={artifact_registry}',
              f'--tag={tag}',
              f'--use_experimental_image={use_experimental_image}',
              f'--instruction={instruction}',
              f'--metadata_large_model_reference_path={metadata_large_model_reference}',
              f'--metadata_reference_model_path_path={metadata_reference_model_path}',
              f'--metadata_reward_model_reference_path={metadata_reward_model_reference}',
              f'--metadata_reward_model_path_path={metadata_reward_model_path}',
              f'--metadata_machine_type_path={metadata_machine_type}',
              f'--metadata_tuning_location_path={metadata_tuning_location}',
              f'--metadata_accelerator_type_path={metadata_accelerator_type}',
              f'--metadata_accelerator_count_path={metadata_accelerator_count}',
              f'--metadata_instruction_path={metadata_instruction}',
              f'--metadata_refined_image_uri_path={metadata_refined_image_uri}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
