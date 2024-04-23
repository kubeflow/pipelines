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
"""Python function-based components used in KFP pipelies."""
import functools
from typing import List, NamedTuple, Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components._implementation.llm import env
from kfp import dsl


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_machine_spec(
    accelerator_type: str = 'GPU',
    use_test_spec: bool = False,
) -> NamedTuple(
    'MachineSpec',
    machine_type=str,
    tuning_location=str,
    accelerator_type=str,
    accelerator_count=int,
):
  """Returns machine spec to use for a given accelerator_type.

  Args:
    accelerator_type: One of 'TPU' or 'GPU'. If 'TPU' is specified, tuning
      components run in europe-west4. Otherwise tuning components run in
      us-central1 on GPUs. Default is 'GPU'.
    use_test_spec: Whether to use a lower resource machine for testing. If True,
      a machine with the specified `accelerator_type` is provisioned.

  Returns:
    Machine spec.
    tuning_location: Where the machine will run.

  Raises:
    ValueError: If accelerators are requested in an unsupported location.
  """
  outputs = NamedTuple(
      'MachineSpec',
      machine_type=str,
      accelerator_count=int,
      tuning_location=str,
      accelerator_type=str,
  )
  if use_test_spec:
    if accelerator_type == 'TPU':
      return outputs(
          machine_type='cloud-tpu',
          accelerator_type='TPU_V3',
          accelerator_count=32,
          tuning_location='europe-west4',
      )
    elif accelerator_type == 'GPU':
      return outputs(
          machine_type='a2-highgpu-1g',
          accelerator_type='NVIDIA_TESLA_A100',
          accelerator_count=1,
          tuning_location='us-central1',
      )
    elif accelerator_type == 'CPU':
      return outputs(
          machine_type='e2-standard-16',
          accelerator_type='ACCELERATOR_TYPE_UNSPECIFIED',
          accelerator_count=0,
          tuning_location='us-central1',
      )
    else:
      raise ValueError(
          f'Unsupported test accelerator_type {accelerator_type}. Must be one '
          'of TPU, GPU or CPU.'
      )

  if accelerator_type == 'TPU':
    return outputs(
        machine_type='cloud-tpu',
        accelerator_type='TPU_V3',
        accelerator_count=64,
        tuning_location='europe-west4',
    )
  elif accelerator_type == 'GPU':
    return outputs(
        machine_type='a2-ultragpu-8g',
        accelerator_type='NVIDIA_A100_80GB',
        accelerator_count=8,
        tuning_location='us-central1',
    )
  else:
    raise ValueError(
        f'Unsupported accelerator_type {accelerator_type}. Must be one of'
        'TPU or GPU.'
    )


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_refined_image_uri(
    project: str,
    location: str,
    artifact_registry: str,
    tag: str,
    accelerator_type: str = '',
    use_experimental_image: bool = False,
) -> str:
  """Generates image uri based on base image name and accelerator type.

  Args:
    project: Project that contains the artifact registry.
    location: Region that contains the artifact registry.
    artifact_registry: Registry that contains Docker images.
    tag: Image tag.
    accelerator_type: One of the supported accelerator types, e.g. ``'TPU_V3'``.
    use_experimental_image: Whether to use refined experimental image. Default
      is False.

  Returns:
    Docker image uri

  Raises:
    ValueError: if an unsupported accelerator type is provided.
  """
  if not accelerator_type or accelerator_type == 'ACCELERATOR_TYPE_UNSPECIFIED':
    accelerator_postfix = 'cpu'
  elif 'TPU' in accelerator_type:
    accelerator_postfix = 'tpu'
  elif 'A100' in accelerator_type:
    accelerator_postfix = 'gpu'
  else:
    raise ValueError(
        f'Unsupported accelerator type {accelerator_type}. Must a TPU, an A100'
        'variant or empty if using a CPU-only machine.'
    )

  image_name_prefix = 'refined_'
  if use_experimental_image:
    image_name_prefix += 'experimental_'

  return f'{location}-docker.pkg.dev/{project}/{artifact_registry}/{image_name_prefix}{accelerator_postfix}:{tag}'


# Resolves image uri from the environment's private artifact registry.
# By default this resolves an image in the vertex private registry.
resolve_private_refined_image_uri = functools.partial(
    resolve_refined_image_uri,
    project=env.PRIVATE_ARTIFACT_REGISTRY_PROJECT,
    location=env.PRIVATE_ARTIFACT_REGISTRY_LOCATION,
    artifact_registry=env.PRIVATE_ARTIFACT_REGISTRY,
    tag=env.get_private_image_tag(),
)


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_reference_model_metadata(
    large_model_reference: str,
    reference_model_path: Optional[str] = None,
) -> NamedTuple(
    'Outputs',
    large_model_reference=str,
    reference_model_path=str,
    reward_model_reference=str,
    reward_model_path=str,
):
  """Resolves reference model metadata needed by downstream components.

  Args:
    large_model_reference: User-provided reference model name.
    reference_model_path: Optional path to a tuned based model to use in place
      of the default base model. If specified, the model at this path must be a
      tuned version of the base model associated with ``large_model_reference``.

  Returns:
    Base model name (used by downstream components to find gin configs and load
    vocabularies) and the path to the base model checkpoint.

  Raises:
    ValueError: if no metadata exists for the given base model.
  """
  reference_model_metadata = NamedTuple(
      'ReferenceModelMetadata',
      large_model_reference=str,
      reference_model_path=str,
      reward_model_reference=str,
      reward_model_path=str,
      is_supported=bool,
  )

  reference_models = {
      't5-small': reference_model_metadata(
          large_model_reference='T5_SMALL',
          reference_model_path=(
              'gs://vertex-llm-restricted/cloud-llm-restricted/checkpoints/'
              'safe_flan_t5/small/v1/checkpoint_1200000/'
          ),
          reward_model_reference='T5_SMALL',
          reward_model_path='gs://t5-data/pretrained_models/t5x/t5_1_1_small',
          is_supported=True,
      ),
      't5-large': reference_model_metadata(
          large_model_reference='T5_LARGE',
          reference_model_path=(
              'gs://vertex-llm-restricted/cloud-llm-restricted/checkpoints/'
              'safe_flan_t5/large/v1/checkpoint_1200000/'
          ),
          reward_model_reference='T5_LARGE',
          reward_model_path='gs://t5-data/pretrained_models/t5x/t5_1_1_large',
          is_supported=True,
      ),
      't5-xl': reference_model_metadata(
          large_model_reference='T5_XL',
          reference_model_path=(
              'gs://vertex-llm-restricted/cloud-llm-restricted/checkpoints/'
              'safe_flan_t5/xl/v1/checkpoint_1200000/'
          ),
          reward_model_reference='T5_XL',
          reward_model_path='gs://t5-data/pretrained_models/t5x/t5_1_1_xl',
          is_supported=True,
      ),
      't5-xxl': reference_model_metadata(
          large_model_reference='T5_XXL',
          reference_model_path=(
              'gs://vertex-llm-restricted/cloud-llm-restricted/checkpoints/'
              'safe_flan_t5/xxl/v1/checkpoint_1190000/'
          ),
          reward_model_reference='T5_XXL',
          reward_model_path='gs://t5-data/pretrained_models/t5x/t5_1_1_xxl',
          is_supported=True,
      ),
      'palm-tiny': reference_model_metadata(
          large_model_reference='PALM_TINY',
          reference_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_palm_tiny/',
          reward_model_reference='PALM_TINY',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_palm_tiny/',
          is_supported=False,
      ),
      'gecko': reference_model_metadata(
          large_model_reference='GECKO',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_gecko/'
          ),
          reward_model_reference='GECKO',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_gecko_pretrain/',
          is_supported=False,
      ),
      'otter': reference_model_metadata(
          large_model_reference='OTTER',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_otter/'
          ),
          reward_model_reference='OTTER',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_otter_pretrain/',
          is_supported=False,
      ),
      'bison': reference_model_metadata(
          large_model_reference='BISON',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          reward_model_reference='BISON',
          reward_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          is_supported=False,  # Deprecated: Use text-bision@001 instead.
      ),
      'text-bison@001': reference_model_metadata(
          large_model_reference='BISON',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          reward_model_reference='BISON',
          reward_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          is_supported=True,
      ),
      'text-bison@002': reference_model_metadata(
          large_model_reference='BISON_002',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison_002/'
          ),
          reward_model_reference='BISON_002',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison_002/',
          is_supported=True,
      ),
      'chat-bison@001': reference_model_metadata(
          large_model_reference='BISON',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          reward_model_reference='BISON',
          reward_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_bison/'
          ),
          is_supported=True,
      ),
      'elephant': reference_model_metadata(
          large_model_reference='ELEPHANT',
          reference_model_path=(
              'gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_elephant/'
          ),
          reward_model_reference='OTTER',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/palm/t5x_otter_pretrain/',
          is_supported=False,
      ),
      'llama-2-7b': reference_model_metadata(
          large_model_reference='LLAMA_2_7B',
          reference_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b/',
          reward_model_reference='LLAMA_2_7B',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b/',
          is_supported=True,
      ),
      'llama-2-13b': reference_model_metadata(
          large_model_reference='LLAMA_2_13B',
          reference_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_13b/',
          reward_model_reference='LLAMA_2_7B',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b/',
          is_supported=True,
      ),
      'llama-2-7b-chat': reference_model_metadata(
          large_model_reference='LLAMA_2_7B_CHAT',
          reference_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b_chat/',
          reward_model_reference='LLAMA_2_7B',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b/',
          is_supported=True,
      ),
      'llama-2-13b-chat': reference_model_metadata(
          large_model_reference='LLAMA_2_13B_CHAT',
          reference_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_13b_chat/',
          reward_model_reference='LLAMA_2_7B',
          reward_model_path='gs://vertex-rlhf-restricted/pretrained_models/llama/t5x_llama_2_7b/',
          is_supported=True,
      ),
  }

  reference_model_key = large_model_reference.lower().replace('_', '-')
  if reference_model_key not in reference_models:
    supported_models = [
        k for k, v in reference_models.items() if v.is_supported
    ]
    raise ValueError(
        f'Unknown reference model {large_model_reference}.'
        ' large_model_reference must be one of'
        f' {sorted(supported_models)}.'
    )

  reference_model = reference_models[reference_model_key]

  outputs = NamedTuple(
      'Outputs',
      large_model_reference=str,
      reference_model_path=str,
      reward_model_reference=str,
      reward_model_path=str,
  )

  return outputs(
      large_model_reference=reference_model.large_model_reference,
      reference_model_path=(
          reference_model_path or reference_model.reference_model_path
      ),
      reward_model_reference=reference_model.reward_model_reference,
      reward_model_path=reference_model.reward_model_path,
  )


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def convert_to_delimited_string(items: List[str], delimiter: str = ',') -> str:
  """Converts a list of strings to single string delimited by the specified character."""
  return delimiter.join(items)


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_regional_endpoint(upload_location: str) -> str:
  """Gets the regional endpoint used to upload a model to the registry.

  Args:
    upload_location: Region where the model will be uploaded.

  Returns:
    Regional endpoint.
  """
  return f'https://{upload_location}-aiplatform.googleapis.com/ui'


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_model_display_name(
    large_model_reference: str,
    model_display_name: Optional[str] = None,
) -> str:
  """Gets the model display name shown in the registry and used for endpoints.

  Args:
    large_model_reference: Base model tuned by the pipeline.
    model_display_name: User-provided display name. If not provided, a default
      display name will be created.

  Returns:
    Either the user-provided name or a default display name with the form
    ``{large_model_reference}-{timestamp}``
  """
  # pylint: disable=g-import-not-at-top
  import datetime
  # pylint: enable=g-import-not-at-top
  now = datetime.datetime.now().strftime('%Y-%m-%d-%H-%M-%S')
  return model_display_name or f'{large_model_reference.lower()}-{now}'


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_deploy_model(
    deploy_model: bool, large_model_reference: str
) -> bool:
  """Resolves runtime parameter that determines whether the tuned model should be deployed."""
  supported_models = {'BISON', 'BISON_002'}
  if deploy_model and large_model_reference in supported_models:
    return True
  return False


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def value_exists(value: Optional[str] = None) -> bool:
  """Returns whether a runtime parameter was provided.

  Args:
    value: That might have been provided.

  Returns:
    Whether the string is not None and non-empty.
  """
  if not value:
    return False
  return True


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_upload_model(large_model_reference: str) -> bool:
  """Returns whether the model should be uploaded."""
  supported_models = {'BISON', 'BISON_002'}
  if large_model_reference in supported_models:
    return True
  return False


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_instruction(
    large_model_reference: str, instruction: Optional[str] = None
) -> str:
  """Resolves the instruction to use for a given reference model.

  Args:
    large_model_reference: Base model tuned by the pipeline.
    instruction: Instruction provided at runtime.

  Returns:
    Instruction to use during tokenization based on model type. Returns an empty
      string for chat models because the instruction is prepended as the default
      context. Otherwise the original instruction is returned.
  """
  instruction = instruction or ''
  return instruction if 'chat' not in large_model_reference.lower() else ''


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def resolve_num_microbatches(large_model_reference: str) -> int:
  """Resolves the number of microbatches to use during training.

  Args:
    large_model_reference: Base model tuned by the pipeline.

  Returns:
    Number of microbatches to break the total batch size into during training.
  """
  if 'llama' in large_model_reference.lower():
    return 2
  return 0
