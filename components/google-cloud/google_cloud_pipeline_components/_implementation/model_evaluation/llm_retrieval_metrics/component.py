"""Information Retrieval metrics computation component used in KFP pipelines."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import container_component
from kfp.dsl import Metrics
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER

# pylint: disable=g-import-not-at-top, g-doc-args
@container_component
def llm_retrieval_metrics(
    gcp_resources: OutputPath(str),
    retrieval_metrics: Output[Metrics],
    project: str,
    location: str,
    golden_docs_pattern: str,
    embedding_retrieval_results_pattern: str,
    retrieval_metrics_top_k_list: str,
    display_name: str = 'llm_retrieval_metrics_component',
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
  """Compute retrieval metrics based on the docs retrieved for each query.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      golden_docs_pattern: Required. Files where queries and corresponding
        golden doc ids are saved. The path pattern can contain glob characters
        (`*`, `?`, and `[...]` sets).
      embedding_retrieval_results_pattern: Required. Files where doc retrieval
        results for each query are saved. The path pattern can contain glob
        characters (`*`, `?`, and `[...]` sets).
      retrieval_metrics_top_k_list: Required. k values for retrieval metrics,
        for example, precision@k, accuracy@k, etc. If more than one value,
        separated by comma. e.g., "1,5,10".
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
      gcp_resources:
        Serialized gcp_resources proto tracking the custom job.
      retrieval_metrics:
        A Metrics artifact representing the retrieval metrics.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              f'--retrieval_metrics={True}',
              f'--project={project}',
              f'--location={location}',
              f'--golden_docs_pattern={golden_docs_pattern}',
              f'--embedding_retrieval_results_pattern={embedding_retrieval_results_pattern}',
              f'--retrieval_metrics_top_k_list={retrieval_metrics_top_k_list}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--gcp_resources={gcp_resources}',
              f'--retrieval_metrics_output_path={retrieval_metrics.path}',
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
