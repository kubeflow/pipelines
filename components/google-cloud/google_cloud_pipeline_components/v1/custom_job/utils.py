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
import logging
from typing import Callable, Dict, Optional, Sequence

from google_cloud_pipeline_components.v1.custom_job import component
from kfp import components
import yaml

from google.protobuf import json_format

_EXECUTOR_PLACEHOLDER = '{{$}}'
# Executor replacement is used as executor content needs to be jsonified before
# injection into the payload, since payload is already a JSON serialized string.
_EXECUTOR_PLACEHOLDER_REPLACEMENT = '{{$.json_escape[1]}}'


def _replace_executor_placeholder(
    container_input: Sequence[str],
) -> Sequence[str]:
  """Replace executor placeholder in container command or args.

  Args:
    container_input: Container command or args

  Returns:
    container input with executor placeholder replaced.
  """
  return [
      _EXECUTOR_PLACEHOLDER_REPLACEMENT
      if input == _EXECUTOR_PLACEHOLDER
      else input
      for input in container_input
  ]


def create_custom_training_job_from_component(
    component_spec: Callable,  # pylint: disable=g-bare-generic
    display_name: str = '',
    replica_count: int = 1,
    machine_type: str = 'n1-standard-4',
    accelerator_type: str = '',
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
    reserved_ip_ranges: Optional[Sequence[str]] = None,
    nfs_mounts: Optional[Sequence[Dict[str, str]]] = None,
    base_output_directory: str = '',
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
    boot_disk_type (Optional[str]): Type of the boot disk (default is "pd-ssd").
      Valid values: "pd-ssd" (Persistent Disk Solid State Drive) or
      "pd-standard" (Persistent Disk Hard Disk Drive). boot_disk_type is set as
      a static value and cannot be changed as a pipeline parameter.
    boot_disk_size_gb (Optional[int]): Size in GB of the boot disk (default is
      100GB). boot_disk_size_gb is set as a static value and cannot be changed
      as a pipeline parameter.
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
      the output of this CustomJob or HyperparameterTuningJob. see below for
      more details:
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/GcsDestination
    labels (Optional[Dict[str, str]]): The labels with user-defined metadata to
      organize CustomJobs. See https://goo.gl/xmQnxf for more information.

  Returns:
    A Custom Job component operator corresponding to the input component
    operator.
  """
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

  custom_training_job_dict = json_format.MessageToDict(
      component.custom_training_job.pipeline_spec
  )

  input_component_spec_dict = json_format.MessageToDict(
      component_spec.pipeline_spec
  )
  component_spec_container = list(
      input_component_spec_dict['deploymentSpec']['executors'].values()
  )[0]['container']

  # Construct worker_pool_spec
  worker_pool_spec = {
      'machine_spec': {'machine_type': machine_type},
      'replica_count': 1,
      'container_spec': {
          'image_uri': component_spec_container['image'],
      },
  }
  worker_pool_spec['container_spec']['command'] = _replace_executor_placeholder(
      component_spec_container.get('command', [])
  )
  worker_pool_spec['container_spec']['args'] = _replace_executor_placeholder(
      component_spec_container.get('args', [])
  )

  if accelerator_type:
    worker_pool_spec['machine_spec']['accelerator_type'] = accelerator_type
    worker_pool_spec['machine_spec']['accelerator_count'] = accelerator_count
  if boot_disk_type:
    worker_pool_spec['disk_spec'] = {
        'boot_disk_type': boot_disk_type,
        'boot_disk_size_gb': boot_disk_size_gb,
    }
  if nfs_mounts:
    worker_pool_spec['nfs_mounts'] = nfs_mounts.copy()

  worker_pool_specs = [worker_pool_spec]

  if int(replica_count) > 1:
    additional_worker_pool_spec = copy.deepcopy(worker_pool_spec)
    additional_worker_pool_spec['replica_count'] = replica_count - 1
    worker_pool_specs.append(additional_worker_pool_spec)

  # Retrieve the custom job input/output parameters
  custom_training_job_dict_components = custom_training_job_dict['components']
  custom_training_job_comp_key = list(
      custom_training_job_dict_components.keys()
  )[0]
  custom_training_job_comp_val = custom_training_job_dict_components[
      custom_training_job_comp_key
  ]
  custom_job_input_params = custom_training_job_comp_val['inputDefinitions'][
      'parameters'
  ]
  custom_job_output_params = custom_training_job_comp_val['outputDefinitions'][
      'parameters'
  ]

  # Insert input arguments into custom_job_input_params as default values
  custom_job_input_params['display_name']['defaultValue'] = (
      display_name or component_spec.component_spec.name
  )
  custom_job_input_params['worker_pool_specs'][
      'defaultValue'
  ] = worker_pool_specs
  custom_job_input_params['timeout']['defaultValue'] = timeout
  custom_job_input_params['restart_job_on_worker_restart'][
      'defaultValue'
  ] = restart_job_on_worker_restart
  custom_job_input_params['service_account']['defaultValue'] = service_account
  custom_job_input_params['tensorboard']['defaultValue'] = tensorboard
  custom_job_input_params['enable_web_access'][
      'defaultValue'
  ] = enable_web_access
  custom_job_input_params['network']['defaultValue'] = network
  custom_job_input_params['reserved_ip_ranges']['defaultValue'] = (
      reserved_ip_ranges or []
  )
  custom_job_input_params['base_output_directory'][
      'defaultValue'
  ] = base_output_directory
  custom_job_input_params['labels']['defaultValue'] = labels or {}
  custom_job_input_params['encryption_spec_key_name'][
      'defaultValue'
  ] = encryption_spec_key_name

  # Merge with the input/output parameters from the input component.
  input_component_spec_comp_val = list(
      input_component_spec_dict['components'].values()
  )[0]
  custom_job_input_params = {
      **(
          input_component_spec_comp_val.get('inputDefinitions', {}).get(
              'parameters', {}
          )
      ),
      **custom_job_input_params,
  }
  custom_job_output_params = {
      **(
          input_component_spec_comp_val.get('outputDefinitions', {}).get(
              'parameters', {}
          )
      ),
      **custom_job_output_params,
  }

  # Copy merged input/output parameters to custom_training_job_dict
  # Using copy.deepcopy here to avoid anchors and aliases in the produced
  # YAML as a result of pointing to the same dict.
  custom_training_job_dict['root']['inputDefinitions']['parameters'] = (
      copy.deepcopy(custom_job_input_params)
  )
  custom_training_job_dict['components'][custom_training_job_comp_key][
      'inputDefinitions'
  ]['parameters'] = copy.deepcopy(custom_job_input_params)
  custom_training_job_tasks_key = list(
      custom_training_job_dict['root']['dag']['tasks'].keys()
  )[0]
  custom_training_job_dict['root']['dag']['tasks'][
      custom_training_job_tasks_key
  ]['inputs']['parameters'] = {
      **(
          list(input_component_spec_dict['root']['dag']['tasks'].values())[0]
          .get('inputs', {})
          .get('parameters', {})
      ),
      **(
          custom_training_job_dict['root']['dag']['tasks'][
              custom_training_job_tasks_key
          ]['inputs']['parameters']
      ),
  }
  custom_training_job_dict['components'][custom_training_job_comp_key][
      'outputDefinitions'
  ]['parameters'] = custom_job_output_params

  # Retrieve the input/output artifacts from the input component.
  custom_job_input_artifacts = input_component_spec_comp_val.get(
      'inputDefinitions', {}
  ).get('artifacts', {})
  custom_job_output_artifacts = input_component_spec_comp_val.get(
      'outputDefinitions', {}
  ).get('artifacts', {})

  # Copy input/output artifacts from the input component to
  # custom_training_job_dict
  if custom_job_input_artifacts:
    custom_training_job_dict['root']['inputDefinitions']['artifacts'] = (
        copy.deepcopy(custom_job_input_artifacts)
    )
    custom_training_job_dict['components'][custom_training_job_comp_key][
        'inputDefinitions'
    ]['artifacts'] = copy.deepcopy(custom_job_input_artifacts)
    custom_training_job_dict['root']['dag']['tasks'][
        custom_training_job_tasks_key
    ]['inputs']['artifacts'] = {
        **(
            list(input_component_spec_dict['root']['dag']['tasks'].values())[0]
            .get('inputs', {})
            .get('artifacts', {})
        ),
        **(
            custom_training_job_dict['root']['dag']['tasks'][
                custom_training_job_tasks_key
            ]['inputs'].get('artifacts', {})
        ),
    }
  if custom_job_output_artifacts:
    custom_training_job_dict['components'][custom_training_job_comp_key][
        'outputDefinitions'
    ]['artifacts'] = custom_job_output_artifacts

  # Create new component from component IR YAML
  custom_training_job_yaml = yaml.safe_dump(custom_training_job_dict)
  new_component = components.load_component_from_text(custom_training_job_yaml)

  # Copy the component name and description
  # TODO(b/262360354): The inner .component_spec.name is needed here as that is
  # the name that is retrieved by the FE for display. Can simply reference the
  # outer .name once setter is implemented.
  new_component.component_spec.name = component_spec.component_spec.name

  if component_spec.description:
    # TODO(chavoshi) Add support for docstring parsing.
    component_description = 'A custom job that wraps '
    component_description += (
        f'{component_spec.component_spec.name}.\n\nOriginal component'
    )
    component_description += (
        f' description:\n{component_spec.description}\n\nCustom'
    )
    component_description += ' Job wrapper description:\n'
    component_description += component.custom_training_job.description

    new_component.description = component_description

  return new_component


# This alias points to the old "create_custom_training_job_op_from_component" to
# avoid potential user breakage.
def create_custom_training_job_op_from_component(*args, **kwargs) -> Callable:  # pylint: disable=g-bare-generic
  """Deprecated.

  Please use create_custom_training_job_from_component instead.

  Args:
    *args: Positional arguments for create_custom_training_job_from_component.
    **kwargs: Keyword arguments for create_custom_training_job_from_component.

  Returns:
    A Custom Job component operator corresponding to the input component
    operator.
  """

  logging.warning(
      'Deprecated. Please use create_custom_training_job_from_component'
      ' instead.'
  )
  return create_custom_training_job_from_component(*args, **kwargs)
