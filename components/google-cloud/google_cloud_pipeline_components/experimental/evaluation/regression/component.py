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

from google_cloud_pipeline_components.types.artifact_types import _RegressionMetrics
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def model_evaluation_regression(
    gcp_resources: OutputPath(str),
    evaluation_metrics: Output[_RegressionMetrics],
    project: str,
    target_field_name: str,
    model: Input[VertexModel] = None,
    location: str = 'us-central1',
    predictions_format: str = 'jsonl',
    predictions_gcs_source: Input[Artifact] = None,
    predictions_bigquery_source: Input[BQTable] = None,
    ground_truth_format: str = 'jsonl',
    ground_truth_gcs_source: list = [],
    ground_truth_bigquery_source: str = '',
    prediction_score_column: str = '',
    dataflow_service_account: str = '',
    dataflow_disk_size: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_direct_runner: bool = False,
):
  # fmt: off
  """Computes a google.RegressionMetrics Artifact, containing evaluation
  metrics given a model's prediction results.

  Creates a dataflow job with Apache Beam and TFMA to compute evaluation
  metrics.
  Supports regression for tabular data.

  Args:
      project: Project to run evaluation container.
      location: Location for running the evaluation. If not set,
        defaulted to `us-central1`.
      predictions_format: The file format for the batch
        prediction results. `jsonl`, `csv`, and `bigquery` are the allowed
        formats, from Vertex Batch Prediction. If not set, defaulted to `jsonl`.
      predictions_gcs_source: An artifact with its
        URI pointing toward a GCS directory with prediction or explanation files
        to be used for this evaluation. For prediction results, the files should
        be named "prediction.results-*". For explanation results, the files
        should be named "explanation.results-*".
      predictions_bigquery_source: BigQuery table
        with prediction or explanation data to be used for this evaluation. For
        prediction results, the table column should be named "predicted_*".
      ground_truth_format: Required for custom tabular and non
        tabular data. The file format for the ground truth files. `jsonl`,
        `csv`, and `bigquery` are the allowed formats. If not set, defaulted to
        `jsonl`.
      ground_truth_gcs_source: Required for custom
        tabular and non tabular data. The GCS uris representing where the ground
        truth is located. Used to provide ground truth for each prediction
        instance when they are not part of the batch prediction jobs prediction
        instance.
      ground_truth_bigquery_source: Required for custom tabular.
        The BigQuery table uri representing where the ground truth is located.
        Used to provide ground truth for each prediction instance when they are
        not part of the batch prediction jobs prediction instance.
      target_field_name: The full name path of the features target field
        in the predictions file. Formatted to be able to find nested columns,
        delimited by `.`. Alternatively referred to as the ground truth (or
        ground_truth_column) field.
      model: The managed Vertex Model used for
        predictions job, if using Vertex batch prediction. Must share the same
        location as the provided input argument `location`.
      prediction_score_column: The column name of the field
        containing batch prediction scores. Formatted to be able to find nested
        columns, delimited by `.`. If not set, defaulted to `prediction.scores`
        for classification.
      dataflow_service_account: Service account to run the
        dataflow job. If not set, dataflow will use the default worker service
        account. For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size: The disk size (in GB) of the machine
        executing the evaluation run. If not set, defaulted to `50`.
      dataflow_machine_type: The machine type executing the
        evaluation run. If not set, defaulted to `n1-standard-4`.
      dataflow_workers_num: The number of workers executing the
        evaluation run. If not set, defaulted to `10`.
      dataflow_max_workers_num: The max number of workers
        executing the evaluation run. If not set, defaulted to `25`.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More
        details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.
      encryption_spec_key_name: Customer-managed encryption key.
      force_direct_runner: Flag to use Beam DirectRunner. If set to true,
        use Apache Beam DirectRunner to execute the task locally instead of
        launching a Dataflow job.

  Returns:
      evaluation_metrics:
        google.ClassificationMetrics representing the classification
        evaluation metrics in GCS.
      gcp_resources: Serialized gcp_resources proto tracking the dataflow
        job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image='gcr.io/ml-pipeline/model-evaluation:v0.9',
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
          'regression',
          '--target_field_name',
          ConcatPlaceholder(['instance.', target_field_name]),
          '--batch_prediction_format',
          predictions_format,
          IfPresentPlaceholder(
              input_name='predictions_gcs_source',
              then=[
                  '--batch_prediction_gcs_source',
                  predictions_gcs_source.uri,
              ],
          ),
          IfPresentPlaceholder(
              input_name='predictions_bigquery_source',
              then=[
                  '--batch_prediction_bigquery_source',
                  ConcatPlaceholder([
                      'bq://',
                      predictions_bigquery_source.metadata['projectId'],
                      '.',
                      predictions_bigquery_source.metadata['datasetId'],
                      '.',
                      predictions_bigquery_source.metadata['tableId'],
                  ]),
              ],
          ),
          IfPresentPlaceholder(
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
          f'{PIPELINE_ROOT_PLACEHOLDER}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--prediction_score_column',
          prediction_score_column,
          '--dataflow_job_prefix',
          f'evaluation-regression-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
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
          '--force_direct_runner',
          force_direct_runner,
          '--output_metrics_gcs_path',
          evaluation_metrics.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
