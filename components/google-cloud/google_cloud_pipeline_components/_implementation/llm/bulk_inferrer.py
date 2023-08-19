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
"""KFP Container component that performs bulk inference."""

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
import kfp


@kfp.dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def get_default_bulk_inference_machine_specs(
    large_model_reference: str,
    use_gpu_defaults: bool = False,
    accelerator_type_override: Optional[str] = None,
    accelerator_count_override: Optional[int] = None,
) -> NamedTuple(
    'MachineSpec', accelerator_type=str, accelerator_count=int, machine_type=str
):
  """Gets default machine specs for bulk inference and overrides params if provided.

  Args:
    large_model_reference: Foundational model to use for default specs.
    use_gpu_defaults: Whether to get default gpu specs (otherwise will get TPU
      specs).
    accelerator_type_override: Accelerator type to override the default.
    accelerator_count_override: Accelerator count to override the default.

  Returns:
    MachineSpec, including accelerator_type, accelerator_count, machine_type.

  Raises:
    ValueError: If large_model_reference is invalid or overridden values are
    invalid.
  """
  # pylint: disable=g-import-not-at-top,redefined-outer-name,reimported
  import collections
  # pylint: enable=g-import-not-at-top,redefined-outer-name,reimported

  machine_spec = collections.namedtuple(
      'MachineSpec', ['accelerator_type', 'accelerator_count', 'machine_type']
  )

  # machine types
  cloud_tpu = 'cloud-tpu'
  ultra_gpu_1g = 'a2-ultragpu-1g'
  ultra_gpu_2g = 'a2-ultragpu-2g'
  ultra_gpu_4g = 'a2-ultragpu-4g'
  ultra_gpu_8g = 'a2-ultragpu-8g'
  high_gpu_1g = 'a2-highgpu-1g'
  high_gpu_2g = 'a2-highgpu-2g'
  high_gpu_4g = 'a2-highgpu-4g'
  high_gpu_8g = 'a2-highgpu-8g'
  mega_gpu_16g = 'a2-megagpu-16g'

  # accelerator types
  tpu_v2 = 'TPU_V2'
  tpu_v3 = 'TPU_V3'
  nvidia_a100_40g = 'NVIDIA_TESLA_A100'
  nvidia_a100_80g = 'NVIDIA_A100_80GB'
  tpu_accelerator_types = frozenset([tpu_v2, tpu_v3])
  gpu_accelerator_types = frozenset([nvidia_a100_40g, nvidia_a100_80g])
  valid_accelerator_types = frozenset(
      list(gpu_accelerator_types) + list(tpu_accelerator_types)
  )

  # base models
  palm_tiny = 'PALM_TINY'
  gecko = 'GECKO'
  otter = 'OTTER'
  bison = 'BISON'
  elephant = 'ELEPHANT'
  t5_small = 'T5_SMALL'
  t5_large = 'T5_LARGE'
  t5_xl = 'T5_XL'
  t5_xxl = 'T5_XXL'

  def _get_machine_type(accelerator_type: str, accelerator_count: int) -> str:
    if accelerator_count < 1:
      raise ValueError('accelerator_count must be at least 1.')

    if accelerator_type in tpu_accelerator_types:
      return cloud_tpu

    elif accelerator_type == nvidia_a100_40g:
      if accelerator_count == 1:
        return high_gpu_1g

      elif accelerator_count == 2:
        return high_gpu_2g

      elif accelerator_count <= 4:
        return high_gpu_4g

      elif accelerator_count <= 8:
        return high_gpu_8g

      elif accelerator_count <= 16:
        return mega_gpu_16g

      else:
        raise ValueError(
            f'Too many {accelerator_type} requested. Must be <= 16.'
        )

    elif accelerator_type == nvidia_a100_80g:
      if accelerator_count == 1:
        return ultra_gpu_1g

      elif accelerator_count == 2:
        return ultra_gpu_2g

      elif accelerator_count <= 4:
        return ultra_gpu_4g

      elif accelerator_count <= 8:
        return ultra_gpu_8g

      else:
        raise ValueError(
            f'Too many {accelerator_type} requested. Must be <= 8.'
        )

    else:
      raise ValueError(
          'accelerator_type_override must be one of'
          f' {sorted(valid_accelerator_types)}.'
      )

  accepted_reference_models = frozenset(
      [palm_tiny, gecko, otter, bison, elephant, t5_small, t5_xxl]
  )

  # Default GPU specs are based on study here:
  # https://docs.google.com/spreadsheets/d/1_ZKqfyLQ5vYrOQH5kfdMb_OoNT48r6vNbqv3dKDxDTw/edit?resourcekey=0-3kgDrn4XDdvlJAc8Kils-Q#gid=255356424
  reference_model_to_model_specs_gpu = {
      palm_tiny: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=1,
          machine_type=high_gpu_1g,
      ),
      gecko: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=1,
          machine_type=high_gpu_1g,
      ),
      otter: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=2,
          machine_type=high_gpu_2g,
      ),
      bison: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=8,
          machine_type=high_gpu_8g,
      ),
      elephant: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=8,
          machine_type=high_gpu_8g,
      ),
      t5_small: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=1,
          machine_type=high_gpu_1g,
      ),
      t5_large: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=1,
          machine_type=high_gpu_1g,
      ),
      t5_xl: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=1,
          machine_type=high_gpu_1g,
      ),
      t5_xxl: machine_spec(
          accelerator_type=nvidia_a100_40g,
          accelerator_count=2,
          machine_type=high_gpu_2g,
      ),
  }

  # Get defaults
  if large_model_reference not in accepted_reference_models:
    raise ValueError(
        'large_model_reference must be one of'
        f' {sorted(accepted_reference_models)}.'
    )

  if use_gpu_defaults:
    default_machine_spec = reference_model_to_model_specs_gpu[
        large_model_reference
    ]

  else:
    # This is the only config available for TPUs in our shared reservation pool.
    default_machine_spec = machine_spec(
        accelerator_type=tpu_v3,
        accelerator_count=32,
        machine_type=cloud_tpu,
    )

  # Override default behavior we defer validations of these to the resource
  # provisioner.
  if any([accelerator_type_override, accelerator_count_override]):
    if not all([accelerator_type_override, accelerator_count_override]):
      raise ValueError('Accelerator type and count must both be set.')
    accelerator_type = accelerator_type_override
    accelerator_count = accelerator_count_override
  else:
    accelerator_type = default_machine_spec.accelerator_type
    accelerator_count = default_machine_spec.accelerator_count

  return machine_spec(
      accelerator_type,
      accelerator_count,
      _get_machine_type(accelerator_type, accelerator_count),
  )


@kfp.dsl.container_component
def BulkInferrer(  # pylint: disable=invalid-name
    project: str,
    location: str,
    inputs_sequence_length: int,
    targets_sequence_length: int,
    accelerator_type: str,
    accelerator_count: int,
    machine_type: str,
    image_uri: str,
    dataset_split: str,
    large_model_reference: str,
    input_model: str,
    input_dataset_path: str,
    output_prediction: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    output_prediction_gcs_path: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    gcp_resources: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
) -> kfp.dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Performs bulk inference.

  Args:
    project: Project used to run the job.
    location: Location used to run the job.
    inputs_sequence_length: Maximum encoder/prefix length. Inputs will be padded
      or truncated to this length.
    targets_sequence_length: Maximum decoder steps. Outputs will be at most this
      length.
    accelerator_type: Type of accelerator.
    accelerator_count: Number of accelerators.
    machine_type: Type of machine.
    image: Location of reward model Docker image.
    input_model: Model to use for inference.
    large_model_reference: Predefined model used to create the ``input_model``.
    input_dataset_path: Path to dataset to use for inference.
    dataset_split: Perform inference on this split of the input dataset.

  Returns:
    output_prediction: Where to save the output prediction.
    gcp_resources: GCP resources that can be used to track the custom finetuning
      job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='BulkInferrer',
          accelerator_type=accelerator_type,
          accelerator_count=accelerator_count,
          machine_type=machine_type,
          image_uri=image_uri,
          args=[
              f'--input_model={input_model}',
              f'--input_dataset={input_dataset_path}',
              f'--dataset_split={dataset_split}',
              f'--large_model_reference={large_model_reference}',
              f'--inputs_sequence_length={inputs_sequence_length}',
              f'--targets_sequence_length={targets_sequence_length}',
              f'--output_prediction={output_prediction}',
              f'--output_prediction_gcs_path={output_prediction_gcs_path}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
