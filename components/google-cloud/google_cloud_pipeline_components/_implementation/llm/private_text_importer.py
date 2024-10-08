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
"""KFP Container component that imports Tensorflow Datasets."""
import os
from typing import Optional

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image(default: str = '') -> str:
  return os.environ.get('TEXT_IMPORTER_IMAGE_OVERRIDE') or default


# pytype: disable=unsupported-operands
@dsl.container_component
def private_text_importer(
    project: str,
    location: str,
    input_text: str,
    inputs_field_name: str,
    targets_field_name: str,
    large_model_reference: str,
    imported_data: dsl.Output[dsl.Dataset],  # pylint: disable=unused-argument
    imported_data_path: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    instruction: str = '',
    image_uri: str = utils.get_default_image_uri('refined_cpu', ''),
    machine_type: str = 'e2-highmem-8',
    output_split_name: str = 'all',
    max_num_input_examples: Optional[int] = None,
    encryption_spec_key_name: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Import a text dataset.

  Args:
    project: Project used to run the job.
    location: Location used to run the job.
    input_text: Path to text data. Supports glob patterns.
    inputs_field_name: Name of field that contains input text.
    targets_field_name: Name of field that contains target text.
    large_model_reference: Predefined model used to create the model to be
      trained. This paramerter is used for obtaining model vocabulary because
      this component tokenizes and then caches the tokenized tasks.
    instruction: Optional instruction to prepend to inputs field.
    image_uri: Optional location of the text importer image.
    machine_type: The type of the machine to provision for the custom job.
    output_split_name: The created seqio task has 1 split, its name is specified
      by this argument.
    max_num_input_examples: Maximum number of examples to import.
    encryption_spec_key_name: Customer-managed encryption key. If this is set,
      then all resources created by the CustomJob will be encrypted with the
      provided encryption key. Note that this is not supported for TPU at the
      moment.

  Returns:
    imported_data: Artifact representing the imported data and cached Tasks.
    imported_data_path: Path to cached SeqIO task created from input dataset.
    gcp_resources: Tracker for GCP resources created by this component.
  """
  subdir = (
      f'{dsl.PIPELINE_TASK_NAME_PLACEHOLDER}_{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
  )
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='TextImporter',
          machine_type=machine_type,
          image_uri=_resolve_image(image_uri),
          args=[
              '--app_name=text_importer',
              f'--input_text={input_text}',
              f'--inputs_field_name={inputs_field_name}',
              f'--targets_field_name={targets_field_name}',
              f'--output_split_name={output_split_name}',
              f'--instruction={instruction}',
              f'--large_model_reference={large_model_reference}',
              f'--private_bucket_subdir={subdir}',
              f'--output_dataset_path={dsl.PIPELINE_ROOT_PLACEHOLDER}{subdir}',
              f'--imported_data_path={imported_data_path}',
              f'--max_num_input_examples={max_num_input_examples}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )


# pytype: enable=unsupported-operands
