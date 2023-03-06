# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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

from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath


@container_component
def custom_training_job(
    project: str,
    display_name: str,
    # TODO(b/243411151): misalignment of arguments in documentation vs function
    # signature.
    gcp_resources: OutputPath(str),
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
    encryption_spec_key_name: str = ''):
  """Launch a Custom training job using Vertex CustomJob API.

  Args:
      project (str): Required. Project to create the custom training job in.
      location (Optional[str]): Location for creating the custom training job.
        If not set, default to us-central1.
      display_name (str): The name of the custom training job.
      worker_pool_specs (Optional[Sequence[str]]): Serialized json spec of the
        worker pools including machine type and Docker image. All worker pools
        except the first one are optional and can be skipped by providing an
        empty value.  For more details about the WorkerPoolSpec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#WorkerPoolSpec
      timeout (Optional[str]): The maximum job running time. The default is 7
        days. A duration in seconds with up to nine fractional digits,
        terminated by 's', for example: "3.5s".
      restart_job_on_worker_restart (Optional[bool]): Restarts the entire
        CustomJob if a worker gets restarted. This feature can be used by
        distributed training jobs that are not resilient to workers leaving and
        joining a job.
      service_account (Optional[str]): Sets the default service account for
        workload run-as account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      tensorboard (Optional[str]): The name of a Vertex AI Tensorboard resource
        to which this CustomJob will upload Tensorboard logs.
      enable_web_access (Optional[bool]): Whether you want Vertex AI to enable
        [interactive shell
        access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell)
        to training containers. If set to `true`, you can access interactive
        shells at the URIs given by [CustomJob.web_access_uris][].
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
      base_output_directory (Optional[str]): The Cloud Storage location to store
        the output of this CustomJob or HyperparameterTuningJob. see below for
        more details:
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination
      labels (Optional[Dict[str, str]]): The labels with user-defined metadata
        to organize CustomJobs. See https://goo.gl/xmQnxf for more information.
      encryption_spec_key_name (Optional[str]): Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.

  Returns:
      gcp_resources (str):
          Serialized gcp_resources proto tracking the batch prediction job.

          For more details, see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-u', '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher'
      ],
      args=[
          '--type',
          'CustomJob',
          '--payload',
          ConcatPlaceholder([
              '{', '"display_name": "', display_name, '"', ', "job_spec": {',
              '"worker_pool_specs": ', worker_pool_specs, ', "scheduling": {',
              '"timeout": "', timeout, '"',
              ', "restart_job_on_worker_restart": "',
              restart_job_on_worker_restart, '"', '}', ', "service_account": "',
              service_account, '"', ', "tensorboard": "', tensorboard, '"',
              ', "enable_web_access": "', enable_web_access, '"',
              ', "network": "', network, '"', ', "reserved_ip_ranges": ',
              reserved_ip_ranges, ', "base_output_directory": {',
              '"output_uri_prefix": "', base_output_directory, '"', '}', '}',
              ', "labels": ', labels, ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name, '"}', '}'
          ]),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
      ])
