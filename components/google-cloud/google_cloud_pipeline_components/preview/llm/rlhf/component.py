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

import json
from typing import NamedTuple, Optional

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.llm import deploy_llm_model
from google_cloud_pipeline_components._implementation.llm import env
from google_cloud_pipeline_components._implementation.llm import function_based
from google_cloud_pipeline_components._implementation.llm import private_text_comparison_importer
from google_cloud_pipeline_components._implementation.llm import private_text_importer
from google_cloud_pipeline_components._implementation.llm import reinforcer
from google_cloud_pipeline_components._implementation.llm import reward_model_trainer
from google_cloud_pipeline_components._implementation.llm import upload_llm_model
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
    location: str = _placeholders.LOCATION_PLACEHOLDER,
) -> PipelineOutput:
  # fmt: off
  """Performs reinforcement learning from human feedback.

  Args:
    prompt_dataset: Cloud storage path to an unlabled prompt dataset used for reinforcement learning. The dataset format is jsonl. Each example in the dataset must have an `input_text` field that contains the prompt.
    preference_dataset: Cloud storage path to a human preference dataset used to train a reward model. The dataset format is jsonl. Each example in the dataset must contain the following fields: `input_text` that contains the prompt, `candidate_0` and `candidate_1` that contain candidate responses, `choice` that specifies the preferred candidate.
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
    eval_dataset: Optional Cloud storage path to an evaluation dataset. If provided, inference will be performed on this dataset after training. The dataset format is jsonl. Each example in the dataset must contain a field `input_text` that contains the prompt.
    project: Project used to run custom jobs. If not specified the project used to run the pipeline will be used.
    location: Location used to run custom jobs. If not specified the location used to run the pipeline will be used.

  Returns:
    model_resource_name: Path to the model uploaded to the Model Registry. This will be an empty string if the model was not deployed.
    endpoint_resource_name: Path the Online Prediction Endpoint. This will be an empty string if the model was not deployed.
  """
  # fmt: on
  policy_model_lora_dim = 1
  reward_model_lora_dim = 0
  batch_size = 64
  prompt_column = 'input_text'
  candidate_columns = ['candidate_0', 'candidate_1']
  choice_column = 'choice'
  upload_location = 'us-central1'
  machine_spec = function_based.resolve_machine_spec(
      location=location, use_test_spec=env.get_use_test_machine_spec()
  )

  reference_model_metadata = function_based.resolve_reference_model_metadata(
      large_model_reference=large_model_reference,
  ).set_display_name('BaseModelMetadataResolver')

  prompt_dataset_image_uri = function_based.resolve_private_image_uri(
      image_name='text_importer'
  ).set_display_name('PromptDatasetImageUriResolver')
  prompt_dataset_importer = private_text_importer.PrivateTextImporter(
      project=project,
      location=location,
      input_text=prompt_dataset,
      inputs_field_name=prompt_column,
      # Target field name does not matter because this field is not used.
      targets_field_name='non_existent_targets_field_name',
      output_split_name=env.TRAIN_SPLIT,
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
      image_uri=prompt_dataset_image_uri.output,
      instruction=instruction,
  ).set_display_name('PromptDatasetImporter')

  preference_dataset_image_uri = function_based.resolve_private_image_uri(
      image_name='text_comparison_importer'
  ).set_display_name('PreferenceDatasetImageUriResolver')
  comma_separated_candidates_field_names = (
      function_based.convert_to_delimited_string(items=candidate_columns)
  )
  preference_dataset_importer = private_text_comparison_importer.PrivateTextComparisonImporter(
      project=project,
      location=location,
      input_text=preference_dataset,
      inputs_field_name=prompt_column,
      comma_separated_candidates_field_names=comma_separated_candidates_field_names.output,
      choice_field_name=choice_column,
      split=env.TRAIN_SPLIT,
      large_model_reference=reference_model_metadata.outputs[
          'reward_model_reference'
      ],
      image_uri=preference_dataset_image_uri.output,
      instruction=instruction,
  ).set_display_name(
      'PreferenceDatasetImporter'
  )

  reward_model_image_uri = function_based.resolve_private_image_uri(
      image_name='reward_model',
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
  ).set_display_name('RewardModelImageUriResolver')
  reward_model = reward_model_trainer.RewardModelTrainer(
      project=project,
      location=location,
      input_model_path=reference_model_metadata.outputs['reward_model_path'],
      input_dataset_path=preference_dataset_importer.outputs[
          'output_dataset_path'
      ],
      train_steps=reward_model_train_steps,
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
      large_model_reference=reference_model_metadata.outputs[
          'reward_model_reference'
      ],
      machine_type=machine_spec.outputs['machine_type'],
      image_uri=reward_model_image_uri.output,
      inputs_sequence_length=prompt_sequence_length,
      targets_sequence_length=target_sequence_length,
      batch_size=batch_size,
      learning_rate_multiplier=reward_model_learning_rate_multiplier,
      lora_dim=reward_model_lora_dim,
  ).set_display_name('RewardModelTrainer')

  rl_image_uri = function_based.resolve_private_image_uri(
      image_name='reinforcer',
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
  ).set_display_name('ReinforcerImageUriResolver')
  rl_model = reinforcer.Reinforcer(
      project=project,
      location=location,
      input_reference_model_path=reference_model_metadata.outputs[
          'reference_model_path'
      ],
      input_reward_model_path=reward_model.outputs['output_model_path'],
      input_dataset_path=prompt_dataset_importer.outputs['imported_data_path'],
      train_steps=reinforcement_learning_train_steps,
      accelerator_type=machine_spec.outputs['accelerator_type'],
      accelerator_count=machine_spec.outputs['accelerator_count'],
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
      reward_model_reference=reference_model_metadata.outputs[
          'reward_model_reference'
      ],
      machine_type=machine_spec.outputs['machine_type'],
      image_uri=rl_image_uri.output,
      inputs_sequence_length=prompt_sequence_length,
      targets_sequence_length=target_sequence_length,
      batch_size=batch_size,
      learning_rate_multiplier=reinforcement_learning_rate_multiplier,
      kl_coeff=kl_coeff,
      lora_dim=policy_model_lora_dim,
  ).set_display_name('Reinforcer')

  should_perform_inference = function_based.value_exists(value=eval_dataset)
  with kfp.dsl.Condition(
      should_perform_inference.output == True, name='Perform Inference'  # pylint: disable=singleton-comparison
  ):
    component.infer_pipeline(
        project=project,
        location=location,
        large_model_reference=large_model_reference,
        model_checkpoint=rl_model.outputs['output_model_path'],
        prompt_dataset=eval_dataset,
        prompt_sequence_length=prompt_sequence_length,
        target_sequence_length=target_sequence_length,
        instruction=instruction,
    )

  adapter_artifact = kfp.dsl.importer(
      artifact_uri=rl_model.outputs['output_adapter_path'],
      artifact_class=kfp.dsl.Artifact,
  )
  regional_endpoint = function_based.resolve_regional_endpoint(
      upload_location=upload_location
  )
  display_name = function_based.resolve_model_display_name(
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
      model_display_name=model_display_name,
  )
  upload_model = function_based.resolve_upload_model(
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ]
  )
  upload_task = upload_llm_model.upload_llm_model(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=upload_location,
      regional_endpoint=regional_endpoint.output,
      artifact_uri=adapter_artifact.output,
      model_display_name=display_name.output,
      model_reference_name='text-bison@001',
      upload_model=upload_model.output,
  ).set_env_variable(
      name='VERTEX_AI_PIPELINES_RUN_LABELS',
      value=json.dumps({'tune-type': 'rlhf'}),
  )
  deploy_model = function_based.resolve_deploy_model(
      deploy_model=deploy_model,
      large_model_reference=reference_model_metadata.outputs[
          'large_model_reference'
      ],
  )
  deploy_task = deploy_llm_model.create_endpoint_and_deploy_model(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=upload_location,
      model_resource_name=upload_task.outputs['model_resource_name'],
      display_name=display_name.output,
      regional_endpoint=regional_endpoint.output,
      deploy_model=deploy_model.output,
  )

  return PipelineOutput(
      model_resource_name=upload_task.outputs['model_resource_name'],
      endpoint_resource_name=deploy_task.outputs['endpoint_resource_name'],
  )
