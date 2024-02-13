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
"""Information Retrieval preprocessor component used in KFP pipelines."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import container_component
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


@container_component
def llm_information_retrieval_preprocessor(
    gcp_resources: OutputPath(str),
    predictions_query_gcs_source: OutputPath(list),
    predictions_corpus_gcs_source: OutputPath(list),
    embedding_retrieval_gcs_source: OutputPath(str),
    project: str,
    location: str,
    corpus_gcs_source: str,
    query_gcs_source: str,
    golden_docs_gcs_source: str,
    embedding_chunking_function: str = 'langchain-RecursiveCharacterTextSplitter',
    embedding_chunk_size: int = 0,
    embedding_chunk_overlap: int = 0,
    display_name: str = 'information-retrieval-preprocessor',
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
  """Preprocess inputs for information retrieval task - json files for corpus

    and queries, and csv for query to golden docs mapping.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      corpus_gcs_source: Required. The path for json file containing corpus
        documents.
      query_gcs_source: Required. The path for json file containing query
        documents.
      golden_docs_gcs_source: Required. The path for csv file containing mapping
        of each query to the golden docs.
      embedding_chunking_function: function used to split a document into
        chunks. Supported values are `langchain-RecursiveCharacterTextSplitter`
        and `sentence-splitter`.
      embedding_chunk_size: The length of each document chunk. If 0, chunking is
        not enabled.
      embedding_chunk_overlap: The length of the overlap part between adjacent
        chunks. Will only be used if embedding_chunk_size > 0.
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
      dataflow_machine_type: The dataflow worker machine type executing the
        evaluation run.
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
      predictions_query_gcs_source (list):
        The GCS directory to save preprocessed query data to run batch
        prediction.
      predictions_corpus_gcs_source (list):
        The GCS directory to save preprocessed corpus data to run batch
        prediction.
      embedding_retrieval_gcs_source (str):
        The GCS directory to save preprocessed golden docs mapping data to run
        batch prediction.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              f'--information_retrieval_preprocessor={True}',
              f'--project={project}',
              f'--location={location}',
              f'--corpus_gcs_source={corpus_gcs_source}',
              f'--query_gcs_source={query_gcs_source}',
              f'--golden_docs_gcs_source={golden_docs_gcs_source}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--gcp_resources={gcp_resources}',
              f'--predictions_query_gcs_source={predictions_query_gcs_source}',
              f'--predictions_corpus_gcs_source={predictions_corpus_gcs_source}',
              f'--embedding_retrieval_gcs_source={embedding_retrieval_gcs_source}',
              f'--embedding_chunking_function={embedding_chunking_function}',
              f'--embedding_chunk_size={embedding_chunk_size}',
              f'--embedding_chunk_overlap={embedding_chunk_overlap}',
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
