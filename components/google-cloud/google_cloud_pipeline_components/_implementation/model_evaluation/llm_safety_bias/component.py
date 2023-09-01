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
"""LLM Safety Bias Metrics component used in KFP pipelines."""

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.model_evaluation import utils
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import Output
from kfp.dsl import OutputPath

_IMAGE_URI = 'us-docker.pkg.dev/vertex-ai-restricted/llm-eval/llm-bias:v0.2'


@container_component
def llm_safety_bias_metrics(
    gcp_resources: OutputPath(str),
    llm_safety_bias_evaluation_metrics: Output[Artifact],
    location: str = 'us-central1',
    slice_spec_gcs_source: str = '',
    predictions_gcs_source: str = '',
    display_name: str = 'llm-safety-bias-component',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  """Aggregates LLM safety bias metrics based on specified data slices.

  Args:
    location: The GCP region that runs the pipeline components.
    slice_spec_gcs_source: Google Cloud Storage location to file with JSONL
      slicing spec definition.
    predictions_gcs_source: A storage URI pointing toward a GCS file or
      directory with prediction results to be used for this evaluation.
    display_name: The display name of the evaluation custom job.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
      https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account. If
      unspecified, the Vertex AI Custom Code Service
      Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
      for the CustomJob's project.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, `projects/12345/global/networks/myVPC`. Format is
      of the form `projects/{project}/global/networks/{network}`. Where
      `{project}` is a project number, as in `12345`, and `{network}` is a
      network name, as in `myVPC`. To specify this field, you must have already
      configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    encryption_spec_key_name: Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`.
      The key needs to be in the same region as where the compute resource is
      created.
    project: The GCP project that runs the pipeline components. Defaults to the
      project in which the PipelineJob is run.

  Returns:
    llm_safety_bias_evaluation_metrics: `Artifact` tracking the LLM safety
      bias evaluation metrics output.
    gcp_resources: Serialized gcp_resources proto tracking the custom job.
      For more details, see
      https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
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
              f'--bias_llm_metrics={llm_safety_bias_evaluation_metrics.path}',
              '--executor_input={{$.json_escape[1]}}',
          ],
          service_account=service_account,
          network=network,
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
