# Copyright 2021 The Kubeflow Authors
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

# Prior to release of kfp V2, we have to use a mix of kfp v1 and v2.
# TODO(chavoshi): switch to using V2 only once it is ready.
import copy
import json
import os
import tempfile
from typing import Callable, Dict, Optional, Sequence
from google_cloud_pipeline_components.aiplatform import utils
from kfp import components
from kfp.dsl import dsl_utils
from kfp.v2.components.types import type_utils
import yaml

_DEFAULT_CUSTOM_JOB_CONTAINER_IMAGE = utils.DEFAULT_CONTAINER_IMAGE
# Executor replacement is used as executor content needs to be jsonified before
# injection into the payload, since payload is already a Json serialized string.
_EXECUTOR_PLACE_HOLDER_REPLACEMENT = '{{$.json_escape[1]}}'


# This method is aliased to "create_custom_training_job_from_component" for
# better readability
def create_custom_training_job_op_from_component(
    component_spec: Callable,  # pylint: disable=g-bare-generic
    display_name: Optional[str] = '',
    replica_count: Optional[int] = 1,
    machine_type: Optional[str] = 'n1-standard-4',
    accelerator_type: Optional[str] = '',
    accelerator_count: Optional[int] = 1,
    boot_disk_type: Optional[str] = 'pd-ssd',
    boot_disk_size_gb: Optional[int] = 100,
    timeout: Optional[str] = '604800s',
    restart_job_on_worker_restart: Optional[bool] = False,
    service_account: Optional[str] = '',
    network: Optional[str] = '',
    encryption_spec_key_name: Optional[str] = '',
    tensorboard: Optional[str] = '',
    enable_web_access: Optional[bool] = False,
    reserved_ip_ranges: Optional[Sequence[str]] = None,
    nfs_mounts: Optional[Sequence[Dict[str, str]]] = None,
    base_output_directory: Optional[str] = '',
    labels: Optional[Dict[str, str]] = None,
) -> Callable:  # pylint: disable=g-bare-generic
  """Create a component spec that runs a custom training in Vertex AI.

  This utility converts a given component to a CustomTrainingJobOp that runs a
  custom training in Vertex AI. This simplifies the creation of custom training
  jobs. All Inputs and Outputs of the supplied component will be copied over to
  the constructed training job.

  Note that this utility constructs a ClusterSpec where the master and all the
  workers use the same spec, meaning all disk/machine spec related parameters
  will apply to all replicas. This is suitable for use cases such as training
  with MultiWorkerMirroredStrategy or Mirrored Strategy.

  This component does not support Vertex AI Python training application.

  For more details on Vertex AI Training service, please refer to
  https://cloud.google.com/vertex-ai/docs/training/create-custom-job

  Args:
    component_spec: The task (ContainerOp) object to run as Vertex AI custom
      job.
    display_name (Optional[str]): The name of the custom job. If not provided
      the component_spec.name will be used instead.
    replica_count (Optional[int]): The count of instances in the cluster. One
      replica always counts towards the master in worker_pool_spec[0] and the
      remaining replicas will be allocated in worker_pool_spec[1]. For more
      details see
      https://cloud.google.com/vertex-ai/docs/training/distributed-training#configure_a_distributed_training_job.
    machine_type (Optional[str]): The type of the machine to run the custom job.
      The default value is "n1-standard-4".  For more details about this input
      config, see
      https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types.
    accelerator_type (Optional[str]): The type of accelerator(s) that may be
      attached to the machine as per accelerator_count.  For more details about
      this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec#acceleratortype.
    accelerator_count (Optional[int]): The number of accelerators to attach to
      the machine. Defaults to 1 if accelerator_type is set.
    boot_disk_type (Optional[str]):
      Type of the boot disk (default is "pd-ssd"). Valid values: "pd-ssd"
        (Persistent Disk Solid State Drive) or "pd-standard" (Persistent Disk
        Hard Disk Drive). boot_disk_type is set as a static value and cannot be
        changed as a pipeline parameter.
    boot_disk_size_gb (Optional[int]): Size in GB of the boot disk (default is
      100GB). boot_disk_size_gb is set as a static value and cannot be
        changed as a pipeline parameter.
    timeout (Optional[str]): The maximum job running time. The default is 7
      days. A duration in seconds with up to nine fractional digits, terminated
      by 's', for example: "3.5s".
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
    network (Optional[str]): The full name of the Compute Engine network to
      which the job should be peered. For example,
      projects/12345/global/networks/myVPC. Format is of the form
      projects/{project}/global/networks/{network}. Where {project} is a project
      number, as in 12345, and {network} is a network name. Private services
      access must already be configured for the network. If left unspecified,
      the job is not peered with any network.
    encryption_spec_key_name (Optional[str]): Customer-managed encryption key
      options for the CustomJob. If this is set, then all resources created by
      the CustomJob will be encrypted with the provided encryption key.
    tensorboard (Optional[str]): The name of a Vertex AI Tensorboard resource to
      which this CustomJob will upload Tensorboard logs.
    enable_web_access (Optional[bool]): Whether you want Vertex AI to enable
      [interactive shell
        access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell)
          to training containers. If set to `true`, you can access interactive
          shells at the URIs given by [CustomJob.web_access_uris][].
    reserved_ip_ranges (Optional[Sequence[str]]): A list of names for the
      reserved ip ranges under the VPC network that can be used for this job. If
      set, we will deploy the job within the provided ip ranges. Otherwise, the
      job will be deployed to any ip ranges under the provided VPC network.
    nfs_mounts (Optional[Sequence[Dict]]): A list of NFS mount specs in Json
      dict format. nfs_mounts is set as a static value and cannot be changed as
      a pipeline parameter. For API spec, see
      https://cloud.devsite.corp.google.com/vertex-ai/docs/reference/rest/v1/CustomJobSpec#NfsMount
        For more details about mounting NFS for CustomJob, see
      https://cloud.devsite.corp.google.com/vertex-ai/docs/training/train-nfs-share
    base_output_directory (Optional[str]): The Cloud Storage location to store
      the output of this CustomJob or
      HyperparameterTuningJob. see below for more details:
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination
    labels (Optional[Dict[str, str]]): The labels with user-defined metadata to
      organize CustomJobs.
      See https://goo.gl/xmQnxf for more information.

  Returns:
    A Custom Job component operator corresponding to the input component
    operator.

  """
  worker_pool_specs = {}
  input_specs = []
  output_specs = []

  # pytype: disable=attribute-error

  if component_spec.component_spec.inputs:
    input_specs = component_spec.component_spec.inputs
  if component_spec.component_spec.outputs:
    output_specs = component_spec.component_spec.outputs

  def _is_output_parameter(output_key: str) -> bool:
    for output in component_spec.component_spec.outputs:
      if output.name == output_key:
        return type_utils.is_parameter_type(output.type)
    return False

  worker_pool_spec = {
      'machine_spec': {
          'machine_type': machine_type
      },
      'replica_count': 1,
      'container_spec': {
          'image_uri':
              component_spec.component_spec.implementation.container.image,
      }
  }
  if component_spec.component_spec.implementation.container.command:
    container_command_copy = component_spec.component_spec.implementation.container.command.copy(
    )
    dsl_utils.resolve_cmd_lines(container_command_copy, _is_output_parameter)
    # Replace executor place holder with the json escaped placeholder.
    for idx, val in enumerate(container_command_copy):
      if val == '{{{{$}}}}':
        container_command_copy[idx] = _EXECUTOR_PLACE_HOLDER_REPLACEMENT
    worker_pool_spec['container_spec']['command'] = container_command_copy

  if component_spec.component_spec.implementation.container.env:
    worker_pool_spec['container_spec'][
        'env'] = component_spec.component_spec.implementation.container.env.copy(
        )

  if component_spec.component_spec.implementation.container.args:
    container_args_copy = component_spec.component_spec.implementation.container.args.copy(
    )
    dsl_utils.resolve_cmd_lines(container_args_copy, _is_output_parameter)
    # Replace executor place holder with the json escaped placeholder.
    for idx, val in enumerate(container_args_copy):
      if val == '{{{{$}}}}':
        container_args_copy[idx] = _EXECUTOR_PLACE_HOLDER_REPLACEMENT
    worker_pool_spec['container_spec']['args'] = container_args_copy
  if accelerator_type:
    worker_pool_spec['machine_spec']['accelerator_type'] = accelerator_type
    worker_pool_spec['machine_spec']['accelerator_count'] = accelerator_count
  if boot_disk_type:
    if 'disk_spec' not in worker_pool_spec:
      worker_pool_spec['disk_spec'] = {}
    worker_pool_spec['disk_spec']['boot_disk_type'] = boot_disk_type
    if 'disk_spec' not in worker_pool_spec:
      worker_pool_spec['disk_spec'] = {}
    worker_pool_spec['disk_spec']['boot_disk_size_gb'] = boot_disk_size_gb
  if nfs_mounts:
    if 'nfs_mounts' not in worker_pool_spec:
      worker_pool_spec['nfs_mounts'] = []
    worker_pool_spec['nfs_mounts'].extend(nfs_mounts)

  worker_pool_specs = [worker_pool_spec]
  if int(replica_count) > 1:
    additional_worker_pool_spec = copy.deepcopy(worker_pool_spec)
    additional_worker_pool_spec['replica_count'] = str(replica_count - 1)
    worker_pool_specs.append(additional_worker_pool_spec)

  # Remove any Vertex Training duplicate input_spec from component input list.
  input_specs[:] = [
      input_spec for input_spec in input_specs
      if input_spec.name not in ('project', 'location', 'display_name',
                                 'worker_pool_specs', 'timeout',
                                 'restart_job_on_worker_restart',
                                 'service_account', 'tensorboard', 'network',
                                 'reserved_ip_ranges', 'nfs_mounts',
                                 'base_output_directory', 'labels',
                                 'encryption_spec_key_name')
  ]

  custom_training_job_json = None
  with open(os.path.join(os.path.dirname(__file__), 'component.yaml')) as file:
    custom_training_job_json = yaml.load(file, Loader=yaml.FullLoader)

  for input_item in custom_training_job_json['inputs']:
    if 'display_name' in input_item.values():
      input_item[
          'default'] = display_name if display_name else component_spec.component_spec.name
      input_item['optional'] = True
    elif 'worker_pool_specs' in input_item.values():
      input_item['default'] = json.dumps(worker_pool_specs)
      input_item['optional'] = True
    elif 'timeout' in input_item.values():
      input_item['default'] = timeout
      input_item['optional'] = True
    elif 'restart_job_on_worker_restart' in input_item.values():
      input_item['default'] = json.dumps(restart_job_on_worker_restart)
      input_item['optional'] = True
    elif 'service_account' in input_item.values():
      input_item['default'] = service_account
      input_item['optional'] = True
    elif 'tensorboard' in input_item.values():
      input_item['default'] = tensorboard
      input_item['optional'] = True
    elif 'enable_web_access' in input_item.values():
      input_item['default'] = json.dumps(enable_web_access)
      input_item['optional'] = True
    elif 'network' in input_item.values():
      input_item['default'] = network
      input_item['optional'] = True
    elif 'reserved_ip_ranges' in input_item.values():
      input_item['default'] = json.dumps(
          reserved_ip_ranges) if reserved_ip_ranges else '[]'
      input_item['optional'] = True
    elif 'base_output_directory' in input_item.values():
      input_item['default'] = base_output_directory
      input_item['optional'] = True
    elif 'labels' in input_item.values():
      input_item['default'] = json.dumps(labels) if labels else '{}'
      input_item['optional'] = True
    elif 'encryption_spec_key_name' in input_item.values():
      input_item['default'] = encryption_spec_key_name
      input_item['optional'] = True
    else:
      # This field does not need to be updated.
      continue

  # Copying over the input and output spec from the given component.
  for input_spec in input_specs:
    custom_training_job_json['inputs'].append(input_spec.to_dict())

  for output_spec in output_specs:
    custom_training_job_json['outputs'].append(output_spec.to_dict())

  # Copy the component name and description
  custom_training_job_json['name'] = component_spec.component_spec.name

  if component_spec.component_spec.description:
    # TODO(chavoshi) Add support for docstring parsing.
    component_description = 'A custom job that wraps '
    component_description += f'{component_spec.component_spec.name}.\n\nOriginal component'
    component_description += f' description:\n{component_spec.component_spec.description}\n\nCustom'
    component_description += ' Job wrapper description:\n'
    component_description += custom_training_job_json['description']
    custom_training_job_json['description'] = component_description

  component_path = tempfile.mktemp()
  with open(component_path, 'w') as out_file:
    yaml.dump(custom_training_job_json, out_file)

  return components.load_component_from_file(component_path)
