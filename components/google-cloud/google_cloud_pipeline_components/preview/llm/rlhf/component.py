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
"""Defines a pipeline that performs reinforcement learning from human feedback."""

from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import deployment_graph
from google_cloud_pipeline_components._implementation.llm import env
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import reinforcement_learning_graph
from google_cloud_pipeline_components._implementation.llm import reward_model_graph
from google_cloud_pipeline_components._implementation.llm import rlhf_preprocessor
from google_cloud_pipeline_components._implementation.llm import utils
from google_cloud_pipeline_components._implementation.llm import validate_pipeline
from google_cloud_pipeline_components.preview.llm.infer import component
import kfp

PipelineOutput = NamedTuple(
    'Outputs', model_resource_name=str, endpoint_resource_name=str
)


@kfp.dsl.pipeline(
    name='rlhf-train-template',
    description='Performs reinforcement learning from human feedback.',
)
def rlhf_pipeline(
    prompt_dataset: str,
    preference_dataset: str,
    large_model_reference: str,
    model_display_name: Optional[str] = None,
    prompt_sequence_length: int = 512,
    target_sequence_length: int = 64,
    reward_model_learning_rate_multiplier: float = 1.0,
    reinforcement_learning_rate_multiplier: float = 1.0,
    reward_model_train_steps: int = 1000,
    reinforcement_learning_train_steps: int = 1000,
    kl_coeff: float = 0.1,
    instruction: Optional[str] = None,
    deploy_model: bool = True,
    eval_dataset: Optional[str] = None,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    accelerator_type: str = 'GPU',
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    encryption_spec_key_name: str = '',
    tensorboard_resource_id: str = '',
) -> PipelineOutput:
  # fmt: off
  """Performs reinforcement learning from human feedback.

  Args:
    prompt_dataset: Cloud storage path to an unlabled JSONL dataset that contains prompts. Text datasets must contain an `input_text` field that contains the prompt. Chat datasets must contain at least 1 message in a `messages` field. Each message must be valid JSON that contains `author` and `content` fields, where valid `author` values are `user` and `assistant` and `content` must be non-empty. Each row may contain multiple messages, but the first and last author must be the `user`. An optional `context` field may be provided for each example in a chat dataset. If provided, the `context` will preprended to the message `content`. The `instruction` serves as the default context. (Useful if most messages use the same system-level context.) Any context provided in the example will override the default value.
    preference_dataset: Cloud storage path to a human preference JSONL dataset used to train a reward model. Each example in a preference dataset must contain `candidate_0` and `candidate_1` fields that contain candidate responses, `choice` that specifies the preferred candidate and either `input_text` (if tuning a text model) or `messages` (if tuning a chat model). Chat datasets must contain at least 1 message in a `messages` field. Each message must be valid JSON that contains `author` and `content` fields, where valid `author` values are `user` and `assistant` and `content` must be non-empty. Each row may contain multiple messages, but the first and last author must be the `user`. An optional `context` field may be provided for each example in a chat dataset. If provided, the `context` will preprended to the message `content`. The `instruction` serves as the default context. (Useful if most messages use the same system-level context.) Any context provided in the example will override the default value.
    large_model_reference: Name of the base model. Supported values are `text-bison@001`, `t5-small`, `t5-large`, `t5-xl` and `t5-xxl`. `text-bison@001` and `t5-small` are supported in `us-central1` and `europe-west4`. `t5-large`, `t5-xl` and `t5-xxl` are only supported in `europe-west4`.
    model_display_name: Name of the fine-tuned model shown in the Model Registry. If not provided, a default name will be created.
    prompt_sequence_length: Maximum tokenized sequence length for input text. Higher values increase memory overhead. This value should be at most 8192. Default value is 512.
    target_sequence_length:  Maximum tokenized sequence length for target text. Higher values increase memory overhead. This value should be at most 1024. Default value is 64.
    reward_model_learning_rate_multiplier: Constant used to adjust the base learning rate used when training a reward model. Multiply by a number > 1 to increase the magnitude of updates applied at each training step or multiply by a number < 1 to decrease the magnitude of updates. Default value is 1.0.
    reinforcement_learning_rate_multiplier: Constant used to adjust the base learning rate used during reinforcement learning. Multiply by a number > 1 to increase the magnitude of updates applied at each training step or multiply by a number < 1 to decrease the magnitude of updates. Default value is 1.0.
    reward_model_train_steps: Number of steps to use when training a reward model. Default value is 1000.
    reinforcement_learning_train_steps: Number of reinforcement learning steps to perform when tuning a base model. Default value is 1000.
    kl_coeff: Coefficient for KL penalty. This regularizes the policy model and penalizes if it diverges from its initial distribution. If set to 0, the reference language model is not loaded into memory. Default value is 0.1.
    instruction: This field lets the model know what task it needs to perform. Base models have been trained over a large set of varied instructions. You can give a simple and intuitive description of the task and the model will follow it, e.g. "Classify this movie review as positive or negative" or "Translate this sentence to Danish". Do not specify this if your dataset already prepends the instruction to the inputs field.
    deploy_model: Whether to deploy the model to an endpoint in `us-central1`. Default is True.
    eval_dataset: Optional Cloud storage path to an evaluation dataset. The dataset format is jsonl. The evaluation dataset can be used to compute train-time metrics (when training a reward model) or perform bulk inference for third-party models. To compute train-time metrics this dataset must contain the same fields as the peference dataset. For bulk inference with third-party models only `input_text` is needed. Note, train-time metrics are only computed for the first 5000 samples in the dataset for efficient evaluation during training.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    accelerator_type: One of 'TPU' or 'GPU'. If 'TPU' is specified, tuning components run in europe-west4. Otherwise tuning components run in us-central1 on GPUs. Default is 'GPU'.
    location: Location used to run non-tuning components, i.e. components that do not require accelerators. If not specified the location used to run the pipeline will be used.
    encryption_spec_key_name: Customer-managed encryption key. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key. Note that this is not supported for TPU at the moment.
    tensorboard_resource_id: Optional tensorboard resource id in format `projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}`. If provided, tensorboard metrics will be uploaded to this location.

  Returns:
    model_resource_name: Path to the model uploaded to the Model Registry. This will be an empty string if the model was not deployed.
    endpoint_resource_name: Path the Online Prediction Endpoint. This will be an empty string if the model was not deployed.
  """
  # fmt: on
  # LoRA dim for reward model
  reward_lora_dim = 4

  validate_pipeline_task = validate_pipeline.validate_pipeline(
      accelerator_type=accelerator_type,
      location=location,
      encryption_spec_key_name=encryption_spec_key_name,
      eval_dataset=eval_dataset,
  ).set_display_name('Validate Inputs')

  preprocess_metadata = rlhf_preprocessor.rlhf_preprocessor(
      large_model_reference=large_model_reference,
      accelerator_type=accelerator_type,
      use_test_spec=env.get_use_test_machine_spec(),
      project=env.PRIVATE_ARTIFACT_REGISTRY_PROJECT,
      location=env.PRIVATE_ARTIFACT_REGISTRY_LOCATION,
      artifact_registry=env.PRIVATE_ARTIFACT_REGISTRY,
      tag=env.get_private_image_tag(),
      evaluation_dataset=eval_dataset,
      tensorboard_resource_id=tensorboard_resource_id,
      upload_location=location,
      model_display_name=model_display_name,
      deploy_model=deploy_model,
  ).set_display_name('Preprocess Inputs')
  num_microbatches = preprocess_metadata.outputs['metadata_num_microbatches']

  reward_model_pipeline = (
      (
          reward_model_graph.pipeline(
              preference_dataset=preference_dataset,
              large_model_reference=large_model_reference,
              reward_model_reference=preprocess_metadata.outputs[
                  'metadata_reward_model_reference'
              ],
              reward_model_path=preprocess_metadata.outputs[
                  'metadata_reward_model_path'
              ],
              machine_type=preprocess_metadata.outputs['metadata_machine_type'],
              tuning_location=preprocess_metadata.outputs[
                  'metadata_tuning_location'
              ],
              accelerator_type=preprocess_metadata.outputs[
                  'metadata_accelerator_type'
              ],
              accelerator_count=preprocess_metadata.outputs[
                  'metadata_accelerator_count'
              ],
              reward_model_image_uri=preprocess_metadata.outputs[
                  'metadata_refined_image_uri'
              ],
              comma_separated_candidates_field_names=preprocess_metadata.outputs[
                  'metadata_candidate_columns_string'
              ],
              prompt_sequence_length=prompt_sequence_length,
              target_sequence_length=target_sequence_length,
              eval_dataset=validate_pipeline_task.outputs[
                  'reward_model_eval_dataset'
              ],
              instruction=instruction,
              reward_model_learning_rate_multiplier=reward_model_learning_rate_multiplier,
              reward_model_train_steps=reward_model_train_steps,
              lora_dim=reward_lora_dim,
              project=project,
              location=location,
              tensorboard_resource_id=tensorboard_resource_id,
              encryption_spec_key_name=encryption_spec_key_name,
              num_microbatches=num_microbatches,
          )
      )
      .set_display_name('Train Reward Model')
      .after(validate_pipeline_task)
  )
  rl_model_pipeline = reinforcement_learning_graph.pipeline(
      prompt_dataset=prompt_dataset,
      input_reward_model_path=preprocess_metadata.outputs[
          'metadata_reward_model_path'
      ],
      input_reward_adapter_path=reward_model_pipeline.outputs[
          'reward_model_adapter_path'
      ],
      input_preference_dataset_path=reward_model_pipeline.outputs[
          'reward_dataset_path'
      ],
      large_model_reference=large_model_reference,
      reward_model_reference=preprocess_metadata.outputs[
          'metadata_reward_model_reference'
      ],
      policy_model_reference=preprocess_metadata.outputs[
          'metadata_large_model_reference'
      ],
      policy_model_path=preprocess_metadata.outputs[
          'metadata_reference_model_path'
      ],
      machine_type=preprocess_metadata.outputs['metadata_machine_type'],
      tuning_location=preprocess_metadata.outputs['metadata_tuning_location'],
      accelerator_type=preprocess_metadata.outputs['metadata_accelerator_type'],
      accelerator_count=preprocess_metadata.outputs[
          'metadata_accelerator_count'
      ],
      rl_image_uri=preprocess_metadata.outputs['metadata_refined_image_uri'],
      prompt_sequence_length=prompt_sequence_length,
      target_sequence_length=target_sequence_length,
      reinforcement_learning_rate_multiplier=reinforcement_learning_rate_multiplier,
      reinforcement_learning_train_steps=reinforcement_learning_train_steps,
      kl_coeff=kl_coeff,
      instruction=instruction,
      reward_lora_dim=reward_lora_dim,
      project=project,
      location=location,
      tensorboard_resource_id=tensorboard_resource_id,
      encryption_spec_key_name=encryption_spec_key_name,
      num_microbatches=num_microbatches,
  ).set_display_name('Reinforcement Learning')

  has_inference_dataset = preprocess_metadata.outputs['has_inference_dataset']

  with kfp.dsl.Condition(
      has_inference_dataset == True,  # pylint: disable=singleton-comparison
      name='Perform Inference',
  ):
    with kfp.dsl.If(
        rl_model_pipeline.outputs['output_model_path'] != '',
        name='CheckModel Checkpoint Exists',
    ):
      component.infer_pipeline(
          project=project,
          location=location,
          large_model_reference=large_model_reference,
          model_checkpoint=rl_model_pipeline.outputs['output_model_path'],
          prompt_dataset=eval_dataset,
          prompt_sequence_length=prompt_sequence_length,
          target_sequence_length=target_sequence_length,
          instruction=instruction,
          accelerator_type=accelerator_type,
          encryption_spec_key_name=encryption_spec_key_name,
      )

  llm_model_handler = deployment_graph.pipeline(
      output_adapter_path=rl_model_pipeline.outputs['output_adapter_path'],
      large_model_reference=large_model_reference,
      policy_model_reference=preprocess_metadata.outputs[
          'metadata_large_model_reference'
      ],
      model_display_name=preprocess_metadata.outputs[
          'metadata_model_display_name'
      ],
      deploy_model=preprocess_metadata.outputs['metadata_deploy_model'],
      upload_model=preprocess_metadata.outputs['metadata_upload_model'],
      encryption_spec_key_name=encryption_spec_key_name,
      upload_location=location,
      regional_endpoint=preprocess_metadata.outputs['metadata_upload_location'],
  ).set_display_name('Upload and Deploy Tuned Model')

  return PipelineOutput(
      model_resource_name=llm_model_handler.outputs['model_resource_name'],
      endpoint_resource_name=llm_model_handler.outputs[
          'endpoint_resource_name'
      ],
  )
