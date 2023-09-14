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

from typing import List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model_evaluation import version
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import ForecastingMetrics
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import container_component


@container_component
def model_evaluation_forecasting(
    gcp_resources: dsl.OutputPath(str),
    evaluation_metrics: dsl.Output[ForecastingMetrics],
    target_field_name: str,
    model: dsl.Input[VertexModel] = None,
    location: str = 'us-central1',
    predictions_format: str = 'jsonl',
    predictions_gcs_source: dsl.Input[dsl.Artifact] = None,
    predictions_bigquery_source: dsl.Input[BQTable] = None,
    ground_truth_format: str = 'jsonl',
    ground_truth_gcs_source: List[str] = [],
    ground_truth_bigquery_source: str = '',
    forecasting_type: str = 'point',
    forecasting_quantiles: List[float] = [],
    point_evaluation_quantile: float = 0.5,
    prediction_score_column: str = 'prediction.value',
    dataflow_service_account: str = '',
    dataflow_disk_size_gb: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Computes a `google.ForecastingMetrics` Artifact, containing evaluation
  metrics given a model's prediction results.

  Creates a Dataflow job with Apache Beam and TFMA to compute evaluation
  metrics.
  Supports point forecasting and quantile forecasting for tabular data.

  Args:
      location: Location for running the evaluation.
      predictions_format: The file format for the batch prediction results. `jsonl`, `csv`, and `bigquery` are the allowed formats, from Vertex Batch Prediction.
      predictions_gcs_source: An artifact with its URI pointing toward a GCS directory with prediction or explanation files to be used for this evaluation. For prediction results, the files should be named "prediction.results-*". For explanation results, the files should be named "explanation.results-*".
      predictions_bigquery_source: BigQuery table with prediction or explanation data to be used for this evaluation. For prediction results, the table column should be named "predicted_*".
      ground_truth_format: Required for custom tabular and non tabular data. The file format for the ground truth files. `jsonl`, `csv`, and `bigquery` are the allowed formats.
      ground_truth_gcs_source: Required for custom tabular and non tabular data. The GCS URIs representing where the ground truth is located. Used to provide ground truth for each prediction instance when they are not part of the batch prediction jobs prediction instance.
      ground_truth_bigquery_source: Required for custom tabular. The BigQuery table URI representing where the ground truth is located. Used to provide ground truth for each prediction instance when they are not part of the batch prediction jobs prediction instance.
      forecasting_type: The forecasting type being addressed by this evaluation run. `point` and `quantile` are the supported types.
      forecasting_quantiles: Required for a `quantile` forecasting_type. The list of quantiles in the same order appeared in the quantile prediction score column.
      point_evaluation_quantile: Required for a `quantile` forecasting_type. A quantile in the list of forecasting_quantiles that will be used for point evaluation metrics.
      target_field_name: The full name path of the features target field in the predictions file. Formatted to be able to find nested columns, delimited by `.`. Alternatively referred to as the ground truth (or ground_truth_column) field.
      model: The Vertex model used for evaluation. Must be located in the same region as the location argument. It is used to set the default configurations for AutoML and custom-trained models.
      prediction_score_column: The column name of the field containing batch prediction scores. Formatted to be able to find nested columns, delimited by `.`.
      dataflow_service_account: Service account to run the Dataflow job. If not set, Dataflow will use the default worker service account.  For more details, see https://cloud.google.com/dataflow/docs/concepts/secURIty-and-permissions#default_worker_service_account
      dataflow_disk_size_gb: The disk size (in GB) of the machine executing the evaluation run.
      dataflow_machine_type: The machine type executing the evaluation run.
      dataflow_workers_num: The number of workers executing the evaluation run.
      dataflow_max_workers_num: The max number of workers executing the evaluation run.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty the default subnetwork will be used. More details: https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP addresses.
      encryption_spec_key_name:  Customer-managed encryption key options. If set, resources created by this pipeline will be encrypted with the provided encryption key. Has the form: `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created.
      force_runner_mode: Flag to choose Beam runner. Valid options are `DirectRunner` and `Dataflow`.
      project: Project to run evaluation container. Defaults to the project in which the PipelineJob is run.


  Returns:
      evaluation_metrics: `google.ForecastingMetrics` representing the forecasting evaluation metrics in GCS.
      gcp_resources: Serialized gcp_resources proto tracking the Dataflow job. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: off
  return dsl.ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
      command=[
          'python3',
          '/main.py',
      ],
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
          '--target_field_name',
          dsl.ConcatPlaceholder(['instance.', target_field_name]),
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
                  dsl.ConcatPlaceholder([
                      'bq://',
                      predictions_bigquery_source.metadata['projectId'],
                      '.',
                      predictions_bigquery_source.metadata['datasetId'],
                      '.',
                      predictions_bigquery_source.metadata['tableId'],
                  ]),
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
          f'{dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--prediction_score_column',
          prediction_score_column,
          '--dataflow_job_prefix',
          f'evaluation-forecasting-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_disk_size',
          dataflow_disk_size_gb,
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
          '--force_runner_mode',
          force_runner_mode,
          '--output_metrics_gcs_path',
          evaluation_metrics.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
