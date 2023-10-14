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
"""KFP container component that trains a reward model."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
import kfp


@kfp.dsl.container_component
def RewardModelTrainer(  # pylint: disable=invalid-name
    project: str,
    location: str,
    train_steps: int,
    accelerator_type: str,
    accelerator_count: int,
    large_model_reference: str,
    machine_type: str,
    image_uri: str,
    inputs_sequence_length: int,
    targets_sequence_length: int,
    input_model_path: str,
    input_dataset_path: str,
    output_model_path: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    tensorboard_metrics: kfp.dsl.Output[kfp.dsl.Artifact],  # pytype: disable=unsupported-operands
    gcp_resources: kfp.dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    train_split: str = 'train',
    batch_size: int = 64,
    learning_rate_multiplier: float = 1.0,
    lora_dim: int = 0,
) -> kfp.dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Trains a reward model.

  Args:
    project: Project used to run the job.
    location: Location used to run the job.
    input_model_path: Path to the base model to fine tune.
    input_dataset_path: Path to dataset to use to train a reward model.
    train_steps: Number of training steps. These are the number of steps
      on top of any steps used to train the base model.
    accelerator_type: Type of TPU accelerator. Can be either TPU_V2 or TPU_V3.
    accelerator_count: Number of TPU accelerators.
    large_model_reference: Predefined model used to create the ``input_model``.
    machine_type: The type of the machine to provision for the custom job. Must
      be a valid GCE instance type and compatible with the accelerator type.
    image_uri: Location of reward model Docker image.
    inputs_sequence_length: Maximum number of input tokens per row.
    targets_sequence_length: Maximum number of target tokens per row.
    train_split: Name of the split in the input dataset that contains training
      data. Default is ``'train'``.
    batch_size: Number of examples in each finetuning step. Default is 64.
    lora_dim: The rank of the LoRA adapter. If >0, then use LoRA-tuning. If =0,
      then use full-tuning.
    learning_rate_multiplier: Constant multiplied by the base learning rate used
      to adjust the learning rate when training a reward model.

  Returns:
    output_model: Trained reward model.
    tensorboard_metrics: Training stats (tensorboard) path.
    gcp_resources: GCP resources that can be used to track the custom finetuning
      job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='RewardModelTrainer',
          accelerator_type=accelerator_type,
          accelerator_count=accelerator_count,
          machine_type=machine_type,
          image_uri=image_uri,
          args=[
              f'--train_steps={train_steps}',
              f'--input_model_path={input_model_path}',
              f'--input_dataset_path={input_dataset_path}',
              f'--output_model_path={output_model_path}',
              f'--tensorboard_metrics_path={tensorboard_metrics.path}',
              f'--large_model_reference={large_model_reference}',
              f'--inputs_sequence_length={inputs_sequence_length}',
              f'--targets_sequence_length={targets_sequence_length}',
              f'--train_split={train_split}',
              f'--batch_size={batch_size}',
              f'--learning_rate_multiplier={learning_rate_multiplier}',
              (
                  '--private_bucket_subdir='
                  f'{kfp.dsl.PIPELINE_TASK_NAME_PLACEHOLDER}_'
                  f'{kfp.dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
              ),
              f'--lora_dim={lora_dim}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
