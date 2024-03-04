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
"""KFP Component for validate_pipeline."""

from typing import Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from kfp import dsl


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def validate_pipeline(
    large_model_reference: str,
    location: str,
    encryption_spec_key_name: str = '',
    machine_type: str = '',
    pipeline_region: str = '{{$.pipeline_google_cloud_location}}',
    eval_dataset: Optional[str] = None,
):
  # fmt: off
  """Validate and preprocess pipeline parameters.

  Args:
      large_model_reference: Name of the base model. Supported values are
      `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`.
      `text-bison@001` and `t5-small` are supported in `us-central1` and
      `europe-west4`.
      location: Region in which all the components except for tuning job should
        run.
      encryption_spec_key_name: If set, CMEK support will be validated.
      machine_type: If 'tpu' is specified, tuning runs in
      europe-west4, else in us-central1.
      pipeline_region: The region the pipeline runs in.
      eval_dataset: Optional Cloud storage path to an evaluation dataset. Note,
      eval dataset can only be provided for third-party models. If provided,
      inference will be performed on this dataset after training. The dataset
      format is jsonl. Each example in the dataset must contain a field
      `input_text` that contains the prompt.
  """
  # fmt: on
  import logging
  import sys

  try:
    models_that_support_bulk_inference = {
        't5-small',
        't5-large',
        't5-xl',
        't5-xxl',
        'llama-2-7b',
        'llama-2-7b-chat',
        'llama-2-13b',
        'llama-2-13b-chat',
    }
    if (
        eval_dataset
        and large_model_reference not in models_that_support_bulk_inference
    ):
      raise ValueError(
          f'eval_dataset not supported for {large_model_reference}. '
          'Please set this value to None when tuning this model. '
          'This model can be evaluated after tuning using Batch or Online '
          'Prediction.'
      )

    if 'gpu' in machine_type:
      accelerator_type = 'GPU'
    elif 'tpu' in machine_type:
      accelerator_type = 'TPU'
    else:
      accelerator_type = None

    supported_pipeline_regions = {
        'europe-west4',
        'us-central1',
    }
    if pipeline_region not in supported_pipeline_regions:
      raise ValueError(
          f'Unsupported pipeline region: {pipeline_region}. Must be one of'
          f' {supported_pipeline_regions}.'
      )

    location = pipeline_region if not location else location

    valid_cmek_config = location == 'us-central1' and accelerator_type == 'GPU'
    if encryption_spec_key_name and not valid_cmek_config:
      raise ValueError(
          'encryption_spec_key_name (CMEK) is only supported for GPU training'
          ' in us-central1. Please either unset encryption_spec_key_name or'
          ' create your pipeline in us-central1 to use GPU instead.'
      )
  except Exception as e:  # pylint: disable=broad-exception-caught
    if isinstance(e, ValueError):
      raise
    logging.exception(str(e))
    sys.exit(13)
