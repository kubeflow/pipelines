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

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
import kfp


@kfp.dsl.container_component
def private_text_comparison_importer(
    project: str,
    location: str,
    input_text: str,
    inputs_field_name: str,
    comma_separated_candidates_field_names: str,
    choice_field_name: str,
    split: str,
    large_model_reference: str,
    output_dataset_path: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    gcp_resources: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    image_uri: str = utils.get_default_image_uri('refined_cpu', ''),
    machine_type: str = 'e2-highmem-8',
    instruction: str = '',
    encryption_spec_key_name: str = '',
) -> kfp.dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Import a text dataset.

  Args:
    project: Project used to run the job.
    location: Location used to run the job.
    input_text: Path to text data. Supports glob patterns.
    inputs_field_name: Name of field that contains input text.
    comma_separated_candidates_field_names: Comma separated list of fields that
      contain candidate text, e.g. ``'field_1,field_2,field_3'``.
    choice_field_name: Name of field that specifies the index of the best
      candidate.
    split: The created seqio task has 1 split, its name is specified by this
      argument.
    large_model_reference: Predefined model used to create the model to be
      trained. This paramerter is used for obtaining model vocabulary because
      this component tokenizes and then caches the tokenized tasks.
    machine_type: The type of the machine to provision for the custom job.
    instruction: Optional instruction to prepend to inputs field.
    image_uri: Optional location of the text comparison importer image.
    dataflow_worker_image_uri: Location of the Dataflow worker image.
    encryption_spec_key_name: Customer-managed encryption key. If this is set,
      then all resources created by the CustomJob will be encrypted with the
      provided encryption key. Note that this is not supported for TPU at the
      moment.

  Returns:
    output_dataset_path: Path to cached SeqIO task created from input dataset.
    gcp_resources: GCP resources that can be used to track the custom job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='TfdsComparisonImporter',
          machine_type=machine_type,
          image_uri=image_uri,
          args=[
              '--app_name=text_comparison_importer',
              f'--input_text={input_text}',
              f'--inputs_field_name={inputs_field_name}',
              f'--comma_separated_candidates_field_names={comma_separated_candidates_field_names}',
              f'--choice_field_name={choice_field_name}',
              f'--split={split}',
              f'--output_cache_dir={output_dataset_path}',
              f'--instruction={instruction}',
              f'--large_model_reference={large_model_reference}',
              (
                  '--private_bucket_subdir='
                  f'{kfp.dsl.PIPELINE_TASK_NAME_PLACEHOLDER}_'
                  f'{kfp.dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
              ),
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
