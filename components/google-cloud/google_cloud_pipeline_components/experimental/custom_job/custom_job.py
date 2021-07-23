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
"""Module for supporting Google Vertex AI Custom Job."""

import copy
import json
import tempfile
from typing import Callable, List, Optional, Mapping, Any
from kfp import components, dsl
from kfp.dsl import dsl_utils
from kfp.dsl import type_utils

from kfp.components.structures import ComponentSpec, ContainerImplementation, ContainerSpec, InputSpec, InputValuePlaceholder, OutputSpec, OutputUriPlaceholder

_DEFAULT_CUSTOM_JOB_MACHINE_TYPE = 'n1-standard-4'


def run_as_custom_job(
    component_spec: Callable,
    display_name: Optional[str] = None,
    replica_count: Optional[int] = None,
    machine_type: Optional[str] = None,
    accelerator_type: Optional[str] = None,
    accelerator_count: Optional[int] = None,
    boot_disk_type: Optional[str] = None,
    boot_disk_size_gb: Optional[int] = None,
    timeout: Optional[str] = None,
    restart_job_on_worker_restart: Optional[bool] = None,
    service_account: Optional[str] = None,
    network: Optional[str] = None,
    worker_pool_specs: Optional[List[Mapping[str, Any]]] = None,
) -> Callable:
    """Run a pipeline task using AI Platform (Unified) custom training job.

    For detailed doc of the service, please refer to
    https://cloud.google.com/ai-platform-unified/docs/training/create-custom-job

    Args:
      component_spec: The task (ContainerOp) object to run as aiplatform custom job.
      display_name: Optional. The name of the custom job. If not provided the
        component_spec.name will be used instead.
      replica_count: Optional. The number of replicas to be split between master
        workerPoolSpec and worker workerPoolSpec. (master always has 1 replica).
      machine_type: Optional. The type of the machine to run the custom job. The
        default value is "n1-standard-4".
      accelerator_type: Optional. The type of accelerator(s) that may be attached
        to the machine as per accelerator_count. Optional.
      accelerator_count: Optional. The number of accelerators to attach to the
        machine.
      boot_disk_type: Optional. Type of the boot disk (default is "pd-ssd"). Valid
        values: "pd-ssd" (Persistent Disk Solid State Drive) or "pd-standard"
          (Persistent Disk Hard Disk Drive).
      boot_disk_size_gb: Optional. Size in GB of the boot disk (default is 100GB).
      timeout: Optional. The maximum job running time. The default is 7 days. A
        duration in seconds with up to nine fractional digits, terminated by 's'.
        Example: "3.5s"
      restart_job_on_worker_restart: Optional. Restarts the entire CustomJob if a
        worker gets restarted. This feature can be used by distributed training
        jobs that are not resilient to workers leaving and joining a job.
      service_account: Optional. Specifies the service account for workload run-as
        account.
      network: Optional. The full name of the Compute Engine network to which the
        job should be peered. For example, projects/12345/global/networks/myVPC.
      worker_pool_specs: Optional, worker_pool_specs for distributed training. this
        will overwite all other cluster configurations. For details, please see:
        https://cloud.google.com/ai-platform-unified/docs/training/distributed-training
    Returns:
      A Custom Job component OP correspoinding to the input component OP.
    """
    job_spec = {}

    if worker_pool_specs is not None:
        worker_pool_specs = copy.deepcopy(worker_pool_specs)

        def _is_output_parameter(output_key: str) -> bool:
            return output_key in (
                component_spec.component_spec.output_definitions.parameters.
                keys()
            )

        for worker_pool_spec in worker_pool_specs:
            if 'container_spec' in worker_pool_spec:
                container_spec = worker_pool_spec['container_spec']
                if 'command' in container_spec:
                    dsl_utils.resolve_cmd_lines(
                        container_spec['command'], _is_output_parameter
                    )
                if 'args' in container_spec:
                    dsl_utils.resolve_cmd_lines(
                        container_spec['args'], _is_output_parameter
                    )

            elif 'pythonPackageSpec' in worker_pool_spec:
                # For custom Python training, resolve placeholders in args only.
                python_spec = worker_pool_spec['pythonPackageSpec']
                if 'args' in python_spec:
                    dsl_utils.resolve_cmd_lines(
                        python_spec['args'], _is_output_parameter
                    )

            else:
                raise ValueError(
                    'Expect either "container_spec" or "pythonPackageSpec" in each '
                    'workerPoolSpec. Got: {}'.format(worker_pool_spec)
                )

        job_spec['worker_pool_specs'] = worker_pool_specs

    else:

        def _is_output_parameter(output_key: str) -> bool:
            for output in component_spec.component_spec.outputs:
                if output.name == output_key:
                    return type_utils.is_parameter_type(output.type)
            return False

        worker_pool_spec = {
            'machine_spec': {
                'machine_type': machine_type or _DEFAULT_CUSTOM_JOB_MACHINE_TYPE
            },
            'replica_count': 1,
            'container_spec': {
                'image_uri':
                    component_spec.component_spec.implementation.container.
                    image,
            }
        }
        if component_spec.component_spec.implementation.container.command:
            dsl_utils.resolve_cmd_lines(
                component_spec.component_spec.implementation.container.command,
                _is_output_parameter
            )
            worker_pool_spec['container_spec'][
                'command'
            ] = component_spec.component_spec.implementation.container.command

        if component_spec.component_spec.implementation.container.args:
            dsl_utils.resolve_cmd_lines(
                component_spec.component_spec.implementation.container.args,
                _is_output_parameter
            )
            worker_pool_spec['container_spec'][
                'args'
            ] = component_spec.component_spec.implementation.container.args
        if accelerator_type is not None:
            worker_pool_spec['machine_spec']['accelerator_type'
                                            ] = accelerator_type
        if accelerator_count is not None:
            worker_pool_spec['machine_spec']['accelerator_count'
                                            ] = accelerator_count
        if boot_disk_type is not None:
            if 'diskSpec' not in worker_pool_spec:
                worker_pool_spec['diskSpec'] = {}
            worker_pool_spec['diskSpec']['bootDiskType'] = boot_disk_type
        if boot_disk_size_gb is not None:
            if 'diskSpec' not in worker_pool_spec:
                worker_pool_spec['diskSpec'] = {}
            worker_pool_spec['diskSpec']['bootDiskSizeGb'] = boot_disk_size_gb

        job_spec['worker_pool_specs'] = [worker_pool_spec]
        if replica_count is not None and replica_count > 1:
            additional_worker_pool_spec = copy.deepcopy(worker_pool_spec)
            additional_worker_pool_spec['replica_count'] = str(
                replica_count - 1
            )
            job_spec['worker_pool_specs'].append(additional_worker_pool_spec)

    if timeout is not None:
        if 'scheduling' not in job_spec:
            job_spec['scheduling'] = {}
        job_spec['scheduling']['timeout'] = timeout
    if restart_job_on_worker_restart is not None:
        if 'scheduling' not in job_spec:
            job_spec['scheduling'] = {}
        job_spec['scheduling']['restartJobOnWorkerRestart'
                              ] = restart_job_on_worker_restart
    if service_account is not None:
        job_spec['serviceAccount'] = service_account
    if network is not None:
        job_spec['network'] = network

    custom_job_payload = {
        'display_name': display_name or component_spec.component_spec.name,
        'job_spec': job_spec
    }

    custom_job_component_spec = ComponentSpec(
        name=component_spec.component_spec.name,
        inputs=component_spec.component_spec.inputs.extend([
            InputSpec(name='gcp_project', type='String'),
            InputSpec(name='gcp_region', type='String')
        ],
        outputs=component_spec.component_spec.outputs.extend([
            OutputSpec(name='gcp_resources', type='gcp_resources')
        ]),
        implementation=ContainerImplementation(
            container=ContainerSpec(
                image='gcr.io/managed-pipeline-test/gcp-launcher:v7',
                command=["python", "-u", "-m", "launcher.py"],
                args=[
                    '--type',
                    'CustomJob',
                    '--gcp_project',
                    InputValuePlaceholder(input_name='gcp_project'),
                    '--gcp_region',
                    InputValuePlaceholder(input_name='gcp_region'),
                    '--payload',
                    json.dumps(custom_job_payload),
                    '--gcp_resources',
                    OutputUriPlaceholder(output_name='gcp_resources'),
                ],
            )
        )
    )
    component_path = tempfile.mktemp()
    custom_job_component_spec.save(component_path)

    return components.load_component_from_file(component_path)
