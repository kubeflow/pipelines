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

from google_cloud_pipeline_components._implementation.model_evaluation import version
from google_cloud_pipeline_components.types.artifact_types import BQTable
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Metrics
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def feature_attribution(
    gcp_resources: OutputPath(str),
    feature_attributions: Output[Metrics],
    project: str,
    problem_type: str,
    location: str = 'us-central1',
    predictions_format: str = 'jsonl',
    predictions_gcs_source: Input[Artifact] = None,
    predictions_bigquery_source: Input[BQTable] = None,
    dataflow_service_account: str = '',
    dataflow_disk_size_gb: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
):
  # fmt: off
  """Compute feature attribution on a trained model's batch explanation
  results.

  Creates a dataflow job with Apache Beam and TFMA to compute feature
  attributions. Will compute feature attribution for every target label if
  possible, typically possible for AutoML Classification models.

  Args:
      project: Project to run feature attribution container.
      location: Location running feature attribution. If not
        set, defaulted to `us-central1`.
      problem_type: Problem type of the pipeline: one of `classification`,
      `regression` and `forecasting`.
      predictions_format: The file format for the batch
        prediction results. `jsonl`, `csv`, and `bigquery` are the allowed
        formats, from Vertex Batch Prediction. If not set, defaulted to `jsonl`.
      predictions_gcs_source: An artifact with its
        URI pointing toward a GCS directory with prediction or explanation files
        to be used for this evaluation. For prediction results, the files should
        be named "prediction.results-*" or "predictions_". For explanation
        results, the files should be named "explanation.results-*".
      predictions_bigquery_source: BigQuery table
        with prediction or explanation data to be used for this evaluation. For
        prediction results, the table column should be named "predicted_*".
      dataflow_service_account: Service account to run the
        dataflow job. If not set, dataflow will use the default worker service
        account. For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size_gb: The disk size (in GB) of the machine
        executing the evaluation run. If not set, defaulted to `50`.
      dataflow_machine_type: The machine type executing the
        evaluation run. If not set, defaulted to `n1-standard-4`.
      dataflow_workers_num: The number of workers executing the
        evaluation run. If not set, defaulted to `10`.
      dataflow_max_workers_num: The max number of workers
        executing the evaluation run. If not set, defaulted to `25`.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.
      encryption_spec_key_name: Customer-managed encryption key
        for the Dataflow job. If this is set, then all resources created by the
        Dataflow job will be encrypted with the provided encryption key.
      force_runner_mode: Flag to choose Beam runner. Valid options are `DirectRunner`
        and `Dataflow`.

  Returns:
      gcs_output_directory: JsonArray of the downsampled dataset GCS
        output.
      bigquery_output_table: String of the downsampled dataset BigQuery
        output.
      gcp_resources: Serialized gcp_resources proto tracking the dataflow
        job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
      command=[
          'python3',
          '/main.py',
      ],
      args=[
          '--task',
          'explanation',
          '--setup_file',
          '/setup.py',
          '--project_id',
          project,
          '--location',
          location,
          '--problem_type',
          problem_type,
          '--root_dir',
          f'{PIPELINE_ROOT_PLACEHOLDER}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
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
          '--dataflow_job_prefix',
          f'evaluation-feautre-attribution-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
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
          '--gcs_output_path',
          feature_attributions.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
