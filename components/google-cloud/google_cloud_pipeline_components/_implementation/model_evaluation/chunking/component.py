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
"""Embedding eval chunking component used in KFP pipelines."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from kfp.dsl import container_component
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


_IMAGE_URI = 'us-docker.pkg.dev/vertex-evaluation/public/llm:v0.3'


@container_component
def chunking(
    gcp_resources: OutputPath(str),
    project: str,
    location: str,
    input_text_gcs_dir: str,
    output_bq_destination: str,
    output_text_gcs_dir: str,
    output_error_file_path: str,
    generation_threshold_microseconds: str,
    display_name: str = 'chunking',
    machine_type: str = 'n1-standard-8',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
):
  """Chunk files in GCS and write to BigQuery table.

  Args:
    project: The GCP project that runs the pipeline component.
    location: The GCP region that runs the pipeline component.
    input_text_gcs_dir: the GCS directory containing the files to chunk. DO NOT
      include '/' at the end of the path.
    output_bq_destination: The BigQuery table URI where the component will write
      chunks to.
    output_text_gcs_dir: The GCS folder to hold intermediate data.
    output_error_file_path: The path to the file containing chunking error.
    generation_threshold_microseconds: only files created on/after this
      generation threshold will be processed, in microseconds.
    display_name: The name of the chunking job/component.
    machine_type: The machine type of this custom job.
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, projects/12345/global/networks/myVPC. Format is of
      the form projects/{project}/global/networks/{network}. Where {project} is
      a project number, as in 12345, and {network} is a network name. Private
      services access must already be configured for the network. If left
      unspecified, the job is not peered with any network.
    encryption_spec_key_name: Customer-managed encryption key options for the
      customJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    gcp_resources (str):
      Serialized gcp_resources proto tracking the custom job.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=_IMAGE_URI,
          args=[
              f'--process_chunking={True}',
              f'--project={project}',
              f'--location={location}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--input_text_gcs_dir={input_text_gcs_dir}',
              f'--output_bq_destination={output_bq_destination}',
              f'--output_text_gcs_dir={output_text_gcs_dir}',
              f'--output_error_file_path={output_error_file_path}',
              f'--generation_threshold_microseconds={generation_threshold_microseconds}',
              f'--gcp_resources={gcp_resources}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
