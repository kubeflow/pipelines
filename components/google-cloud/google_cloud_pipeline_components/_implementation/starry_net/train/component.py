# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Container Component for training STARRY-Net."""

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils
from google_cloud_pipeline_components._implementation.starry_net import version

from kfp import dsl


@dsl.container_component
def train(
    gcp_resources: dsl.OutputPath(str),
    trainer_dir: dsl.Output[dsl.Artifact],  # pytype: disable=unsupported-operands
    num_epochs: int,
    backcast_length: int,
    forecast_length: int,
    train_end_date: str,
    csv_data_path: str,
    csv_static_covariates_path: str,
    static_covariates_vocab_path: str,
    train_tf_record_patterns: str,
    val_tf_record_patterns: str,
    test_tf_record_patterns: str,
    n_decomposition_plots: int,
    n_val_windows: int,
    n_test_windows: int,
    test_set_stride: int,
    nan_threshold: float,
    zero_threshold: float,
    cleaning_activation_regularizer_coeff: float,
    change_point_activation_regularizer_coeff: float,
    change_point_output_regularizer_coeff: float,
    alpha_upper_bound: float,
    beta_upper_bound: float,
    phi_lower_bound: float,
    b_fixed_val: int,
    b0_fixed_val: int,
    phi_fixed_val: int,
    quantiles: str,
    use_static_covariates: bool,
    static_covariate_names: str,
    model_blocks: str,
    freeze_point_forecasts: bool,
    machine_type: str,
    accelerator_type: str,
    docker_region: str,
    location: str,
    job_id: str,
    project: str,
    encryption_spec_key_name: str,
):
  # fmt: off
  """Trains a STARRY-Net model.

  Args:
    gcp_resources: Serialized JSON of ``gcp_resources`` which tracks the
      CustomJob.
    trainer_dir: The gcp bucket path where training artifacts are saved.
    num_epochs: The number of epochs to train for.
    backcast_length: The length of the input window to feed into the model.
    forecast_length: The length of the forecast horizon.
    train_end_date: The last date of data to use in the training set. All
      subsequent dates are part of the test set.
    csv_data_path: The path to the training data csv.
    csv_static_covariates_path: The path to the static covariates csv.
    static_covariates_vocab_path: The path to the master static covariates vocab
      json.
    train_tf_record_patterns: The glob patterns to the tf records to use for
      training.
    val_tf_record_patterns: The glob patterns to the tf records to use for
      validation.
    test_tf_record_patterns: The glob patterns to the tf records to use for
      testing.
    n_decomposition_plots: How many decomposition plots to save to tensorboard.
    n_val_windows: The number of windows to use for the val set. If 0, no
      validation set is used.
    n_test_windows: The number of windows to use for the test set. Must be >= 1.
    test_set_stride: The number of timestamps to roll forward when
      constructing the val and test sets.
    nan_threshold: Series having more nan / missing values than
      nan_threshold (inclusive) in percentage for either backtest or forecast
      will not be sampled in the training set (including missing due to
      train_start and train_end). All existing nans are replaced by zeros.
    zero_threshold: Series having more 0.0 values than zero_threshold
      (inclusive) in percentage for either backtest or forecast will not be
      sampled in the training set.
    cleaning_activation_regularizer_coeff: The regularization coefficient for
      the cleaning param estimator's final layer's activation in the cleaning
      block.
    change_point_activation_regularizer_coeff: The regularization coefficient
      for the change point param estimator's final layer's activation in the
      change_point block.
    change_point_output_regularizer_coeff: The regularization coefficient
      for the change point param estimator's output in the change_point block.
    alpha_upper_bound: The upper bound for data smooth parameter alpha in the
      trend block.
    beta_upper_bound: The upper bound for data smooth parameter beta in the
      trend block.
    phi_lower_bound: The lower bound for damping param phi in the trend block.
    b_fixed_val: The fixed value for b in the trend block. If set to anything
      other than -1, the trend block will not learn to provide estimates
      but use the fixed value directly.
    b0_fixed_val: The fixed value for b0 in the trend block. If set to
      anything other than -1, the trend block will not learn to provide
      estimates but use the fixed value directly.
    phi_fixed_val: The fixed value for phi in the trend block. If set to
      anything other than -1, the trend block will not learn to provide
      estimates but use the fixed value directly.
    quantiles: The stringified tuple of quantiles to learn in the quantile
      block, e.g., 0.5,0.9,0.95. This should always start with 0.5,
      representing the point forecasts.
    use_static_covariates: Whether to use static covariates.
    static_covariate_names: The stringified tuple of names of the static
      covariates.
    model_blocks: The stringified tuple of blocks to use in the order
      that they appear in the model. Possible values are `cleaning`,
      `change_point`, `trend`, `hour_of_week-hybrid`, `day_of_week-hybrid`,
      `day_of_year-hybrid`, `week_of_year-hybrid`, `month_of_year-hybrid`,
      `residual`, `quantile`.
    freeze_point_forecasts: Whether or not to do two pass training, where
      first the point forecast model is trained, then the quantile block is,
      added, all preceding blocks are frozen, and the quantile block is trained.
      This should always be True if quantiles != [0.5].
    machine_type: The machine type.
    accelerator_type: The accelerator type.
    docker_region: The docker region, used to determine which image to use.
    location: Location for creating the custom training job. If not set,
      defaults to us-central1.
    job_id: The pipeline job id.
    project: Project to create the custom training job in. Defaults to
      the project in which the PipelineJob is run.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    gcp_resources: Serialized JSON of ``gcp_resources`` which tracks the
      CustomJob.
    trainer_dir: The gcp bucket path where training artifacts are saved.
  """
  job_name = f'trainer-{job_id}'
  payload = {
      'display_name': job_name,
      'encryption_spec': {
          'kms_key_name': str(encryption_spec_key_name),
      },
      'job_spec': {
          'worker_pool_specs': [{
              'replica_count': '1',
              'machine_spec': {
                  'machine_type': str(machine_type),
                  'accelerator_type': str(accelerator_type),
                  'accelerator_count': 1,
              },
              'disk_spec': {
                  'boot_disk_type': 'pd-ssd',
                  'boot_disk_size_gb': 100,
              },
              'container_spec': {
                  'image_uri': f'{docker_region}-docker.pkg.dev/vertex-ai-restricted/starryn/trainer:{version.TRAINER_VERSION}',
                  'args': [
                      f'--vertex_experiment_dir={trainer_dir.path}',
                      f'--vertex_job_id={job_id}',
                      '--config=analysis/trafficforecast/starryn/experiments/configs/vertex.py',
                      f'--config.num_epochs={num_epochs}',
                      f'--config.freeze_point_forecasts={freeze_point_forecasts}',
                      f'--config.callbacks.tensorboard.n_decomposition_plots={n_decomposition_plots}',
                      f'--config.datasets.backcast_length={backcast_length}',
                      f'--config.datasets.forecast_length={forecast_length}',
                      f'--config.datasets.train_end_date={train_end_date}',
                      f'--config.datasets.train_path={csv_data_path}',
                      f'--config.datasets.static_covariates_path={csv_static_covariates_path}',
                      f'--config.datasets.static_covariates_vocab_path={static_covariates_vocab_path}',
                      f'--config.datasets.train_tf_record_patterns={train_tf_record_patterns}',
                      f'--config.datasets.val_tf_record_patterns={val_tf_record_patterns}',
                      f'--config.datasets.test_tf_record_patterns={test_tf_record_patterns}',
                      f'--config.datasets.n_val_windows={n_val_windows}',
                      f'--config.datasets.val_rolling_window_size={test_set_stride}',
                      f'--config.datasets.n_test_windows={n_test_windows}',
                      f'--config.datasets.test_rolling_window_size={test_set_stride}',
                      f'--config.datasets.nan_threshold={nan_threshold}',
                      f'--config.datasets.zero_threshold={zero_threshold}',
                      f'--config.model.regularizer_coeff={cleaning_activation_regularizer_coeff}',
                      f'--config.model.activation_regularizer_coeff={change_point_activation_regularizer_coeff}',
                      f'--config.model.output_regularizer_coeff={change_point_output_regularizer_coeff}',
                      f'--config.model.alpha_upper_bound={alpha_upper_bound}',
                      f'--config.model.beta_upper_bound={beta_upper_bound}',
                      f'--config.model.phi_lower_bound={phi_lower_bound}',
                      f'--config.model.b_fixed_val={b_fixed_val}',
                      f'--config.model.b0_fixed_val={b0_fixed_val}',
                      f'--config.model.phi_fixed_val={phi_fixed_val}',
                      f'--config.model.quantiles={quantiles}',
                      f'--config.model.use_static_covariates_trend={use_static_covariates}',
                      f'--config.model.use_static_covariates_calendar={use_static_covariates}',
                      f'--config.model.static_cov_names={static_covariate_names}',
                      f'--config.model.blocks_list={model_blocks}',
                  ],
              },
          }]
      }
    }
  return utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=payload,
      gcp_resources=gcp_resources,
    )
