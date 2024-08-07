# Copyright 2023 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Module for supporting Google Vertex AI Custom Training Job Op."""

import copy
import textwrap
from typing import Callable, Dict, List, Optional
import warnings

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components.preview.custom_job import component
from kfp import components
import yaml

from google.protobuf import json_format


def _replace_executor_placeholder(
    container_input: List[str],
) -> List[str]:
  """Replace executor placeholder in container command or args.

  Args:
    container_input: Container command or args.

  Returns: container_input with executor placeholder replaced.
  """
  # Executor replacement is used as executor content needs to be jsonified before
  # injection into the payload, since payload is already a JSON serialized string.
  EXECUTOR_INPUT_PLACEHOLDER = '{{$}}'
  JSON_ESCAPED_EXECUTOR_INPUT_PLACEHOLDER = '{{$.json_escape[1]}}'
  return [
      JSON_ESCAPED_EXECUTOR_INPUT_PLACEHOLDER
      if cmd_part == EXECUTOR_INPUT_PLACEHOLDER
      else cmd_part
      for cmd_part in container_input
  ]


# keep identical to CustomTrainingJobOp
def create_custom_training_job_from_component(
    component_spec: Callable,
    display_name: str = '',
    replica_count: int = 1,
    machine_type: str = 'n1-standard-4',
    accelerator_type: str = 'ACCELERATOR_TYPE_UNSPECIFIED',
    accelerator_count: int = 1,
    boot_disk_type: str = 'pd-ssd',
    boot_disk_size_gb: int = 100,
    timeout: str = '604800s',
    restart_job_on_worker_restart: bool = False,
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
    tensorboard: str = '',
    enable_web_access: bool = False,
    reserved_ip_ranges: Optional[List[str]] = None,
    nfs_mounts: Optional[List[Dict[str, str]]] = None,
    base_output_directory: str = '',
    labels: Optional[Dict[str, str]] = None,
    persistent_resource_id: str = _placeholders.PERSISTENT_RESOURCE_ID_PLACEHOLDER,
    env: Optional[List[Dict[str, str]]] = None,
) -> Callable:
  # fmt: off
  """Convert a KFP component into Vertex AI [custom training job](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) using the [CustomJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs) API.

  This utility converts a [KFP component](https://www.kubeflow.org/docs/components/pipelines/v2/components/) provided to `component_spec` into `CustomTrainingJobOp` component. Your components inputs, outputs, and logic are carried over, with additional [CustomJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec) parameters exposed. Note that this utility constructs a ClusterSpec where the master and all the workers use the same spec, meaning all disk/machine spec related parameters will apply to all replicas. This is suitable for uses cases such as executing a training component over multiple replicas with [MultiWorkerMirroredStrategy](https://www.tensorflow.org/api_docs/python/tf/distribute/MultiWorkerMirroredStrategy) or [MirroredStrategy](https://www.tensorflow.org/api_docs/python/tf/distribute/MirroredStrategy). See [Create custom training jobs](https://cloud.google.com/vertex-ai/docs/training/create-custom-job) for more information.

  Args:
      component_spec: A KFP component.
      display_name: The name of the CustomJob. If not provided the component's name will be used instead.
      replica_count: The count of instances in the cluster. One replica always counts towards the master in worker_pool_spec[0] and the remaining replicas will be allocated in worker_pool_spec[1]. See [more information.](https://cloud.google.com/vertex-ai/docs/training/distributed-training#configure_a_distributed_training_job)
      machine_type: The type of the machine to run the CustomJob. The default value is "n1-standard-4". See [more information](https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types).
      accelerator_type: The type of accelerator(s) that may be attached to the machine per `accelerator_count`. See [more information](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec#acceleratortype).
      accelerator_count: The number of accelerators to attach to the machine. Defaults to 1 if `accelerator_type` is set statically.
      boot_disk_type: Type of the boot disk (default is "pd-ssd"). Valid values: "pd-ssd" (Persistent Disk Solid State Drive) or "pd-standard" (Persistent Disk Hard Disk Drive).
      boot_disk_size_gb: Size in GB of the boot disk (default is 100GB).
      timeout: The maximum job running time. The default is 7 days. A duration in seconds with up to nine fractional digits, terminated by 's', for example: "3.5s".
      restart_job_on_worker_restart: Restarts the entire CustomJob if a worker gets restarted. This feature can be used by distributed training jobs that are not resilient to workers leaving and joining a job.
      service_account: Sets the default service account for workload run-as account. The [service account](https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account) running the pipeline submitting jobs must have act-as permission on this run-as account. If unspecified, the Vertex AI Custom Code [Service Agent](https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents) for the CustomJob's project.
      network: The full name of the Compute Engine network to which the job should be peered. For example, `projects/12345/global/networks/myVPC`. Format is of the form `projects/{project}/global/networks/{network}`. Where `{project}` is a project number, as in `12345`, and `{network}` is a network name. Private services access must already be configured for the network. If left unspecified, the job is not peered with any network.
      encryption_spec_key_name: Customer-managed encryption key options for the CustomJob. If this is set, then all resources created by the CustomJob will be encrypted with the provided encryption key.
      tensorboard: The name of a Vertex AI TensorBoard resource to which this CustomJob will upload TensorBoard logs.
      enable_web_access: Whether you want Vertex AI to enable [interactive shell access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell) to training containers. If `True`, you can access interactive shells at the URIs given by [CustomJob.web_access_uris][].
      reserved_ip_ranges: A list of names for the reserved IP ranges under the VPC network that can be used for this job. If set, we will deploy the job within the provided IP ranges. Otherwise, the job will be deployed to any IP ranges under the provided VPC network.
      nfs_mounts: A list of [NfsMount](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#NfsMount) resource specs in Json dict format. For more details about mounting NFS for CustomJob, see [Mount an NFS share for custom training](https://cloud.google.com/vertex-ai/docs/training/train-nfs-share). `nfs_mounts` is set as a static value and cannot be changed as a pipeline parameter.
      base_output_directory: The Cloud Storage location to store the output of this CustomJob or HyperparameterTuningJob. See [more information](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination).
      labels: The labels with user-defined metadata to organize the CustomJob. See [more information](https://goo.gl/xmQnxf).
      persistent_resource_id: The ID of the PersistentResource in the same Project and Location which to run. The default value is a placeholder that will be resolved to the PipelineJob [RuntimeConfig](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs#PipelineJob.RuntimeConfig)'s persistent resource id at runtime. However, if the PipelineJob doesn't set Persistent Resource as the job level runtime, the placedholder will be resolved to an empty string and the custom job will be run on demand. If the value is set explicitly, the custom job will runs in the specified persistent resource, in this case, please note the network and CMEK configs on the job should be consistent with those on the PersistentResource, otherwise, the job will be rejected. (This is a Preview feature not yet recommended for production workloads.)
      env: Environment variables to be passed to the container. Takes the form `[{'name': '...', 'value': '...'}]`. Maximum limit is 100. `env` is set as a static value and cannot be changed as a pipeline parameter.

  Returns:
      A KFP component with CustomJob specification applied.
  """
  # fmt: on
  # This function constructs a Custom Job component based on the input
  # component, by performing a 3-way merge of the inputs/outputs of the
  # input component, the Custom Job component and the arguments given to this
  # function.
  #
  # It first retrieves the PipelineSpec (as a Python dict) for each of the two
  # components (the input component and the Custom Job component).
  #   Note: The advantage of using the PipelineSpec here is that the
  #   placeholders are (mostly) serialized, so there is less processing
  #   needed (and avoids unnecessary dependency on KFP internals).
  #
  # The arguments to this function are first inserted into each input parameter
  # of the Custom Job component as a default value (which will be used at
  # runtime, unless when overridden by specifying the input).
  # One particular input parameter that needs detailed construction is the
  # worker_pool_spec, before being inserted into the Custom Job component.
  #
  # After inserting the arguments into the Custom Job input parameters as
  # default values, the input/output parameters from the input component are
  # then merged with the Custom Job input/output parameters. Preference is given
  # to Custom Job input parameters to make sure they are not overridden (which
  # follows the same logic as the original version).
  #
  # It is assumed that the Custom Job component itself has no input/output
  # artifacts, so the artifacts from the input component needs no merging.
  # (There is a unit test to make sure this is the case, otherwise merging of
  # artifacts need to be done here.)
  #
  # Once the above is done, and the dict of the Custom Job is converted back
  # into a KFP component (by first converting to YAML, then using
  # load_component_from_text to load the YAML).
  # After adding the appropriate description and the name, the new component
  # is returned.

  cj_pipeline_spec = json_format.MessageToDict(
      component.custom_training_job.pipeline_spec
  )
  user_pipeline_spec = json_format.MessageToDict(component_spec.pipeline_spec)

  user_component_container = list(
      user_pipeline_spec['deploymentSpec']['executors'].values()
  )[0]['container']

  worker_pool_spec = {
      'machine_spec': {
          'machine_type': "{{$.inputs.parameters['machine_type']}}",
          'accelerator_type': "{{$.inputs.parameters['accelerator_type']}}",
          'accelerator_count': "{{$.inputs.parameters['accelerator_count']}}",
      },
      'replica_count': 1,
      'container_spec': {
          'image_uri': user_component_container['image'],
          'command': _replace_executor_placeholder(
              user_component_container.get('command', [])
          ),
          'args': _replace_executor_placeholder(
              user_component_container.get('args', [])
          ),
          'env': env or [],
      },
      'disk_spec': {
          'boot_disk_type': "{{$.inputs.parameters['boot_disk_type']}}",
          'boot_disk_size_gb': "{{$.inputs.parameters['boot_disk_size_gb']}}",
      },
  }
  if nfs_mounts:
    worker_pool_spec['nfs_mounts'] = nfs_mounts

  worker_pool_specs = [worker_pool_spec]

  if int(replica_count) > 1:
    additional_worker_pool_spec = copy.deepcopy(worker_pool_spec)
    additional_worker_pool_spec['replica_count'] = replica_count - 1
    worker_pool_specs.append(additional_worker_pool_spec)

  # get the component spec for both components
  cj_component_spec_key = list(cj_pipeline_spec['components'].keys())[0]
  cj_component_spec = cj_pipeline_spec['components'][cj_component_spec_key]

  user_component_spec_key = list(user_pipeline_spec['components'].keys())[0]
  user_component_spec = user_pipeline_spec['components'][
      user_component_spec_key
  ]

  # add custom job defaults based on user-provided args
  custom_job_param_defaults = {
      'display_name': display_name or component_spec.component_spec.name,
      'worker_pool_specs': worker_pool_specs,
      'timeout': timeout,
      'restart_job_on_worker_restart': restart_job_on_worker_restart,
      'service_account': service_account,
      'tensorboard': tensorboard,
      'enable_web_access': enable_web_access,
      'network': network,
      'reserved_ip_ranges': reserved_ip_ranges or [],
      'base_output_directory': base_output_directory,
      'labels': labels or {},
      'encryption_spec_key_name': encryption_spec_key_name,
      'persistent_resource_id': persistent_resource_id,
  }

  for param_name, default_value in custom_job_param_defaults.items():
    cj_component_spec['inputDefinitions']['parameters'][param_name][
        'defaultValue'
    ] = default_value

  # add workerPoolSpec parameters into the customjob component
  cj_component_spec['inputDefinitions']['parameters']['machine_type'] = {
      'parameterType': 'STRING',
      'defaultValue': machine_type,
      'isOptional': True,
  }
  cj_component_spec['inputDefinitions']['parameters']['accelerator_type'] = {
      'parameterType': 'STRING',
      'defaultValue': accelerator_type,
      'isOptional': True,
  }
  cj_component_spec['inputDefinitions']['parameters']['accelerator_count'] = {
      'parameterType': 'NUMBER_INTEGER',
      'defaultValue': (
          accelerator_count
          if accelerator_type != 'ACCELERATOR_TYPE_UNSPECIFIED'
          else 0
      ),
      'isOptional': True,
  }
  cj_component_spec['inputDefinitions']['parameters']['boot_disk_type'] = {
      'parameterType': 'STRING',
      'defaultValue': boot_disk_type,
      'isOptional': True,
  }
  cj_component_spec['inputDefinitions']['parameters']['boot_disk_size_gb'] = {
      'parameterType': 'NUMBER_INTEGER',
      'defaultValue': boot_disk_size_gb,
      'isOptional': True,
  }

  # check if user component has any input parameters that already exist in the
  # custom job component
  for param_name in user_component_spec.get('inputDefinitions', {}).get(
      'parameters', {}
  ):
    if param_name in cj_component_spec['inputDefinitions']['parameters']:
      raise ValueError(
          f'Input parameter {param_name} already exists in the CustomJob component.'  # pylint: disable=line-too-long
      )
  for param_name in user_component_spec.get('outputDefinitions', {}).get(
      'parameters', {}
  ):
    if param_name in cj_component_spec['outputDefinitions']['parameters']:
      raise ValueError(
          f'Output parameter {param_name} already exists in the CustomJob component.'  # pylint: disable=line-too-long
      )

  # merge parameters from user component into the customjob component
  cj_component_spec['inputDefinitions']['parameters'].update(
      user_component_spec.get('inputDefinitions', {}).get('parameters', {})
  )
  cj_component_spec['outputDefinitions']['parameters'].update(
      user_component_spec.get('outputDefinitions', {}).get('parameters', {})
  )

  # use artifacts from user component
  ## assign artifacts, not update, since customjob has no artifact outputs
  cj_component_spec['inputDefinitions']['artifacts'] = user_component_spec.get(
      'inputDefinitions', {}
  ).get('artifacts', {})
  cj_component_spec['outputDefinitions']['artifacts'] = user_component_spec.get(
      'outputDefinitions', {}
  ).get('artifacts', {})

  # copy the input definitions to the root, which will have an identical interface for a single-step pipeline
  cj_pipeline_spec['root']['inputDefinitions'] = copy.deepcopy(
      cj_component_spec['inputDefinitions']
  )
  cj_pipeline_spec['root']['outputDefinitions'] = copy.deepcopy(
      cj_component_spec['outputDefinitions']
  )

  # update the customjob task with the user inputs
  cj_task_key = list(cj_pipeline_spec['root']['dag']['tasks'].keys())[0]
  user_task_key = list(user_pipeline_spec['root']['dag']['tasks'].keys())[0]

  cj_pipeline_spec['root']['dag']['tasks'][cj_task_key]['inputs'].update(
      user_pipeline_spec['root']['dag']['tasks'][user_task_key].get(
          'inputs', {}
      )
  )

  # reload the pipelinespec as a component using KFP
  new_component = components.load_component_from_text(
      yaml.safe_dump(cj_pipeline_spec)
  )

  # Copy the component name and description
  # TODO(b/262360354): The inner .component_spec.name is needed here as that is
  # the name that is retrieved by the FE for display. Can simply reference the
  # outer .name once setter is implemented.
  new_component.component_spec.name = component_spec.component_spec.name

  if component_spec.description:
    component_description = textwrap.dedent(f"""
    A CustomJob that wraps {component_spec.component_spec.name}.

    Original component description:
    {component_spec.description}

    Custom Job wrapper description:
    {component.custom_training_job.description}
    """)
    new_component.description = component_description

  return new_component


def create_custom_training_job_op_from_component(*args, **kwargs) -> Callable:
  """Deprecated.

  Please use create_custom_training_job_from_component instead.
  """

  warnings.warn(
      f'{create_custom_training_job_op_from_component.__name__!r} is'
      ' deprecated. Please use'
      f' {create_custom_training_job_from_component.__name__!r} instead.',
      DeprecationWarning,
  )

  return create_custom_training_job_from_component(*args, **kwargs)
