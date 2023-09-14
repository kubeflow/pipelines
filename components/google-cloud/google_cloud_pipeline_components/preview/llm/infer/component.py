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
"""Pipeline that performs bulk inference using a large-language model."""

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import bulk_inferrer
from google_cloud_pipeline_components._implementation.llm import env
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import private_text_importer
import kfp

PipelineOutput = NamedTuple('Outputs', output_prediction_gcs_path=str)


@kfp.dsl.pipeline(
    name='infer-eval-template',
    description=(
        'Performs bulk inference on a dataset using a model checkpoint.'
    ),
)
def infer_pipeline(
    large_model_reference: str,
    model_checkpoint: str,
    prompt_dataset: str,
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    sampling_strategy: str = 'greedy',
    instruction: Optional[str] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
) -> PipelineOutput:
  # fmt: off
  """Uses a large-language model to perform bulk inference on a prompt dataset.

  Args:
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    model_checkpoint: Cloud storage path to the model checkpoint.
    prompt_dataset: Cloud storage path to an unlabled prompt dataset used for reinforcement learning. The dataset format is jsonl. Each example in the dataset must have an `input_text` field that contains the prompt.
    prompt_sequence_length: Maximum tokenized sequence length for input text. Higher values increase memory overhead. This value should be at most 8192. Default value is 512.
    target_sequence_length:  Maximum tokenized sequence length for target text. Higher values increase memory overhead. This value should be at most 1024. Default value is 64.
    sampling_strategy: This field specifies the sampling strategy. The valid options are 'greedy' and 'temperature_sampling'.
    instruction: This field lets the model know what task it needs to perform. Base models have been trained over a large set of varied instructions. You can give a simple and intuitive description of the task and the model will follow it, e.g. "Classify this movie review as positive or negative" or "Translate this sentence to Danish". Do not specify this if your dataset already prepends the instruction to the inputs field.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    location: Location used to run custom jobs. If not specified the location used to run the pipeline will be used.

  Returns:
    Cloud storage path to output predictions.
  """
  # fmt: on
  prompt_column = 'input_text'
  machine_spec = function_based.resolve_machine_spec(
      location=location,
      use_test_spec=env.get_use_test_machine_spec(),
  )
  reference_model_metadata = function_based.resolve_reference_model_metadata(
      large_model_reference=large_model_reference
  ).set_display_name('BaseModelMetadataResolver')

  prompt_dataset_image_uri = function_based.resolve_private_image_uri(
      image_name='text_importer',
  ).set_display_name('PromptDatasetImageUriResolver')
  prompt_dataset_importer = private_text_importer.PrivateTextImporter(
      project=project,
      location=location,
      input_text=prompt_dataset,
      inputs_field_name=prompt_column,
      targets_field_name='',  # ignore targets_field_name
      output_split_name=env.TRAIN_SPLIT,
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
      image_uri=prompt_dataset_image_uri.output,
      instruction=instruction,
  ).set_display_name('PromptDatasetImporter')

  bulk_inferrer_image_uri = function_based.resolve_private_image_uri(
      image_name='infer',
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
  ).set_display_name('BulkInferrerImageUriResolver')
  bulk_inference = bulk_inferrer.BulkInferrer(
      project=project,
      location=location,
      input_model=model_checkpoint,
      input_dataset_path=prompt_dataset_importer.outputs['imported_data_path'],
      dataset_split=env.TRAIN_SPLIT,
      inputs_sequence_length=prompt_sequence_length,
      targets_sequence_length=target_sequence_length,
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
      sampling_strategy=sampling_strategy,
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
      machine_type=machine_spec.outputs['machine_type'],
      image_uri=bulk_inferrer_image_uri.output,
  ).set_display_name('Bulk Inferrer')

  return PipelineOutput(
      output_prediction_gcs_path=bulk_inference.outputs[
          'output_prediction_gcs_path'
      ]
  )
