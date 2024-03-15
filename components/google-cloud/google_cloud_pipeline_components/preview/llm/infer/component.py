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
from google_cloud_pipeline_components._implementation.llm import preprocess_chat_dataset
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
    prompt_dataset: str,
    model_checkpoint: Optional[str] = None,
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    sampling_strategy: str = 'greedy',
    instruction: Optional[str] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    accelerator_type: str = 'GPU',
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    encryption_spec_key_name: str = '',
) -> PipelineOutput:
  # fmt: off
  """Uses a large-language model to perform bulk inference on a prompt dataset.

  Args:
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    model_checkpoint: Optional Cloud storage path to the model checkpoint. If not provided, the default checkpoint for the `large_model_reference` will be used.
    prompt_dataset: Cloud storage path to an unlabled JSONL dataset that contains prompts. Text datasets must contain an `input_text` field that contains the prompt. Chat datasets must contain at least 1 message in a `messages` field. Each message must be valid JSON that contains `author` and `content` fields, where valid `author` values are `user` and `assistant` and `content` must be non-empty. Each row may contain multiple messages, but the first and last author must be the `user`. An optional `context` field may be provided for each example in a chat dataset. If provided, the `context` will preprended to the message `content`. The `instruction` serves as the default context. (Useful if most messages use the same system-level context.) Any context provided in the example will override the default value.
    prompt_sequence_length: Maximum tokenized sequence length for input text. Higher values increase memory overhead. This value should be at most 8192. Default value is 512.
    target_sequence_length:  Maximum tokenized sequence length for target text. Higher values increase memory overhead. This value should be at most 1024. Default value is 64.
    sampling_strategy: This field specifies the sampling strategy. The valid options are 'greedy' and 'temperature_sampling'.
    instruction: This field lets the model know what task it needs to perform. Base models have been trained over a large set of varied instructions. You can give a simple and intuitive description of the task and the model will follow it, e.g. "Classify this movie review as positive or negative" or "Translate this sentence to Danish". Do not specify this if your dataset already prepends the instruction to the inputs field.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    accelerator_type: One of 'TPU' or 'GPU'. If 'TPU' is specified, tuning components run in europe-west4. Otherwise tuning components run in us-central1 on GPUs. Default is 'GPU'.
    location: Location used to run non-tuning components, i.e. components that do not require accelerators. If not specified the location used to run the pipeline will be used.
    encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key. Note that this is not supported for TPU at the moment.

  Returns:
    Cloud storage path to output predictions.
  """
  # fmt: on
  prompt_column = 'input_text'
  machine_spec = function_based.resolve_machine_spec(
      accelerator_type=accelerator_type,
      use_test_spec=env.get_use_test_machine_spec(),
  ).set_display_name('Resolve Machine Spec')
  reference_model_metadata = function_based.resolve_reference_model_metadata(
      large_model_reference=large_model_reference,
      reference_model_path=model_checkpoint,
  ).set_display_name('Resolve Model Metadata')

  processed_dataset = preprocess_chat_dataset.preprocess_chat_dataset(
      large_model_reference=large_model_reference,
      input_dataset_uri=prompt_dataset,
      default_context=instruction,
      dataset_type='prompt',
  ).set_display_name('Preprocess Dataset')

  resolved_text_instruction = function_based.resolve_instruction(
      large_model_reference=large_model_reference,
      instruction=instruction,
  ).set_display_name('Resolve Instruction')
  prompt_dataset_importer = (
      private_text_importer.private_text_importer(
          project=project,
          location=location,
          input_text=processed_dataset.outputs['processed_dataset_uri'],
          inputs_field_name=prompt_column,
          targets_field_name='',  # ignore targets_field_name
          output_split_name=env.TRAIN_SPLIT,
          large_model_reference=reference_model_metadata.outputs[
              'large_model_reference'
          ],
          instruction=resolved_text_instruction.output,
          encryption_spec_key_name=encryption_spec_key_name,
      )
      .set_display_name('Import Prompt Dataset')
      .set_caching_options(False)
  )

  bulk_inferrer_image_uri = function_based.resolve_private_refined_image_uri(
      accelerator_type=machine_spec.outputs['accelerator_type'],
  ).set_display_name('Resolve Bulk Inferrer Image URI')
  bulk_inference = bulk_inferrer.bulk_inferrer(
      project=project,
      location=machine_spec.outputs['tuning_location'],
      input_model=reference_model_metadata.outputs['reference_model_path'],
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
      encryption_spec_key_name=encryption_spec_key_name,
  ).set_display_name('Bulk Inferrer')

  return PipelineOutput(
      output_prediction_gcs_path=bulk_inference.outputs[
          'output_prediction_gcs_path'
      ]
  )
