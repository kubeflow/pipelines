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

"""AutoML Wide and Deep Trainer component spec."""

from typing import Optional

from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def wide_and_deep_trainer(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    learning_rate: float,
    dnn_learning_rate: float,
    instance_baseline: Input[Artifact],
    metadata: Input[Artifact],
    materialized_train_split: Input[Artifact],
    materialized_eval_split: Input[Artifact],
    transform_output: Input[Artifact],
    training_schema_uri: Input[Artifact],
    gcp_resources: dsl.OutputPath(str),
    unmanaged_container_model: Output[UnmanagedContainerModel],  # pylint: disable=unused-argument
    weight_column: Optional[str] = '',
    max_steps: Optional[int] = -1,
    max_train_secs: Optional[int] = -1,
    optimizer_type: Optional[str] = 'adam',
    l1_regularization_strength: Optional[float] = 0,
    l2_regularization_strength: Optional[float] = 0,
    l2_shrinkage_regularization_strength: Optional[float] = 0,
    beta_1: Optional[float] = 0.9,
    beta_2: Optional[float] = 0.999,
    hidden_units: Optional[str] = '30,30,30',
    use_wide: Optional[bool] = True,
    embed_categories: Optional[bool] = True,
    dnn_dropout: Optional[float] = 0,
    dnn_optimizer_type: Optional[str] = 'ftrl',
    dnn_l1_regularization_strength: Optional[float] = 0,
    dnn_l2_regularization_strength: Optional[float] = 0,
    dnn_l2_shrinkage_regularization_strength: Optional[float] = 0,
    dnn_beta_1: Optional[float] = 0.9,
    dnn_beta_2: Optional[float] = 0.999,
    enable_profiler: Optional[bool] = False,
    cache_data: Optional[str] = 'auto',
    seed: Optional[int] = 1,
    eval_steps: Optional[int] = 0,
    batch_size: Optional[int] = 100,
    measurement_selection_type: Optional[str] = 'BEST_MEASUREMENT',
    optimization_metric: Optional[str] = '',
    eval_frequency_secs: Optional[int] = 600,
    training_machine_spec: Optional[dict] = {'machine_type': 'c2-standard-16'},
    training_disk_spec: Optional[dict] = {
        'boot_disk_type': 'pd-ssd',
        'boot_disk_size_gb': 100,
    },
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Trains a Wide & Deep model using Vertex CustomJob API.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region that runs the pipeline components.
      root_dir: The root GCS directory for the pipeline components.
      target_column: The target column name.
      prediction_type: The type of prediction the model is to
        produce. "classification" or "regression".
      weight_column: The weight column name.
      max_steps: Number of steps to run the trainer for.
      max_train_secs: Amount of time in seconds to run the
        trainer for.
      learning_rate: The learning rate used by the linear optimizer.
      optimizer_type: The type of optimizer to use. Choices are
        "adam", "ftrl" and "sgd" for the Adam, FTRL, and Gradient Descent
        Optimizers, respectively.
      l1_regularization_strength: L1 regularization strength
        for optimizer_type="ftrl".
      l2_regularization_strength: L2 regularization strength
        for optimizer_type="ftrl"
      l2_shrinkage_regularization_strength: L2 shrinkage
        regularization strength for optimizer_type="ftrl".
      beta_1: Beta 1 value for optimizer_type="adam".
      beta_2: Beta 2 value for optimizer_type="adam".
      hidden_units: Hidden layer sizes to use for DNN feature
        columns, provided in comma-separated layers.
      use_wide: If set to true, the categorical columns will be
        used in the wide part of the DNN model.
      embed_categories: If set to true, the categorical columns
        will be used embedded and used in the deep part of the model. Embedding
        size is the square root of the column cardinality.
      dnn_dropout: The probability we will drop out a given
        coordinate.
      dnn_learning_rate: The learning rate for training the
        deep part of the model.
      dnn_optimizer_type: The type of optimizer to use for the
        deep part of the model. Choices are "adam", "ftrl" and "sgd". for the
        Adam, FTRL, and Gradient Descent Optimizers, respectively.
      dnn_l1_regularization_strength: L1 regularization
        strength for dnn_optimizer_type="ftrl".
      dnn_l2_regularization_strength: L2 regularization
        strength for dnn_optimizer_type="ftrl".
      dnn_l2_shrinkage_regularization_strength: L2 shrinkage
        regularization strength for dnn_optimizer_type="ftrl".
      dnn_beta_1: Beta 1 value for dnn_optimizer_type="adam".
      dnn_beta_2: Beta 2 value for dnn_optimizer_type="adam".
      enable_profiler: Enables profiling and saves a trace
        during evaluation.
      cache_data: Whether to cache data or not. If set to
        'auto', caching is determined based on the dataset size.
      seed: Seed to be used for this run.
      eval_steps: Number of steps to run evaluation for. If not
        specified or negative, it means run evaluation on the whole validation
        dataset. If set to 0, it means run evaluation for a fixed number of
        samples.
      batch_size: Batch size for training.
      measurement_selection_type: Which measurement to use
        if/when the service automatically selects the final measurement from
        previously reported intermediate measurements. One of "BEST_MEASUREMENT"
        or "LAST_MEASUREMENT".
      optimization_metric: Optimization metric used for
        `measurement_selection_type`. Default is "rmse" for regression and "auc"
        for classification.
      eval_frequency_secs: Frequency at which evaluation and
        checkpointing will take place.
      training_machine_spec: The training machine
        spec. See https://cloud.google.com/compute/docs/machine-types for
        options.
      training_disk_spec: The training disk spec.
      instance_baseline: The path to a JSON file for baseline values.
      metadata: Amount of time in seconds to run the trainer for.
      materialized_train_split: The path to the materialized train split.
      materialized_eval_split: The path to the materialized validation split.
      transform_output: The path to transform output.
      training_schema_uri: The path to the training schema.
      encryption_spec_key_name: The KMS key name.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the custom training job.
      unmanaged_container_model: The UnmanagedContainerModel artifact.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.44',
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
                      f' "wide-and-deep-trainer-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  '"}, "job_spec": {"worker_pool_specs": [{"replica_count":"',
                  '1',
                  '", "machine_spec": ',
                  training_machine_spec,
                  ', "disk_spec": ',
                  training_disk_spec,
                  ', "container_spec": {"image_uri":"',
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/wide-and-deep-training:20230619_1325',
                  '", "args": ["--target_column=',
                  target_column,
                  '", "--weight_column=',
                  weight_column,
                  '", "--model_type=',
                  prediction_type,
                  '", "--prediction_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:20230619_1325',
                  '", "--baseline_path=',
                  instance_baseline.uri,
                  '", "--metadata_path=',
                  metadata.uri,
                  '", "--transform_output_path=',
                  transform_output.uri,
                  '", "--training_schema_path=',
                  training_schema_uri.uri,
                  '", "--job_dir=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/train",'
                      ' "--training_data_path='
                  ),
                  materialized_train_split.uri,
                  '", "--validation_data_path=',
                  materialized_eval_split.uri,
                  '", "--max_steps=',
                  max_steps,
                  '", "--max_train_secs=',
                  max_train_secs,
                  '", "--learning_rate=',
                  learning_rate,
                  '", "--optimizer_type=',
                  optimizer_type,
                  '", "--l1_regularization_strength=',
                  l1_regularization_strength,
                  '", "--l2_regularization_strength=',
                  l2_regularization_strength,
                  '", "--l2_shrinkage_regularization_strength=',
                  l2_shrinkage_regularization_strength,
                  '", "--beta_1=',
                  beta_1,
                  '", "--beta_2=',
                  beta_2,
                  '", "--hidden_units=',
                  hidden_units,
                  '", "--use_wide=',
                  use_wide,
                  '", "--embed_categories=',
                  embed_categories,
                  '", "--dnn_dropout=',
                  dnn_dropout,
                  '", "--dnn_learning_rate=',
                  dnn_learning_rate,
                  '", "--dnn_optimizer_type=',
                  dnn_optimizer_type,
                  '", "--dnn_l1_regularization_strength=',
                  dnn_l1_regularization_strength,
                  '", "--dnn_l2_regularization_strength=',
                  dnn_l2_regularization_strength,
                  '", "--dnn_l2_shrinkage_regularization_strength=',
                  dnn_l2_shrinkage_regularization_strength,
                  '", "--dnn_beta_1=',
                  dnn_beta_1,
                  '", "--dnn_beta_2=',
                  dnn_beta_2,
                  '", "--enable_profiler=',
                  enable_profiler,
                  '", "--cache_data=',
                  cache_data,
                  '", "--seed=',
                  seed,
                  '", "--eval_steps=',
                  eval_steps,
                  '", "--batch_size=',
                  batch_size,
                  '", "--measurement_selection_type=',
                  measurement_selection_type,
                  '", "--optimization_metric=',
                  optimization_metric,
                  '", "--eval_frequency_secs=',
                  eval_frequency_secs,
                  '", "--executor_input={{$.json_escape[1]}}"]}}]}}',
              ]
          ),
      ],
  )
