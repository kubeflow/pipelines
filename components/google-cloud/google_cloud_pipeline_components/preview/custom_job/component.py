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

from typing import Dict, List

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils
from kfp import dsl


# keep consistent with create_custom_training_job_from_component
@dsl.container_component
def custom_training_job(
    display_name: str,
    gcp_resources: dsl.OutputPath(str),
    location: str = 'us-central1',
    worker_pool_specs: List[Dict[str, str]] = [],
    timeout: str = '604800s',
    restart_job_on_worker_restart: bool = False,
    service_account: str = '',
    tensorboard: str = '',
    enable_web_access: bool = False,
    network: str = '',
    reserved_ip_ranges: List[str] = [],
    base_output_directory: str = '',
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
    persistent_resource_id: str = _placeholders.PERSISTENT_RESOURCE_ID_PLACEHOLDER,
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Launch a Vertex AI [custom training job](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) using the [CustomJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs) API. See [Create custom training jobs ](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) for more information.

  Args:
    location: Location for creating the custom training job. If not set, default to us-central1.
    display_name: The name of the CustomJob.
    worker_pool_specs: Serialized json spec of the worker pools including machine type and Docker image. All worker pools except the first one are optional and can be skipped by providing an empty value. See [more information](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#WorkerPoolSpec).
    timeout: The maximum job running time. The default is 7 days. A duration in seconds with up to nine fractional digits, terminated by 's', for example: "3.5s".
    restart_job_on_worker_restart: Restarts the entire CustomJob if a worker gets restarted. This feature can be used by distributed training jobs that are not resilient to workers leaving and joining a job.
    service_account: Sets the default service account for workload run-as account. The [service account ](https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account) running the pipeline submitting jobs must have act-as permission on this run-as account. If unspecified, the Vertex AI Custom Code [Service Agent ](https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents) for the CustomJob's project.
    tensorboard: The name of a Vertex AI TensorBoard resource to which this CustomJob will upload TensorBoard logs.
    enable_web_access: Whether you want Vertex AI to enable [interactive shell access ](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell) to training containers. If `True`, you can access interactive shells at the URIs given by [CustomJob.web_access_uris][].
    network: The full name of the Compute Engine network to which the job should be peered. For example, `projects/12345/global/networks/myVPC`. Format is of the form `projects/{project}/global/networks/{network}`. Where `{project}` is a project number, as in `12345`, and `{network}` is a network name. Private services access must already be configured for the network. If left unspecified, the job is not peered with any network.
    reserved_ip_ranges: A list of names for the reserved IP ranges under the VPC network that can be used for this job. If set, we will deploy the job within the provided IP ranges. Otherwise, the job will be deployed to any IP ranges under the provided VPC network.
    base_output_directory: The Cloud Storage location to store the output of this CustomJob or HyperparameterTuningJob. See [more information ](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination).
    labels: The labels with user-defined metadata to organize the CustomJob. See [more information](https://goo.gl/xmQnxf).
    encryption_spec_key_name: Customer-managed encryption key options for the CustomJob. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key.
    persistent_resource_id: The ID of the PersistentResource in the same Project and Location which to run. The default value is a placeholder that will be resolved to the PipelineJob [RuntimeConfig](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs#PipelineJob.RuntimeConfig)'s persistent resource id at runtime. However, if the PipelineJob doesn't set Persistent Resource as the job level runtime, the placedholder will be resolved to an empty string and the custom job will be run on demand. If the value is set explicitly, the custom job will runs in the specified persistent resource, in this case, please note the network and CMEK configs on the job should be consistent with those on the PersistentResource, otherwise, the job will be rejected. (This is a Preview feature not yet recommended for production workloads.)
    project: Project to create the custom training job in. Defaults to the project in which the PipelineJob is run.
  Returns:
    gcp_resources: Serialized JSON of `gcp_resources` [proto](https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto) which tracks the CustomJob.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.preview.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--payload',
          utils.container_component_dumps({
              'display_name': display_name,
              'job_spec': {
                  'worker_pool_specs': worker_pool_specs,
                  'scheduling': {
                      'timeout': timeout,
                      'restart_job_on_worker_restart': (
                          restart_job_on_worker_restart
                      ),
                  },
                  'service_account': service_account,
                  'tensorboard': tensorboard,
                  'enable_web_access': enable_web_access,
                  'network': network,
                  'reserved_ip_ranges': reserved_ip_ranges,
                  'base_output_directory': {
                      'output_uri_prefix': base_output_directory
                  },
                  'persistent_resource_id': persistent_resource_id,
              },
              'labels': labels,
              'encryption_spec': {'kms_key_name': encryption_spec_key_name},
          }),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
      ],
  )
