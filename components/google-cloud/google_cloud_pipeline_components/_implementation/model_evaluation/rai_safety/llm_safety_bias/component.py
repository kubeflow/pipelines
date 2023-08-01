"""Python LLM Safety Model Evaluation component used in KFP pipelines."""

from typing import List, Optional

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Output
from kfp.dsl import OutputPath


_IMAGE_URI = 'us-docker.pkg.dev/vertex-ai-restricted/llm-eval/llm-bias:v0.2'


@container_component
def llm_safety_metrics_bias(
    gcp_resources: OutputPath(str),
    bias_llm_metrics: Output[Artifact],
    project: str,
    location: str = 'us-central1',
    slice_spec_gcs_source: str = '',
    predictions_gcs_source: str = '',
    display_name: str = 'llm_safety_bias_component',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: Optional[List[str]] = None,
    encryption_spec_key_name: str = '',
):
  """Reports aggregated safety metrics from a model's predictions based on specified data slices.

  Args:
      project (str): Required. Project to run the component.
      location (Optional[str]): Location for running the component. If not set,
        defaulted to `us-central1`.
      slice_spec_gcs_source (Optional[str]): Google Cloud Storage location to
        file with jsonl slice spec definition.
      predictions_gcs_source (Optional[str]): A storage URI pointing toward a
        GCS file or directory with prediction results to be used for this
        evaluation.
      display_name (Optional[str]): The name of the Evaluation job.
      machine_type (Optional[str]): The machine type of this custom job. If not
        set, defaulted to `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
      service_account (Optional[str]): Sets the default service account for
        workload run-as account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      network (Optional[str]): The full name of the Compute Engine network to
        which the job should be peered. For example,
        projects/12345/global/networks/myVPC. Format is of the form
        projects/{project}/global/networks/{network}. Where {project} is a
        project number, as in 12345, and {network} is a network name. Private
        services access must already be configured for the network. If left
        unspecified, the job is not peered with any network.
      reserved_ip_ranges (Optional[Sequence[str]]): A list of names for the
        reserved ip ranges under the VPC network that can be used for this job.
        If set, we will deploy the job within the provided ip ranges. Otherwise,
        the job will be deployed to any ip ranges under the provided VPC
        network.
      encryption_spec_key_name (Optional[str]): Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.

  Returns:
      bias_llm_metrics (system.Artifact):
        Artifact tracking the LLM model bias detection output.
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
              f'--safety_metrics={True}',
              f'--predictions_gcs_source={predictions_gcs_source}',
              f'--slice_spec_gcs_source={slice_spec_gcs_source}',
              f'--bias_llm_metrics={bias_llm_metrics.path}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          reserved_ip_ranges=reserved_ip_ranges,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
