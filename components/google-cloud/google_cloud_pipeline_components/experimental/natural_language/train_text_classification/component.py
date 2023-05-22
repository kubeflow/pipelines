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


from typing import Optional

from google_cloud_pipeline_components import _image
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Output


@dsl.container_component
def train_tfhub_model(
    project: str,
    location: str,
    input_data_path: str,
    gcp_resources: dsl.OutputPath(str),
    model_output: Output[Artifact],
    input_format: Optional[str] = 'JSONL',
    machine_type: Optional[str] = 'n1-highmem-8',
    accelerator_type: Optional[str] = 'NVIDIA_TESLA_T4',
    accelerator_count: Optional[int] = 1,
    natural_language_task_type: Optional[str] = 'CLASSIFICATION',
    model_architecture: Optional[str] = 'MULTILINGUAL_BERT_BASE_CASED',
    train_batch_size: Optional[int] = 32,
    eval_batch_size: Optional[int] = 32,
    num_epochs: Optional[int] = 30,
    dropout: Optional[float] = 0.35,
    initial_learning_rate: Optional[float] = 1e-4,
    warmup: Optional[float] = 0.1,
    optimizer_type: Optional[str] = 'lamb',
    random_seed: Optional[int] = 0,
    train_steps_per_epoch: Optional[int] = -1,
    validation_steps_per_epoch: Optional[int] = -1,
    test_steps_per_epoch: Optional[int] = -1,
):
  # fmt: off
  """Launch Vertex CustomJob to train a new Cloud natural language TFHub model.

  Args:
      project: GCP project to run CustomJob.
      location: Location of the job. Defaults to `us-central1`.
      input_data_path: GCS path to the file where data is stored. Should contain train and validation splits.

          For details on how to format the data, see
          https://cloud.google.com/vertex-ai/docs/text-data/classification/prepare-data.
      input_format: Input data format; supports "csv" and "jsonl". Defaults to "jsonl".

          For details on how to format the data, see
          https://cloud.google.com/vertex-ai/docs/text-data/classification/prepare-data.
      machine_type: Type of machine for running the model training on dedicated resources. Defaults to `n1-highmem-8.`

          For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      accelerator_type: Type of accelerator. Defaults to `NVIDIA_TESLA_T4`.

          For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec.
      accelerator_count: Number of accelerators. Defaults to `1`.

          For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec.
      natural_language_task_type: Task type to train the model on.
          Possible values are ["CLASSIFICATION", "MULTILABEL_CLASSIFICATION", "BINARY_CLASSIFICATION"]. Defaults to `CLASSIFICATION`.
      model_architecture: Model you wish to train. Possible values are ["MULTILINGUAL_BERT_BASE_CASED", "ENGLISH_BERT_BASE_UNCASED"]. Defaults to `MULTILINGUAL_BERT_BASE_CASED`.
      train_batch_size: Batch size to use during training.
          Note that increasing this when running on GPUs could cause training failures due to insufficient memory. Defaults to `32`.
      eval_batch_size: Batch size to use during eval.
          Note that increasing this when running on GPUs could cause training failures due to insufficient memory. Defaults to `32`.
      num_epochs: Max number of epochs to train for.
          Early stopping may prevent this number from being reached, but the learning rate schedule is influence by this value regardless. Defaults to `30`.
      dropout: Dropout rate for dropout layer(s). Defaults to `0.35`.
      initial_learning_rate: Starting learning rate for the model. Defaults to `1e-4`.
      warmup: Fraction of training steps that we should slowly raise the learning rate for (i.e. first 10 percent of training steps).
          Note that the "lamb" optimizer will ignore this value as it has built-in warmup. Defaults to `0.1`.
      optimizer_type: Type of optimizer to use. Options are ["lamb", "adamw"]. Defaults to `lamb`.
      random_seed: Random seed to use during training; maintain value for reproducibility. Defaults to `0`.
      train_steps_per_epoch: Training steps per epoch during diintibuted training.
          Only used when there are multiple GPUs. Defaults to `-1`.
      validation_steps_per_epoch: Validation steps per epoch during distributed training.
          Only used when there are multiple GPUs. Defaults to `-1`.
      test_steps_per_epoch: Test steps per epoch during distributed training.
          Only used when there are multiple GPUs. Defaults to `-1`.
  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the training job.
          For more details on GCP resources proto, see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components.google_cloud_pipeline_components/proto/README.md.
      model_output: Artifact tracking the batch prediction job output. This is only available if
          gcs_destination_output_uri_prefix is specified.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--payload',
          dsl.ConcatPlaceholder(
              items=[
                  (
                      '{"display_name":'
                      ' "train-tfhub-model-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}",'
                      ' "job_spec": {"worker_pool_specs": [{"replica_count":"'
                  ),
                  '1',
                  '", "machine_spec": {',
                  '"machine_type": "',
                  machine_type,
                  '"',
                  ', "accelerator_type": "',
                  accelerator_type,
                  '"',
                  ', "accelerator_count": ',
                  accelerator_count,
                  '}',
                  ', "container_spec": {"image_uri":"',
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-language/tfhub_model_trainer_image_gpu:latest',
                  '", "args": [ "--input_data_path=',
                  input_data_path,
                  '", "--input_format=',
                  input_format,
                  '", "--natural_language_task_type=',
                  natural_language_task_type,
                  '", "--model_architecture=',
                  model_architecture,
                  '", "--train_batch_size=',
                  train_batch_size,
                  '", "--eval_batch_size=',
                  eval_batch_size,
                  '", "--num_epochs=',
                  num_epochs,
                  '", "--dropout=',
                  dropout,
                  '", "--initial_learning_rate=',
                  initial_learning_rate,
                  '", "--warmup=',
                  warmup,
                  '", "--optimizer_type=',
                  optimizer_type,
                  '", "--random_seed=',
                  random_seed,
                  '", "--train_steps_per_epoch=',
                  train_steps_per_epoch,
                  '", "--validation_steps_per_epoch=',
                  validation_steps_per_epoch,
                  '", "--test_steps_per_epoch=',
                  test_steps_per_epoch,
                  '", "--model_output_path=',
                  model_output.path,
                  '" ]}}]}}',
              ]
          ),
      ],
  )
