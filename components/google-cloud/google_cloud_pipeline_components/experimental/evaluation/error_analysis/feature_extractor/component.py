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

from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def feature_extractor_error_analysis(
    gcp_resources: OutputPath(str),
    embeddings_dir: OutputPath(str),
    project: str,
    root_dir: str,
    test_dataset: Input[VertexDataset],
    training_dataset: Input[VertexDataset],
    preprocessed_test_dataset_storage_source: str,
    preprocessed_training_dataset_storage_source: str,
    location: str = 'us-central1',
    dataflow_service_account: str = '',
    dataflow_disk_size: int = 50,
    dataflow_machine_type: str = 'n1-standard-8',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
):
  """Extracts feature embeddings of a dataset.

  Args:
      project (str): Required. GCP Project ID.
      location (Optional[str]): GCP Region. If not set, defaulted to
        `us-central1`.
      root_dir (str): Required. The GCS directory for keeping staging files. A
        random subdirectory will be created under the directory to keep job info
        for resuming the job in case of failure.
      test_dataset (google.VertexDataset): Required. A google.VertexDataset
        artifact of the test dataset.
      training_dataset (google.VertexDataset): Required. A google.VertexDataset
        artifact of the training dataset.
      preprocessed_test_dataset_storage_source (str): Required. Google Cloud
        Storage URI to preprocessed test dataset for Vision Analysis pipelines.
      preprocessed_training_dataset_storage_source (str): Required. Google Cloud
        Storage URI to preprocessed training dataset for Vision Error Analysis
        pipelines.
      dataflow_service_account (Optional[str]): Optional. Service account to run
        the dataflow job. If not set, dataflow will use the default woker
        service account.  For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size (Optional[int]): Optional. The disk size (in GB) of the
        machine executing the evaluation run. If not set, defaulted to `50`.
      dataflow_machine_type (Optional[str]): Optional. The machine type
        executing the evaluation run. If not set, defaulted to `n1-standard-4`.
      dataflow_workers_num (Optional[int]): Optional. The number of workers
        executing the evaluation run. If not set, defaulted to `10`.
      dataflow_max_workers_num (Optional[int]): Optional. The max number of
        workers executing the evaluation run. If not set, defaulted to `25`.
      dataflow_subnetwork (Optional[str]): Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More details:
          https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips (Optional[bool]): Specifies whether Dataflow
        workers use public IP addresses.
      encryption_spec_key_name (Optional[str]): Customer-managed encryption key
        for the Dataflow job. If this is set, then all resources created by the
        Dataflow job will be encrypted with the provided encryption key.

  Returns:
      embeddings_dir (str):
          Google Cloud Storage URI to computed feature embeddings of the image
          datasets.
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.

        For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:latest',
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
          '--gcp_resource',
          gcp_resources,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"display_name":',
              f' "feature-extractor-{PIPELINE_JOB_ID_PLACEHOLDER}',
              f'-{PIPELINE_TASK_ID_PLACEHOLDER}", ',
              '"job_spec": {"worker_pool_specs": [{"replica_count":"1',
              '", "machine_spec": {"machine_type": "n1-standard-4"},',
              ' "container_spec": {"image_uri":"',
              'gcr.io/cloud-aiplatform-private/starburst/v5/cmle:20230323_1221_RC00',
              '", "args": ["--project_id=',
              project,
              '", "--location=',
              location,
              '", "--root_dir=',
              f'{root_dir}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
              '", "--test_dataset_storage_source=',
              preprocessed_test_dataset_storage_source,
              '", "--training_dataset_storage_source=',
              preprocessed_training_dataset_storage_source,
              '", "--test_dataset_resource_name=',
              "{{$.inputs.artifacts['test_dataset'].metadata['resourceName']}}",
              '", "--training_dataset_resource_name=',
              "{{$.inputs.artifacts['training_dataset'].metadata['resourceName']}}",
              '", "--dataflow_job_prefix=',
              f'feature-extraction-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
              '", "--dataflow_service_account=',
              dataflow_service_account,
              '", "--dataflow_disk_size=',
              dataflow_disk_size,
              '", "--dataflow_machine_type=',
              dataflow_machine_type,
              '", "--dataflow_workers_num=',
              dataflow_workers_num,
              '", "--dataflow_max_workers_num=',
              dataflow_max_workers_num,
              '", "--dataflow_subnetwork=',
              dataflow_subnetwork,
              '", "--dataflow_use_public_ips=',
              dataflow_use_public_ips,
              '", "--kms_key_name=',
              encryption_spec_key_name,
              '", "--embeddings_dir=',
              embeddings_dir,
              '", "--executor_input={{$.json_escape[1]}}"]}}]}}',
          ]),
      ],
  )
