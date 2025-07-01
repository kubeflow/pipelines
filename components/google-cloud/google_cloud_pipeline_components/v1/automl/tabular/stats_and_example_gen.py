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

"""AutoML Stats and Example Generation component spec."""

from typing import Optional

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Output


@dsl.container_component
def tabular_stats_and_example_gen(
    project: str,
    location: str,
    root_dir: str,
    target_column_name: str,
    prediction_type: str,
    transformations: str,
    dataset_schema: Output[Artifact],
    dataset_stats: Output[Artifact],
    train_split: Output[Dataset],
    eval_split: Output[Dataset],
    test_split: Output[Dataset],
    test_split_json: dsl.OutputPath(list),
    downsampled_test_split_json: dsl.OutputPath(list),
    instance_baseline: Output[Artifact],
    metadata: Output[Artifact],
    gcp_resources: dsl.OutputPath(str),
    weight_column_name: Optional[str] = '',
    optimization_objective: Optional[str] = '',
    optimization_objective_recall_value: Optional[float] = -1,
    optimization_objective_precision_value: Optional[float] = -1,
    transformations_path: Optional[str] = '',
    request_type: Optional[str] = 'COLUMN_STATS_ONLY',
    dataflow_machine_type: Optional[str] = 'n1-standard-16',
    dataflow_max_num_workers: Optional[int] = 25,
    dataflow_disk_size_gb: Optional[int] = 40,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    dataflow_service_account: Optional[str] = '',
    encryption_spec_key_name: Optional[str] = '',
    run_distillation: Optional[bool] = False,
    additional_experiments: Optional[str] = '',
    additional_experiments_json: Optional[dict] = {},
    data_source_csv_filenames: Optional[str] = '',
    data_source_bigquery_table_path: Optional[str] = '',
    predefined_split_key: Optional[str] = '',
    timestamp_split_key: Optional[str] = '',
    stratified_split_key: Optional[str] = '',
    training_fraction: Optional[float] = -1,
    validation_fraction: Optional[float] = -1,
    test_fraction: Optional[float] = -1,
    quantiles: Optional[list] = [],
    enable_probabilistic_inference: Optional[bool] = False,
):
  # fmt: off
  """Generates stats and training instances for tabular data.

  Args:
      project: Project to run dataset statistics and example generation.
      location: Location for running dataset statistics and example generation.
      root_dir: The Cloud Storage location to store the output.
      target_column_name: The target column name.
      weight_column_name: The weight column name.
      prediction_type: The prediction type. Supported values: "classification", "regression".
      optimization_objective: Objective function the model is optimizing towards. The training process creates a model that maximizes/minimizes the value of the objective function over the validation set. The supported optimization objectives depend on the prediction type. If the field is not set, a default objective function is used.  classification: "maximize-au-roc" (default) - Maximize the area under the receiver operating characteristic (ROC) curve.  "minimize-log-loss" - Minimize log loss. "maximize-au-prc" - Maximize the area under the precision-recall curve.  "maximize-precision-at-recall" - Maximize precision for a specified recall value. "maximize-recall-at-precision" - Maximize recall for a specified precision value.  classification (multi-class): "minimize-log-loss" (default) - Minimize log loss.  regression: "minimize-rmse" (default) - Minimize root-mean-squared error (RMSE). "minimize-mae" - Minimize mean-absolute error (MAE).  "minimize-rmsle" - Minimize root-mean-squared log error (RMSLE).
      optimization_objective_recall_value: Required when optimization_objective is "maximize-precision-at-recall". Must be between 0 and 1, inclusive.
      optimization_objective_precision_value: Required when optimization_objective is "maximize-recall-at-precision". Must be between 0 and 1, inclusive.
      transformations: Quote escaped JSON string for transformations. Each transformation will apply transform function to given input column. And the result will be used for training. When creating transformation for BigQuery Struct column, the column should be flattened using "." as the delimiter.
      transformations_path: Path to a GCS file containing JSON string for transformations.
      dataflow_machine_type: The machine type used for dataflow jobs. If not set, default to n1-standard-16.
      dataflow_max_num_workers: The number of workers to run the dataflow job. If not set, default to 25.
      dataflow_disk_size_gb: The disk size, in gigabytes, to use on each Dataflow worker instance. If not set, default to 40.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. More details: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
      dataflow_service_account: Custom service account to run dataflow jobs.
      encryption_spec_key_name: Customer-managed encryption key.
      run_distillation: True if in distillation mode. The default value is false.

  Returns:
      dataset_schema: The schema of the dataset.
      dataset_stats: The stats of the dataset.
      train_split: The train split.
      eval_split: The eval split.
      test_split: The test split.
      test_split_json: The test split JSON object.
      downsampled_test_split_json: The downsampled test split JSON object.
      instance_baseline: The instance baseline used to calculate explanations.
      metadata: The tabular example gen metadata.
      gcp_resources: GCP resources created by this component. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
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
                      f' "tabular-stats-and-example-gen-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20250620_0525',
                  '", "args": ["stats_generator",',
                  '"--train_spec={\\"prediction_type\\": \\"',
                  prediction_type,
                  '\\", \\"target_column\\": \\"',
                  target_column_name,
                  '\\", \\"optimization_objective\\": \\"',
                  optimization_objective,
                  '\\", \\"weight_column_name\\": \\"',
                  weight_column_name,
                  '\\", \\"transformations\\": ',
                  transformations,
                  ', \\"quantiles\\": ',
                  quantiles,
                  ', \\"enable_probabilistic_inference\\": ',
                  enable_probabilistic_inference,
                  '}", "--transformations_override_path=',
                  transformations_path,
                  '", "--data_source_csv_filenames=',
                  data_source_csv_filenames,
                  '", "--data_source_bigquery_table_path=',
                  data_source_bigquery_table_path,
                  '", "--predefined_split_key=',
                  predefined_split_key,
                  '", "--timestamp_split_key=',
                  timestamp_split_key,
                  '", "--stratified_split_key=',
                  stratified_split_key,
                  '", "--training_fraction=',
                  training_fraction,
                  '", "--validation_fraction=',
                  validation_fraction,
                  '", "--test_fraction=',
                  test_fraction,
                  '", "--target_column=',
                  target_column_name,
                  '", "--request_type=',
                  request_type,
                  '", "--optimization_objective_recall_value=',
                  optimization_objective_recall_value,
                  '", "--optimization_objective_precision_value=',
                  optimization_objective_precision_value,
                  '", "--example_gen_gcs_output_prefix=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/example_gen_output",'
                      ' "--dataset_stats_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/stats/",'
                      ' "--stats_result_path='
                  ),
                  dataset_stats.uri,
                  '", "--dataset_schema_path=',
                  dataset_schema.uri,
                  (
                      f'", "--job_name=tabular-stats-and-example-gen-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}'
                  ),
                  '", "--dataflow_project=',
                  project,
                  '", "--error_file_path=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.pb",'
                      ' "--dataflow_staging_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_staging",'
                      ' "--dataflow_tmp_dir='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/dataflow_tmp",'
                      ' "--dataflow_max_num_workers='
                  ),
                  dataflow_max_num_workers,
                  '", "--dataflow_worker_container_image=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/dataflow-worker:20250620_0525',
                  '", "--dataflow_machine_type=',
                  dataflow_machine_type,
                  '", "--dataflow_disk_size_gb=',
                  dataflow_disk_size_gb,
                  '", "--dataflow_kms_key=',
                  encryption_spec_key_name,
                  '", "--dataflow_subnetwork_fully_qualified=',
                  dataflow_subnetwork,
                  '", "--dataflow_use_public_ips=',
                  dataflow_use_public_ips,
                  '", "--dataflow_service_account=',
                  dataflow_service_account,
                  '", "--is_distill=',
                  run_distillation,
                  '", "--additional_experiments=',
                  additional_experiments,
                  '", "--metadata_path=',
                  metadata.uri,
                  '", "--train_split=',
                  train_split.uri,
                  '", "--eval_split=',
                  eval_split.uri,
                  '", "--test_split=',
                  test_split.uri,
                  '", "--test_split_for_batch_prediction_component=',
                  test_split_json,
                  (
                      '", "--downsampled_test_split_for_batch_prediction_component='
                  ),
                  downsampled_test_split_json,
                  '", "--instance_baseline_path=',
                  instance_baseline.uri,
                  '", "--lro_job_info=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/lro",'
                      ' "--gcp_resources_path='
                  ),
                  gcp_resources,
                  (
                      '", "--parse_json=true",'
                      ' "--generate_additional_downsample_test_split=true",'
                      ' "--executor_input={{$.json_escape[1]}}"]}}]}}'
                  ),
              ]
          ),
      ],
  )
