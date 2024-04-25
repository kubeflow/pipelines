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

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from kfp import dsl


@dsl.component(base_image=_image.GCPC_IMAGE_TAG, install_kfp_package=False)
def validate_pipeline(
    location: str,
    encryption_spec_key_name: str = '',
    accelerator_type: str = '',
    eval_dataset: Optional[str] = None,
) -> NamedTuple('PreprocessedInputs', reward_model_eval_dataset=str):
  # fmt: off
  """Validates and preprocesses RLHF pipeline parameters.

  Args:
    location: Location used to run non-tuning components, i.e. components
      that do not require accelerators. If not specified the location used
      to run the pipeline will be used.
    encryption_spec_key_name: If set, CMEK support will be validated.
    accelerator_type: One of 'TPU' or 'GPU'. If 'TPU' is specified, tuning
      components run in europe-west4. Otherwise tuning components run in
      us-central1 on GPUs. Default is 'GPU'.
    eval_dataset: Optional Cloud storage path to an evaluation dataset. The
      format should match that of the preference dataset.
  """
  # fmt: on
  # pylint: disable=g-import-not-at-top,import-outside-toplevel
  import json
  import logging
  import re
  import sys
  import glob
  # pylint: enable=g-import-not-at-top,import-outside-toplevel
  outputs = NamedTuple(
      'PreprocessedInputs',
      reward_model_eval_dataset=str,
  )

  try:
    # [ Set eval_dataset
    eval_dataset = eval_dataset or ''
    gcs_eval_dataset_uri = re.sub('^gs://', '/gcs/', eval_dataset)
    files_in_folder = glob.glob(gcs_eval_dataset_uri)
    if not files_in_folder:
      eval_dataset = ''
    else:
      first_file = files_in_folder[0]
      required_fields = ('candidate_0', 'candidate_1', 'choice')
      oneof_fields = {'input_text', 'messages'}
      max_lines_to_check = 100
      with open(first_file, 'r') as inputs:
        for i, line in enumerate(inputs):
          json_data = json.loads(line)
          is_valid_preference_data = all(
              field in json_data for field in required_fields
          ) and any(oneof_field in json_data for oneof_field in oneof_fields)
          if not is_valid_preference_data:
            eval_dataset = ''
          if not eval_dataset or i >= max_lines_to_check:
            break
    # ]
    # [ Check CMEK
    supported_pipeline_regions = {
        'asia-northeast1',
        'asia-northeast3',
        'asia-southeast1',
        'europe-west1',
        'europe-west2',
        'europe-west3',
        'europe-west4',
        'europe-west9',
        'northamerica-northeast1',
        'us-central1',
        'us-east4',
        'us-west1',
        'us-west4',
    }
    if location not in supported_pipeline_regions:
      raise ValueError(
          f'Unsupported pipeline region: {location}. Must be one of'
          f' {supported_pipeline_regions}.'
      )

    valid_cmek_accelerator_types = {
        'GPU',
        'CPU',  # Only used for testing.
    }
    valid_cmek_config = (
        location == 'us-central1'
        and accelerator_type in valid_cmek_accelerator_types
    )
    if encryption_spec_key_name and not valid_cmek_config:
      raise ValueError(
          'encryption_spec_key_name (CMEK) is only supported for GPU training'
          ' in us-central1. Please either unset encryption_spec_key_name or'
          ' create your pipeline in us-central1 to use GPU instead.'
      )
    # CMEK ]

    return outputs(reward_model_eval_dataset=eval_dataset)

  except Exception as e:  # pylint: disable=broad-exception-caught
    if isinstance(e, ValueError):
      raise
    logging.exception(str(e))
    sys.exit(13)
