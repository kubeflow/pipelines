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
"""Python LLM Embedding Retrieval component used in KFP pipelines."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Input
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


@container_component
def llm_embedding_retrieval(
    gcp_resources: OutputPath(str),
    embedding_retrieval_results_path: OutputPath(str),
    project: str,
    location: str,
    query_embedding_source_directory: Input[Artifact],
    doc_embedding_source_directory: Input[Artifact],
    embedding_retrieval_top_n: int,
    embedding_retrieval_combination_function: str = 'max',
    display_name: str = 'llm_embedding_retrieval_component',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    runner: str = 'DirectRunner',
    dataflow_service_account: str = '',
    dataflow_disk_size_gb: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
):
  """Top N doc retrieval for queries, based on their embedding similarities.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      query_embedding_source_directory: Required. Directory where query
        embedding results are saved.
      doc_embedding_source_directory: Required. Directory where doc embedding
        results are saved.
      embedding_retrieval_top_n: Required. Top N docs will be retrieved for each
        query, based on similarity.
      embedding_retrieval_combination_function: The function to combine
        query-chunk similarities to query-doc similarity. Supported functions
        are avg, max, and sum.
      display_name: The name of the Evaluation job.
      machine_type: The machine type of this custom job. If not set, defaulted
        to `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      service_account: Sets the default service account for workload run-as
        account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      network: The full name of the Compute Engine network to which the job
        should be peered. For example, projects/12345/global/networks/myVPC.
        Format is of the form projects/{project}/global/networks/{network}.
        Where {project} is a project number, as in 12345, and {network} is a
        network name. Private services access must already be configured for the
        network. If left unspecified, the job is not peered with any network.
      runner: runner for the beam pipeline. DirectRunner and DataflowRunner are
        supported.
      dataflow_service_account: Service account to run the dataflow job. If not
        set, dataflow will use the default worker service account. For more
        details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size_gb: The disk size (in GB) of the machine executing the
        evaluation run.
      dataflow_machine_type: The machine type executing the evaluation run.
      dataflow_workers_num: The number of workers executing the evaluation run.
      dataflow_max_workers_num: The max number of workers executing the
        evaluation run.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when
        empty the default subnetwork will be used. More details:
          https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
        addresses.
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.
      embedding_retrieval_results_path (str):
        The prefix of sharded GCS output of document retrieval results based on
        embedding similarity, in JSONL format.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              f'--embedding_retrieval={True}',
              f'--project={project}',
              f'--location={location}',
              f'--query_embedding_source_directory={query_embedding_source_directory.path}',
              f'--doc_embedding_source_directory={doc_embedding_source_directory.path}',
              f'--embedding_retrieval_top_n={embedding_retrieval_top_n}',
              f'--embedding_retrieval_combination_function={embedding_retrieval_combination_function}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--gcp_resources={gcp_resources}',
              f'--embedding_retrieval_results_path={embedding_retrieval_results_path}',
              f'--runner={runner}',
              f'--dataflow_service_account={dataflow_service_account}',
              f'--dataflow_disk_size={dataflow_disk_size_gb}',
              f'--dataflow_machine_type={dataflow_machine_type}',
              f'--dataflow_workers_num={dataflow_workers_num}',
              f'--dataflow_max_workers_num={dataflow_max_workers_num}',
              f'--dataflow_subnetwork={dataflow_subnetwork}',
              f'--dataflow_use_public_ips={dataflow_use_public_ips}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
