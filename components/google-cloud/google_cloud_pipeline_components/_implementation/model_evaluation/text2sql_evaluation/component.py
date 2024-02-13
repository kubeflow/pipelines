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
"""Text2SQL evaluation component used in KFP pipelines."""

from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Input
from kfp.dsl import Metrics
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


@container_component
def text2sql_evaluation(
    gcp_resources: OutputPath(str),
    error_analysis_path: OutputPath(str),
    generated_sql_results_path: OutputPath(str),
    text2sql_evaluation_metrics: Output[Metrics],
    project: str,
    location: str,
    sql_dialect: str,
    evaluation_method: str,
    model_inference_results_directory: Input[Artifact],
    tables_metadata_path: str,
    display_name: str = 'text2sql-evaluation',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
):
  """Get evaluation metrics, generated SQL results and error analysis in text2sql evaluation pipeline.

  Args:
      project: Required. The GCP project that runs the pipeline component.
      location: Required. The GCP region that runs the pipeline component.
      sql_dialect: Required. SQL dialect type, e.g. bigquery, mysql, etc.
      evaluation_method: Required. Text2SQL evaluation method, value can be
        'parser', 'execution', 'all'.
      model_inference_results_directory: Required. The path for json file
        containing text2sql model inference results from the last step.
      tables_metadata_path: Required. The path for json file containing database
        metadata, including table names, schema fields.
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
      encryption_spec_key_name: Customer-managed encryption key options for the
        CustomJob. If this is set, then all resources created by the CustomJob
        will be encrypted with the provided encryption key.

  Returns:
      gcp_resources (str):
        Serialized gcp_resources proto tracking the custom job.
      error_analysis_path:
        Path for aggregrated error analysis for genereated SQL queris.
      generated_sql_results_path:
        Path for generated sql queries and execution results (optional)
        in json format.
      text2sql_evaluation_metrics:
        A Metrics artifact representing the text2sql evaluation metrics.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_custom_job_payload(
          display_name=display_name,
          machine_type=machine_type,
          image_uri=version.LLM_EVAL_IMAGE_TAG,
          args=[
              f'--text2sql_evaluation={True}',
              f'--project={project}',
              f'--location={location}',
              f'--sql_dialect={sql_dialect}',
              f'--evaluation_method={evaluation_method}',
              f'--model_inference_results_directory={model_inference_results_directory.path}',
              f'--tables_metadata_path={tables_metadata_path}',
              f'--root_dir={PIPELINE_ROOT_PLACEHOLDER}',
              f'--gcp_resources={gcp_resources}',
              f'--error_analysis_path={error_analysis_path}',
              f'--generated_sql_results_path={generated_sql_results_path}',
              f'--text2sql_evaluation_metrics_output_path={text2sql_evaluation_metrics.path}',
              f'--error_analysis_path={error_analysis_path}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
