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

from google_cloud_pipeline_components.types.artifact_types import _ForecastingMetrics
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def model_evaluation_forecasting(
    project: str,
    root_dir: str,
    target_field_name: str,
    evaluation_metrics: Output[_ForecastingMetrics],
    gcp_resources: dsl.OutputPath(str),
    location: Optional[str] = 'us-central1',
    predictions_format: Optional[str] = 'jsonl',
    predictions_gcs_source: Optional[Input[Artifact]] = None,
    predictions_bigquery_source: Optional[Input[BQTable]] = None,
    ground_truth_format: Optional[str] = 'jsonl',
    ground_truth_gcs_source: Optional[list] = [],
    ground_truth_bigquery_source: Optional[str] = '',
    model: Optional[Input[VertexModel]] = None,
    prediction_score_column: Optional[str] = '',
    forecasting_type: Optional[str] = 'point',
    forecasting_quantiles: Optional[list] = [0.5],
    example_weight_column: Optional[str] = '',
    point_evaluation_quantile: Optional[float] = 0.5,
    dataflow_service_account: Optional[str] = '',
    dataflow_disk_size: Optional[int] = 50,
    dataflow_machine_type: Optional[str] = 'n1-standard-4',
    dataflow_workers_num: Optional[int] = 1,
    dataflow_max_workers_num: Optional[int] = 5,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Computes a google.ForecastingMetrics Artifact, containing evaluation
  metrics given a model's prediction results.

  Creates a dataflow job with Apache Beam and TFMA to compute evaluation
  metrics.
  Supports point forecasting and quantile forecasting for tabular data.

  Args:
      project: Project to run evaluation container.
      location: Location for running the evaluation. If not set,
        defaulted to `us-central1`.
      root_dir: The GCS directory for keeping staging files. A random
        subdirectory will be created under the directory to keep job info for
        resuming the job in case of failure.
      predictions_format: The file format for the batch
        prediction results. `jsonl` is currently the only allowed format. If not
        set, defaulted to `jsonl`.
      predictions_gcs_source: An artifact with its
        URI pointing toward a GCS directory with prediction or explanation files
        to be used for this evaluation. For prediction results, the files should
        be named "prediction.results-*". For explanation results, the files
        should be named "explanation.results-*".
      predictions_bigquery_source: BigQuery table
        with prediction or explanation data to be used for this evaluation. For
        prediction results, the table column should be named "predicted_*".
      ground_truth_format: Required for custom tabular and non
        tabular data. The file format for the ground truth files. `jsonl` is
        currently the only allowed format. If not set, defaulted to `jsonl`.
      ground_truth_gcs_source: Required for custom
        tabular and non tabular data. The GCS uris representing where the ground
        truth is located. Used to provide ground truth for each prediction
        instance when they are not part of the batch prediction jobs prediction
        instance.
      ground_truth_bigquery_source: Required for
        custom tabular. The BigQuery table uri representing where the ground
        truth is located. Used to provide ground truth for each prediction
        instance when they are not part of the batch prediction jobs prediction
        instance.
      target_field_name: The full name path of the features target field
        in the predictions file. Formatted to be able to find nested columns,
        delimited by `.`. Alternatively referred to as the ground truth (or
        ground_truth_column) field.
      model: The Model used for predictions job.
        Must share the same ancestor Location.
      prediction_score_column: The column name of the
        field containing batch prediction scores. Formatted to be able to find
        nested columns, delimited by `.`. If not set, defaulted to
        `prediction.value` for a `point` forecasting_type and
        `prediction.quantile_predictions` for a `quantile` forecasting_type.
      forecasting_type: If the problem_type is
        `forecasting`, then the forecasting type being addressed by this
        regression evaluation run. `point` and `quantile` are the supported
        types. If not set, defaulted to `point`.
      forecasting_quantiles: Required for a
        `quantile` forecasting_type. The list of quantiles in the same order
        appeared in the quantile prediction score column. If one of the
        quantiles is set to `0.5f`, point evaluation will be set on that index.
      example_weight_column: The column name of the
        field containing example weights. Each value of positive_classes
        provided.
      point_evaluation_quantile: Required for a `quantile`
        forecasting_type. A quantile in the list of forecasting_quantiles that
        will be used for point evaluation metrics.
      dataflow_service_account: Service account to run
        the dataflow job. If not set, dataflow will use the default woker
        service account. For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size: The disk size (in GB) of the
        machine executing the evaluation run. If not set, defaulted to `50`.
      dataflow_machine_type: The machine type
        executing the evaluation run. If not set, defaulted to `n1-standard-4`.
      dataflow_workers_num: The number of workers
        executing the evaluation run. If not set, defaulted to `10`.
      dataflow_max_workers_num: The max number of
        workers executing the evaluation run. If not set, defaulted to `25`.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.
      encryption_spec_key_name: Customer-managed encryption key.

  Returns:
      evaluation_metrics: google.ForecastingMetrics artifact representing the forecasting
          evaluation metrics in GCS.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/model-evaluation:v0.9',
      command=['python', '/main.py'],
      args=[
          '--setup_file',
          '/setup.py',
          '--json_mode',
          'true',
          '--project_id',
          project,
          '--location',
          location,
          '--problem_type',
          'forecasting',
          '--forecasting_type',
          forecasting_type,
          '--forecasting_quantiles',
          forecasting_quantiles,
          '--point_evaluation_quantile',
          point_evaluation_quantile,
          '--batch_prediction_format',
          predictions_format,
          dsl.IfPresentPlaceholder(
              input_name='predictions_gcs_source',
              then=[
                  '--batch_prediction_gcs_source',
                  predictions_gcs_source.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='predictions_bigquery_source',
              then=[
                  '--batch_prediction_bigquery_source',
                  f"bq://{predictions_bigquery_source.metadata['projectId']}.{predictions_bigquery_source.metadata['datasetId']}.{predictions_bigquery_source.metadata['tableId']}",
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model',
              then=[
                  '--model_name',
                  model.metadata['resourceName'],
              ],
          ),
          '--ground_truth_format',
          ground_truth_format,
          '--ground_truth_gcs_source',
          ground_truth_gcs_source,
          '--ground_truth_bigquery_source',
          ground_truth_bigquery_source,
          '--root_dir',
          f'{root_dir}/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--target_field_name',
          f'instance.{target_field_name}',
          '--prediction_score_column',
          prediction_score_column,
          '--dataflow_job_prefix',
          f'evaluation-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_disk_size',
          dataflow_disk_size,
          '--dataflow_machine_type',
          dataflow_machine_type,
          '--dataflow_workers_num',
          dataflow_workers_num,
          '--dataflow_max_workers_num',
          dataflow_max_workers_num,
          '--dataflow_subnetwork',
          dataflow_subnetwork,
          '--dataflow_use_public_ips',
          dataflow_use_public_ips,
          '--kms_key_name',
          encryption_spec_key_name,
          '--output_metrics_gcs_path',
          evaluation_metrics.uri,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
