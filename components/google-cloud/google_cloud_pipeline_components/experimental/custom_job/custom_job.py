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
import logging
import tempfile
from typing import Callable, List, Optional, Mapping, Any, Dict
from kfp import components
from kfp.dsl import dsl_utils
from kfp.v2.components.types import type_utils
from google_cloud_pipeline_components.aiplatform import utils
from kfp.components import structures

_DEFAULT_CUSTOM_JOB_CONTAINER_IMAGE = utils.DEFAULT_CONTAINER_IMAGE
# Using an empty placeholder instead of "{{$.json_escape[1]}}" while
# backend support has not been enabled.
_EXECUTOR_PLACE_HOLDER_REPLACEMENT = json.dumps({"outputs": {"outputFile": "tmp/temp_output_file"}})


def custom_training_job_op(
    component_spec: Callable,
    display_name: Optional[str] = "",
    replica_count: Optional[int] = 1,
    machine_type: Optional[str] = "n1-standard-4",
    accelerator_type: Optional[str] = "",
    accelerator_count: Optional[int] = 1,
    boot_disk_type: Optional[str] = "pd-ssd",
    boot_disk_size_gb: Optional[int] = 100,
    timeout: Optional[str] = "",
    restart_job_on_worker_restart: Optional[bool] = False,
    service_account: Optional[str] = "",
    network: Optional[str] = "",
    worker_pool_specs: Optional[List[Mapping[str, Any]]] = None,
    encryption_spec_key_name: Optional[str] = "",
    tensorboard: Optional[str] = "",
    base_output_directory: Optional[str] = "",
    labels: Optional[Dict[str, str]] = None,
) -> Callable:
  """Run a pipeline task using Vertex AI custom training job.

    For detailed doc of the service, please refer to
    https://cloud.google.com/vertex-ai/docs/training/create-custom-job

    Args:
      component_spec:
        The task (ContainerOp) object to run as Vertex AI custom job.
      display_name:
        Optional. The name of the custom job. If not provided the
        component_spec.name will be used instead.
      replica_count:
        Optional. The number of replicas to be split between master
        workerPoolSpec and worker workerPoolSpec. (master always has 1 replica).
      machine_type:
        Optional. The type of the machine to run the custom job. The default
        value is "n1-standard-4".

        For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types
      accelerator_type:
        Optional. The type of accelerator(s) that may be attached
        to the machine as per accelerator_count.

        For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec#acceleratortype
      accelerator_count:
        Optional. The number of accelerators to attach to the machine. Defaults
        to 1 if accelerator_type is set.
      boot_disk_type:
        Optional. Type of the boot disk (default is "pd-ssd"). Valid values:
        "pd-ssd" (Persistent Disk Solid State Drive) or "pd-standard"
          (Persistent Disk Hard Disk Drive).
      boot_disk_size_gb:
        Optional. Size in GB of the boot disk (default is 100GB).
      timeout:
        Optional. The maximum job running time. The default is 7 days. A
        duration in seconds with up to nine fractional digits, terminated by 's'.
        Example: "3.5s".
      restart_job_on_worker_restart:
        Optional. Restarts the entire CustomJob if a worker gets restarted. This
        feature can be used by distributed training jobs that are not resilient
        to workers leaving and joining a job.
      service_account:
        Optional. Sets the default service account for workload run-as account.
        Users submitting jobs must have act-as permission on this run-as account.
        If unspecified, the Vertex AI Custom Code Service Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project is used.
      network:
        Optional. The full name of the Compute Engine network to which the
        job should be peered. For example, projects/12345/global/networks/myVPC.
        Format is of the form projects/{project}/global/networks/{network}.
        Where {project} is a project number, as in 12345, and {network} is a
        network name.
        Private services access must already be configured for the network.
        If left unspecified, the job is not peered with any network.
      worker_pool_specs:
        Optional, worker_pool_specs for distributed training. this
        will overwite all other cluster configurations. For details, please see:
        https://cloud.google.com/ai-platform-unified/docs/training/distributed-training
      encryption_spec_key_name:
        Optional, customer-managed encryption key options for the CustomJob.
        If this is set, then all resources created by the CustomJob will be
        encrypted with the provided encryption key.
      tensorboard:
        The name of a Vertex AI Tensorboard resource to which this CustomJob
        will upload Tensorboard logs.
      base_output_directory:
        The Cloud Storage location to store the output of this CustomJob or
        HyperparameterTuningJob. see below for more details:
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination
      labels:
        Optional. The labels with user-defined metadata to organize CustomJobs.
        See https://goo.gl/xmQnxf for more information.
    Returns:
      A Custom Job component OP correspoinding to the input component OP.
    """
  job_spec = {}
  input_specs = []
  output_specs = []
  if component_spec.component_spec.inputs:
    input_specs = component_spec.component_spec.inputs
  if component_spec.component_spec.outputs:
    output_specs = component_spec.component_spec.outputs

  if worker_pool_specs is not None:
    worker_pool_specs = copy.deepcopy(worker_pool_specs)

    def _is_output_parameter(output_key: str) -> bool:
      return output_key in (
          component_spec.component_spec.output_definitions.parameters.keys())

    for worker_pool_spec in worker_pool_specs:
      if 'container_spec' in worker_pool_spec:
        container_spec = worker_pool_spec['container_spec']
        if 'command' in container_spec:
          dsl_utils.resolve_cmd_lines(container_spec['command'],
                                      _is_output_parameter)
        if 'args' in container_spec:
          dsl_utils.resolve_cmd_lines(container_spec['args'],
                                      _is_output_parameter)
          # Temporarily remove {{{{$}}}} executor_input arg as it is not supported by the backend.
          logging.info(
              'Setting executor_input to empty, as it is currently not supported by the backend.'
          )
          for idx, val in enumerate(container_spec['args']):
            if val == '{{{{$}}}}':
              container_spec['args'][idx] = _EXECUTOR_PLACE_HOLDER_REPLACEMENT

      elif 'python_package_spec' in worker_pool_spec:
        # For custom Python training, resolve placeholders in args only.
        python_spec = worker_pool_spec['python_package_spec']
        if 'args' in python_spec:
          dsl_utils.resolve_cmd_lines(python_spec['args'], _is_output_parameter)
          # Temporarily remove {{{{$}}}} executor_input arg as it is not supported by the backend.
          logging.info(
              'Setting executor_input to empty, as it is currently not supported by the backend.'
          )
          for idx, val in enumerate(python_spec['args']):
            if val == '{{{{$}}}}':
              python_spec['args'][idx] = _EXECUTOR_PLACE_HOLDER_REPLACEMENT

      else:
        raise ValueError(
            'Expect either "container_spec" or "python_package_spec" in each '
            'workerPoolSpec. Got: {}'.format(worker_pool_spec))

    job_spec['worker_pool_specs'] = worker_pool_specs

  else:

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
      worker_pool_spec['container_spec']['command'] = container_command_copy

    if component_spec.component_spec.implementation.container.args:
      container_args_copy = component_spec.component_spec.implementation.container.args.copy(
      )
      dsl_utils.resolve_cmd_lines(container_args_copy, _is_output_parameter)
      # Temporarily remove {{{{$}}}} executor_input arg as it is not supported by the backend.
      logging.info(
          'Setting executor_input to empty, as it is currently not supported by the backend.'
          'This may result in python componnet artifacts not working correctly.'
      )
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

    job_spec['worker_pool_specs'] = [worker_pool_spec]
    if replica_count > 1:
      additional_worker_pool_spec = copy.deepcopy(worker_pool_spec)
      additional_worker_pool_spec['replica_count'] = str(replica_count - 1)
      job_spec['worker_pool_specs'].append(additional_worker_pool_spec)

  #TODO(chavoshi): Use input parameter instead of hard coded string label.
  # This requires Dictionary input type to be supported in V2.
  if labels is not None:
    job_spec['labels'] = labels

  if timeout:
    if 'scheduling' not in job_spec:
      job_spec['scheduling'] = {}
    job_spec['scheduling']['timeout'] = timeout
  if restart_job_on_worker_restart:
    if 'scheduling' not in job_spec:
      job_spec['scheduling'] = {}
    job_spec['scheduling'][
        'restart_job_on_worker_restart'] = restart_job_on_worker_restart

  if encryption_spec_key_name:
    job_spec['encryption_spec'] = {}
    job_spec['encryption_spec'][
        'kms_key_name'] = "{{$.inputs.parameters['encryption_spec_key_name']}}"
    input_specs.append(
        structures.InputSpec(
            name='encryption_spec_key_name',
            type='String',
            optional=True,
            default=encryption_spec_key_name),)

  # Remove any existing service_account from component input list.
  input_specs[:] = [
      input_spec for input_spec in input_specs
      if input_spec.name not in ('service_account', 'network', 'tensorboard',
                                 'base_output_directory')
  ]
  job_spec['service_account'] = "{{$.inputs.parameters['service_account']}}"
  job_spec['network'] = "{{$.inputs.parameters['network']}}"

  job_spec['tensorboard'] = "{{$.inputs.parameters['tensorboard']}}"
  job_spec['base_output_directory'] = {}
  job_spec['base_output_directory'][
      'output_uri_prefix'] = "{{$.inputs.parameters['base_output_directory']}}"
  custom_job_payload = {
      'display_name': display_name or component_spec.component_spec.name,
      'job_spec': job_spec
  }

  custom_job_component_spec = structures.ComponentSpec(
      name=component_spec.component_spec.name,
      inputs=input_specs + [
          structures.InputSpec(
              name='base_output_directory',
              type='String',
              optional=True,
              default=base_output_directory),
          structures.InputSpec(
              name='tensorboard',
              type='String',
              optional=True,
              default=tensorboard),
          structures.InputSpec(
              name='network', type='String', optional=True, default=network),
          structures.InputSpec(
              name='service_account',
              type='String',
              optional=True,
              default=service_account),
          structures.InputSpec(name='project', type='String'),
          structures.InputSpec(name='location', type='String')
      ],
      outputs=output_specs +
      [structures.OutputSpec(name='gcp_resources', type='String')],
      implementation=structures.ContainerImplementation(
          container=structures.ContainerSpec(
              image=_DEFAULT_CUSTOM_JOB_CONTAINER_IMAGE,
              command=[
                  'python3', '-u', '-m',
                  'google_cloud_pipeline_components.container.experimental.gcp_launcher.launcher'
              ],
              args=[
                  '--type',
                  'CustomJob',
                  '--payload',
                  json.dumps(custom_job_payload),
                  '--project',
                  structures.InputValuePlaceholder(input_name='project'),
                  '--location',
                  structures.InputValuePlaceholder(input_name='location'),
                  '--gcp_resources',
                  structures.OutputPathPlaceholder(output_name='gcp_resources'),
              ],
          )))
  component_path = tempfile.mktemp()
  custom_job_component_spec.save(component_path)

  return components.load_component_from_file(component_path)
