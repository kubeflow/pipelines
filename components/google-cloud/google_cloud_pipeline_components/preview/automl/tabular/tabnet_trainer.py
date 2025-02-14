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

"""AutoML Tabnet Trainer component spec."""

from typing import Optional

from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def tabnet_trainer(
    project: str,
    location: str,
    root_dir: str,
    target_column: str,
    prediction_type: str,
    learning_rate: float,
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
    large_category_dim: Optional[int] = 1,
    large_category_thresh: Optional[int] = 300,
    yeo_johnson_transform: Optional[bool] = True,
    feature_dim: Optional[int] = 64,
    feature_dim_ratio: Optional[float] = 0.5,
    num_decision_steps: Optional[int] = 6,
    relaxation_factor: Optional[float] = 1.5,
    decay_every: Optional[float] = 100,
    decay_rate: Optional[float] = 0.95,
    gradient_thresh: Optional[float] = 2000,
    sparsity_loss_weight: Optional[float] = 1e-05,
    batch_momentum: Optional[float] = 0.95,
    batch_size_ratio: Optional[float] = 0.25,
    num_transformer_layers: Optional[int] = 4,
    num_transformer_layers_ratio: Optional[float] = 0.25,
    class_weight: Optional[float] = 1.0,
    loss_function_type: Optional[str] = 'default',
    alpha_focal_loss: Optional[float] = 0.25,
    gamma_focal_loss: Optional[float] = 2.0,
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
  """Trains a TabNet model using Vertex CustomJob API.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region that runs the pipeline components.
      root_dir: The root GCS directory for the pipeline components.
      target_column: The target column name.
      prediction_type: The type of prediction the model is to produce. "classification" or "regression".
      weight_column: The weight column name.
      max_steps: Number of steps to run the trainer for.
      max_train_secs: Amount of time in seconds to run the trainer for.
      learning_rate: The learning rate used by the linear optimizer.
      large_category_dim: Embedding dimension for categorical feature with large number of categories.
      large_category_thresh: Threshold for number of categories to apply large_category_dim embedding dimension to.
      yeo_johnson_transform: Enables trainable Yeo-Johnson power transform.
      feature_dim: Dimensionality of the hidden representation in feature transformation block.
      feature_dim_ratio: The ratio of output dimension (dimensionality of the outputs of each decision step) to feature dimension.
      num_decision_steps: Number of sequential decision steps.
      relaxation_factor: Relaxation factor that promotes the reuse of each feature at different decision steps. When it is 1, a feature is enforced to be used only at one decision step and as it increases, more flexibility is provided to use a feature at multiple decision steps.
      decay_every: Number of iterations for periodically applying learning rate decaying.
      decay_rate: Learning rate decaying.
      gradient_thresh: Threshold for the norm of gradients for clipping.
      sparsity_loss_weight: Weight of the loss for sparsity regularization (increasing it will yield more sparse feature selection).
      batch_momentum: Momentum in ghost batch normalization.
      batch_size_ratio: The ratio of virtual batch size (size of the ghost batch normalization) to batch size.
      num_transformer_layers: The number of transformer layers for each decision step. used only at one decision step and as it increases, more flexibility is provided to use a feature at multiple decision steps.
      num_transformer_layers_ratio: The ratio of shared transformer layer to transformer layers.
      class_weight: The class weight is used to computes a weighted cross entropy which is helpful in classify imbalanced dataset. Only used for classification.
      loss_function_type: Loss function type. Loss function in classification [cross_entropy, weighted_cross_entropy, focal_loss], default is cross_entropy. Loss function in regression: [rmse, mae, mse], default is mse.
      alpha_focal_loss: Alpha value (balancing factor) in focal_loss function. Only used for classification.
      gamma_focal_loss: Gamma value (modulating factor) for focal loss for focal loss. Only used for classification.
      enable_profiler: Enables profiling and saves a trace during evaluation.
      cache_data: Whether to cache data or not. If set to 'auto', caching is determined based on the dataset size.
      seed: Seed to be used for this run.
      eval_steps: Number of steps to run evaluation for. If not specified or negative, it means run evaluation on the whole validation dataset. If set to 0, it means run evaluation for a fixed number of samples.
      batch_size: Batch size for training.
      measurement_selection_type: Which measurement to use if/when the service automatically selects the final measurement from previously reported intermediate measurements. One of "BEST_MEASUREMENT" or "LAST_MEASUREMENT".
      optimization_metric: Optimization metric used for `measurement_selection_type`. Default is "rmse" for regression and "auc" for classification.
      eval_frequency_secs: Frequency at which evaluation and checkpointing will take place.
      training_machine_spec: The training machine spec. See https://cloud.google.com/compute/docs/machine-types for options.
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
                      f' "tabnet-trainer-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
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
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/tabnet-training:20240419_0625',
                  '", "args": ["--target_column=',
                  target_column,
                  '", "--weight_column=',
                  weight_column,
                  '", "--model_type=',
                  prediction_type,
                  '", "--prediction_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:20240419_0625',
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
                  '", "--large_category_dim=',
                  large_category_dim,
                  '", "--large_category_thresh=',
                  large_category_thresh,
                  '", "--yeo_johnson_transform=',
                  yeo_johnson_transform,
                  '", "--feature_dim=',
                  feature_dim,
                  '", "--feature_dim_ratio=',
                  feature_dim_ratio,
                  '", "--num_decision_steps=',
                  num_decision_steps,
                  '", "--relaxation_factor=',
                  relaxation_factor,
                  '", "--decay_every=',
                  decay_every,
                  '", "--decay_rate=',
                  decay_rate,
                  '", "--gradient_thresh=',
                  gradient_thresh,
                  '", "--sparsity_loss_weight=',
                  sparsity_loss_weight,
                  '", "--batch_momentum=',
                  batch_momentum,
                  '", "--batch_size_ratio=',
                  batch_size_ratio,
                  '", "--num_transformer_layers=',
                  num_transformer_layers,
                  '", "--num_transformer_layers_ratio=',
                  num_transformer_layers_ratio,
                  '", "--class_weight=',
                  class_weight,
                  '", "--loss_function_type=',
                  loss_function_type,
                  '", "--alpha_focal_loss=',
                  alpha_focal_loss,
                  '", "--gamma_focal_loss=',
                  gamma_focal_loss,
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
                  (
                      '", "--generate_feature_importance=true",'
                      ' "--executor_input={{$.json_escape[1]}}"]}}]}}'
                  ),
              ]
          ),
      ],
  )
