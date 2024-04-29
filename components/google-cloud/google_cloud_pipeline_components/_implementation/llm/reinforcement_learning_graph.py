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
"""Graph component for preparing data and performing reinforcement learning."""

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import env
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import preprocess_chat_dataset
from google_cloud_pipeline_components._implementation.llm import private_text_importer
from google_cloud_pipeline_components._implementation.llm import reinforcer
from google_cloud_pipeline_components._implementation.llm import upload_tensorboard_metrics
import kfp

PipelineOutput = NamedTuple(
    'Outputs',
    output_model_path=str,
    output_adapter_path=str,
)


@kfp.dsl.pipeline(
    name='reinforcement-learning-graph',
    description='Prepares prompt data and trains an RL model.',
)
def pipeline(
    prompt_dataset: str,
    input_reward_model_path: str,
    input_reward_adapter_path: str,
    input_preference_dataset_path: str,
    large_model_reference: str,
    reward_model_reference: str,
    policy_model_reference: str,
    policy_model_path: str,
    machine_type: str,
    tuning_location: str,
    accelerator_type: str,
    accelerator_count: int,
    rl_image_uri: str,
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    lora_dim: int = 1,
    reward_lora_dim: int = 4,
    batch_size: int = 64,
    reinforcement_learning_rate_multiplier: float = 1.0,
    reinforcement_learning_train_steps: int = 1000,
    kl_coeff: float = 0.1,
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
    prompt_dataset: Cloud storage path to an unlabled JSONL dataset that contains prompts. Text datasets must contain an `input_text` field that contains the prompt. Chat datasets must contain at least 1 message in a `messages` field. Each message must be valid JSON that contains `author` and `content` fields, where valid `author` values are `user` and `assistant` and `content` must be non-empty. Each row may contain multiple messages, but the first and last author must be the `user`. An optional `context` field may be provided for each example in a chat dataset. If provided, the `context` will preprended to the message `content`. The `instruction` serves as the default context. (Useful if most messages use the same system-level context.) Any context provided in the example will override the default value.
    input_reward_adapter_path: Path to the reward LoRA adapter to use during reinforcement learning.
    input_preference_dataset_path: Path to preference dataset used by the reward model.
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    reward_model_reference: Name of the reward model. The name should be in capitalized snake case format.
    policy_model_reference: Name of the policy model. The name should be in capitalized snake case format.
    policy_model_path: The model checkpoint path to the reinforcer model.
    machine_type: The type of the machine to provision for the custom job. Must be a valid GCE instance type and compatible with the accelerator type.
    tuning_location: The GCP region to run the custom job.
    accelerator_type: Specific accelerator type for the custom job.
    accelerator_count: The number of accelerator.
    rl_image_uri: Docker image URI to use for the reinforcement learning training job.
    prompt_sequence_length: Maximum tokenized sequence length for input text. Higher values increase memory overhead. This value should be at most 8192. Default value is 512.
    target_sequence_length: Maximum tokenized sequence length for target text. Higher values increase memory overhead. This value should be at most 1024. Default value is 64.
    lora_dim: The rank of the LoRA adapter. If >0, then use LoRA-tuning. If =0, then use full-tuning. Default is 1.
    reward_lora_dim: The rank of the reward LoRA adapter. Full tuning is not support for the reward model. Default is 4.
    batch_size: Number of examples in each finetuning step. Default is 64.
    reinforcement_learning_rate_multiplier: Constant used to adjust the base learning rate used during reinforcement learning. Multiply by a number > 1 to increase the magnitude of updates applied at each training step or multiply by a number < 1 to decrease the magnitude of updates. Default value is 1.0.
    reinforcement_learning_train_steps: Number of reinforcement learning steps to perform when tuning a base model. Default value is 1000.
    kl_coeff: Coefficient for KL penalty. This regularizes the policy model and penalizes if it diverges from its initial distribution. If set to 0, the reference language model is not loaded into memory. Default value is 0.1.
    instruction: This field lets the model know what task it needs to perform. Base models have been trained over a large set of varied instructions. You can give a simple and intuitive description of the task and the model will follow it, e.g. "Classify this movie review as positive or negative" or "Translate this sentence to Danish". Do not specify this if your dataset already prepends the instruction to the inputs field.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    location: Location used to run non-tuning components, i.e. components that do not require accelerators. If not specified the location used to run the pipeline will be used.
    tensorboard_resource_id: Optional tensorboard resource id in format `projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}`. If provided, tensorboard metrics will be uploaded to this location.
    encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key. Note that this is not supported for TPU at the moment.

  Returns:
    output_model_path: Path to the trained model checkpoint.
    output_adapter_path: Path to the trained model adapter if LoRA tuning was used.
  """
  # fmt: on
  prompt_column = 'input_text'

  processed_dataset = preprocess_chat_dataset.preprocess_chat_dataset(
      large_model_reference=large_model_reference,
      input_dataset_uri=prompt_dataset,
      default_context=instruction,
      dataset_type='prompt',
  ).set_display_name('Preprocess Prompt Dataset')
  prompt_dataset_importer = (
      private_text_importer.private_text_importer(
          project=project,
          location=location,
          input_text=processed_dataset.outputs['processed_dataset_uri'],
          inputs_field_name=prompt_column,
          # Target field name does not matter because this field is not used.
          targets_field_name='non_existent_targets_field_name',
          output_split_name=env.TRAIN_SPLIT,
          large_model_reference=policy_model_reference,
          instruction=instruction,
          encryption_spec_key_name=encryption_spec_key_name,
      )
      .set_display_name('Import Prompt Dataset')
      .set_caching_options(False)
  )
  rl_model = (
      reinforcer.reinforcer(
          project=project,
          location=tuning_location,
          input_reference_model_path=policy_model_path,
          input_reward_model_path=input_reward_model_path,
          input_reward_adapter_path=input_reward_adapter_path,
          input_dataset_path=prompt_dataset_importer.outputs[
              'imported_data_path'
          ],
          input_preference_dataset_path=input_preference_dataset_path,
          train_steps=reinforcement_learning_train_steps,
          accelerator_type=accelerator_type,
          accelerator_count=accelerator_count,
          large_model_reference=policy_model_reference,
          reward_model_reference=reward_model_reference,
          machine_type=machine_type,
          image_uri=rl_image_uri,
          inputs_sequence_length=prompt_sequence_length,
          targets_sequence_length=target_sequence_length,
          batch_size=batch_size,
          learning_rate_multiplier=reinforcement_learning_rate_multiplier,
          kl_coeff=kl_coeff,
          lora_dim=lora_dim,
          reward_lora_dim=reward_lora_dim,
          num_microbatches=num_microbatches,
          encryption_spec_key_name=encryption_spec_key_name,
          tensorboard_resource_id=tensorboard_resource_id,
      )
      .set_display_name('Reinforcer')
      .set_caching_options(False)
  )

  return PipelineOutput(
      output_model_path=rl_model.outputs['output_model_path'],
      output_adapter_path=rl_model.outputs['output_adapter_path'],
  )
