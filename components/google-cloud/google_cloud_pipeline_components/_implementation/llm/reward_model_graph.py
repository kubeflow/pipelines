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
"""Graph Component for preparing data and training a reward model ."""

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import env
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import preprocess_chat_dataset
from google_cloud_pipeline_components._implementation.llm import private_text_comparison_importer
from google_cloud_pipeline_components._implementation.llm import reward_model_trainer
from google_cloud_pipeline_components._implementation.llm import rlhf_preprocessor
from google_cloud_pipeline_components._implementation.llm import upload_tensorboard_metrics
import kfp

PipelineOutput = NamedTuple(
    'Outputs',
    reward_model_adapter_path=str,
    reward_dataset_path=str,
)


@kfp.dsl.pipeline(
    name='reward-model-graph',
    description='Prepares preference data and trains a reward model.',
)
def pipeline(
    preference_dataset: str,
    large_model_reference: str,
    reward_model_reference: str,
    reward_model_path: str,
    machine_type: str,
    tuning_location: str,
    accelerator_type: str,
    accelerator_count: int,
    reward_model_image_uri: str,
    comma_separated_candidates_field_names: str,
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    batch_size: int = 64,
    lora_dim: int = 4,
    reward_model_learning_rate_multiplier: float = 1.0,
    reward_model_train_steps: int = 1000,
    eval_dataset: Optional[str] = None,
    instruction: Optional[str] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    tensorboard_resource_id: str = '',
    encryption_spec_key_name: str = '',
    num_microbatches: int = 0,
) -> PipelineOutput:
  # fmt: off
  """Trains a reward model.

  Args:
    preference_dataset: Cloud storage path to a human preference JSONL dataset used to train a reward model. Each example in a preference dataset must contain `candidate_0` and `candidate_1` fields that contain candidate responses, `choice` that specifies the preferred candidate and either `input_text` (if tuning a text model) or `messages` (if tuning a chat model). Chat datasets must contain at least 1 message in a `messages` field. Each message must be valid JSON that contains `author` and `content` fields, where valid `author` values are `user` and `assistant` and `content` must be non-empty. Each row may contain multiple messages, but the first and last author must be the `user`. An optional `context` field may be provided for each example in a chat dataset. If provided, the `context` will preprended to the message `content`. The `instruction` serves as the default context. (Useful if most messages use the same system-level context.) Any context provided in the example will override the default value.
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    reward_model_reference: Name of the base model. The name should be in capitalized snake case format.
    reward_model_path: The model checkpoint path for the reward model.
    machine_type: The type of the machine to provision for the custom job. Must be a valid GCE instance type and compatible with the accelerator type.
    tuning_location: The GCP region to run the custom job.
    accelerator_type: Specific accelerator type for the custom job.
    accelerator_count: The number of accelerator.
    reward_model_image_uri: Docker image URI to use for the reward model training job.
    comma_separated_candidates_field_names: Comma separated list of fields that contain candidate text, e.g. ``'field_1,field_2,field_3'``.
    prompt_sequence_length: Maximum tokenized sequence length for input text. Higher values increase memory overhead. This value should be at most 8192. Default value is 512.
    target_sequence_length:  Maximum tokenized sequence length for target text. Higher values increase memory overhead. This value should be at most 1024. Default value is 64.
    batch_size: Number of examples in each finetuning step. Default is 64.
    lora_dim: The rank of the LoRA adapter. If >0, then use LoRA-tuning. Full tuning is not supported for the reward model. Default is 4.
    reward_model_learning_rate_multiplier: Constant used to adjust the base learning rate used when training a reward model. Multiply by a number > 1 to increase the magnitude of updates applied at each training step or multiply by a number < 1 to decrease the magnitude of updates. Default value is 1.0.
    reward_model_train_steps: Number of steps to use when training a reward model. Default value is 1000.
    instruction: This field lets the model know what task it needs to perform. Base models have been trained over a large set of varied instructions. You can give a simple and intuitive description of the task and the model will follow it, e.g. "Classify this movie review as positive or negative" or "Translate this sentence to Danish". Do not specify this if your dataset already prepends the instruction to the inputs field.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    location: Location used to run non-tuning components, i.e. components that do not require accelerators. If not specified the location used to run the pipeline will be used.
    tensorboard_resource_id: Optional tensorboard resource id in format `projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}`. If provided, tensorboard metrics will be uploaded to this location.
    encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key. Note that this is not supported for TPU at the moment.
    num_microbatches: The number of microbatches to break the total batch size into during training.

  Returns:
    reward_model_adapter_path: Path to the output LoRA adapter.
    reward_dataset_path: Preference dataset use for tuning the reward model.
  """
  # fmt: on
  prompt_column = 'input_text'
  choice_column = 'choice'

  processed_preference_dataset = (
      preprocess_chat_dataset.preprocess_chat_dataset(
          large_model_reference=large_model_reference,
          input_dataset_uri=preference_dataset,
          default_context=instruction,
          dataset_type='preference',
      ).set_display_name('Preprocess Prompt Dataset')
  )

  preference_dataset_importer = (
      private_text_comparison_importer.private_text_comparison_importer(
          project=project,
          location=location,
          input_text=processed_preference_dataset.outputs[
              'processed_dataset_uri'
          ],
          inputs_field_name=prompt_column,
          comma_separated_candidates_field_names=comma_separated_candidates_field_names,
          choice_field_name=choice_column,
          split=env.TRAIN_SPLIT,
          large_model_reference=reward_model_reference,
          instruction=instruction,
          encryption_spec_key_name=encryption_spec_key_name,
      )
      .set_display_name('Import Preference Dataset')
      .set_caching_options(False)
  )

  preference_eval_dataset_importer = (
      private_text_comparison_importer.private_text_comparison_importer(
          project=project,
          location=location,
          input_text=eval_dataset,
          inputs_field_name=prompt_column,
          comma_separated_candidates_field_names=comma_separated_candidates_field_names,
          choice_field_name=choice_column,
          split=env.TRAIN_SPLIT,
          large_model_reference=reward_model_reference,
          instruction=instruction,
          encryption_spec_key_name=encryption_spec_key_name,
      )
      .set_display_name('Import Preference Eval Dataset')
      .set_caching_options(False)
  )

  reward_model = (
      reward_model_trainer.reward_model_trainer(
          project=project,
          location=tuning_location,
          input_model_path=reward_model_path,
          input_dataset_path=preference_dataset_importer.outputs[
              'output_dataset_path'
          ],
          eval_dataset_path=preference_eval_dataset_importer.outputs[
              'output_dataset_path'
          ],
          train_steps=reward_model_train_steps,
          accelerator_type=accelerator_type,
          accelerator_count=accelerator_count,
          large_model_reference=reward_model_reference,
          machine_type=machine_type,
          image_uri=reward_model_image_uri,
          inputs_sequence_length=prompt_sequence_length,
          targets_sequence_length=target_sequence_length,
          batch_size=batch_size,
          learning_rate_multiplier=reward_model_learning_rate_multiplier,
          lora_dim=lora_dim,
          num_microbatches=num_microbatches,
          encryption_spec_key_name=encryption_spec_key_name,
          tensorboard_resource_id=tensorboard_resource_id,
      )
      .set_display_name('Reward Model Trainer')
      .set_caching_options(False)
  )

  return PipelineOutput(
      reward_model_adapter_path=reward_model.outputs['output_adapter_path'],
      reward_dataset_path=preference_dataset_importer.outputs[
          'output_dataset_path'
      ],
  )
