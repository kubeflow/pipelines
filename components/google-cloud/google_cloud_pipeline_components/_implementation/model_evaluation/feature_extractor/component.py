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

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
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
    preprocessed_test_dataset_storage_source: str,
    preprocessed_training_dataset_storage_source: str,
    test_dataset: Input[VertexDataset] = None,
    training_dataset: Input[VertexDataset] = None,
    location: str = 'us-central1',
    feature_extractor_machine_type: str = 'n1-standard-32',
    encryption_spec_key_name: str = '',
):
  # fmt: off
  """Extracts feature embeddings of a dataset.

  Args:
      project: GCP Project ID.
      location: GCP Region. If not set, defaulted to
        `us-central1`.
      root_dir: The GCS directory for keeping staging files. A
        random subdirectory will be created under the directory to keep job info
        for resuming the job in case of failure.
      preprocessed_test_dataset_storage_source: Google Cloud
        Storage URI to preprocessed test dataset for Vision Analysis pipelines.
      preprocessed_training_dataset_storage_source: Google Cloud
        Storage URI to preprocessed training dataset for Vision Error Analysis
        pipelines.
      test_dataset: A google.VertexDataset
        artifact of the test dataset.
      training_dataset: A google.VertexDataset
        artifact of the training dataset.
      feature_extractor_machine_type: The machine type executing
        the Apache Beam pipeline using DirectRunner. If not set, defaulted to
        `n1-standard-32`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      encryption_spec_key_name: Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.

  Returns:
      embeddings_dir: Google Cloud Storage URI to computed feature embeddings of the image
          datasets.
      gcp_resources: Serialized gcp_resources proto tracking the custom job.

        For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
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
              '", "machine_spec": {"machine_type": "',
              feature_extractor_machine_type,
              '"},',
              ' "container_spec": {"image_uri":"',
              'gcr.io/cloud-aiplatform-private/starburst/v5/cmle:20230420_0621_RC00',
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
              IfPresentPlaceholder(
                  input_name='test_dataset',
                  then=(
                      f"\", \"--test_dataset_resource_name={test_dataset.metadata['resourceName']}"
                  ),
                  else_='',
              ),
              IfPresentPlaceholder(
                  input_name='training_dataset',
                  then=(
                      f"\", \"--training_dataset_resource_name={training_dataset.metadata['resourceName']}"
                  ),
                  else_='',
              ),
              '", "--embeddings_dir=',
              embeddings_dir,
              '", "--executor_input={{$.json_escape[1]}}"]}}]}',
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name,
              '"}',
              '}',
          ]),
      ],
  )
