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

from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def evaluation_data_sampler(
    gcp_resources: OutputPath(str),
    bigquery_output_table: OutputPath(str),
    gcs_output_directory: OutputPath(list),
    project: str,
    location: str = 'us-central1',
    gcs_source_uris: List[str] = [],
    bigquery_source_uri: str = '',
    instances_format: str = 'jsonl',
    sample_size: int = 10000,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
):
  # fmt: off
  """Randomly downsamples an input dataset to a specified size.

  Used for computing Vertex XAI feature attributions for AutoML Tables and
  custom models. Creates a Dataflow job with Apache Beam to downsample the
  dataset.

  Args:
      project: Project to retrieve dataset from.
      location: Location to retrieve dataset from. If not set,
        defaulted to `us-central1`.
      gcs_source_uris: Google Cloud Storage URI(-s) to your
        instances to run data sampler on. They must match `instances_format`.
        May contain wildcards. For more information on wildcards, see
        https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
      bigquery_source_uri: Google BigQuery Table URI to your
        instances to run data sampler on.
      instances_format: The format in which instances are given,
        must be one of the model's supported input storage formats. If not set,
        default to "jsonl".
      sample_size: Sample size of the randomly sampled dataset.
        10k by default.
      dataflow_service_account: Service account to run the
        dataflow job. If not set, dataflow will use the default worker service
        account. For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
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
          'data_sampler',
          '--display_name',
          'data-sampler-run',
          '--project_id',
          project,
          '--location',
          location,
          '--root_dir',
          f'{PIPELINE_ROOT_PLACEHOLDER}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--gcs_source_uris',
          gcs_source_uris,
          '--bigquery_source_uri',
          bigquery_source_uri,
          '--instances_format',
          instances_format,
          '--sample_size',
          sample_size,
          '--dataflow_job_prefix',
          f'evaluation-data-sampler-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_subnetwork',
          dataflow_subnetwork,
          '--dataflow_use_public_ips',
          dataflow_use_public_ips,
          '--kms_key_name',
          encryption_spec_key_name,
          '--force_runner_mode',
          force_runner_mode,
          '--gcs_directory_for_gcs_output_uris',
          gcs_output_directory,
          '--gcs_directory_for_bigquery_output_table_uri',
          bigquery_output_table,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
