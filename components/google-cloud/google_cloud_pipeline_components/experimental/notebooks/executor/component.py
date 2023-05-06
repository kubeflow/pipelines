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

from typing import Optional

from kfp import dsl


@dsl.container_component
def notebooks_executor(
    project: str,
    input_notebook_file: str,
    output_notebook_folder: str,
    execution_id: str,
    location: str,
    master_type: str,
    container_image_uri: str,
    state: dsl.OutputPath(str),
    output_notebook_file: dsl.OutputPath(str),
    gcp_resources: dsl.OutputPath(str),
    error: dsl.OutputPath(str),
    accelerator_type: Optional[str] = None,
    accelerator_core_count: Optional[str] = '0',
    labels: Optional[str] = 'src=notebooks_executor_api',
    params_yaml_file: Optional[str] = None,
    parameters: Optional[str] = None,
    service_account: Optional[str] = None,
    job_type: Optional[str] = 'VERTEX_AI',
    kernel_spec: Optional[str] = 'python3',
    block_pipeline: Optional[bool] = True,
    fail_pipeline: Optional[bool] = True,
):
    # fmt: off
    """
  Executes a notebook using the Notebooks Executor API.

  The component uses the same inputs as the Notebooks Executor API and additional
  ones for blocking and failing the pipeline.

  Args:
    project (str):
      Project to run the execution.
    input_notebook_file: str
      Path to the notebook file to execute.
    output_notebook_folder: str
      Path to the notebook folder to write to.
    execution_id: str
      Unique identificator for the execution.
    location: str
      Region to run the
    master_type: str
      Type of virtual machine to use for training job's master worker.
    accelerator_type: str
      Type of accelerator.
    accelerator_core_count: str
      Count of cores of the accelerator.
    labels: str
      Labels for execution.
    container_image_uri: str
      Container Image URI to a DLVM Example: 'gcr.io/deeplearning-platform-release/base-cu100'.
    params_yaml_file: str
      File with parameters to be overridden in the `inputNotebookFile` during execution.
    parameters: str
      Parameters to be overriden in the `inputNotebookFile` notebook.
    service_account: str
      Email address of a service account to use when running the execution.
    job_type: str
      Type of Job to be used on this execution.
    kernel_spec: str
      Name of the kernel spec to use.
    block_pipeline: bool
      Whether to block the pipeline until the execution operation is done.
    fail_pipeline: bool
      Whether to fail the pipeline if the execution raises an error.

  Returns:
    state:str
      State of the execution. Empty if there is an error.
    output_notebook_file:str
      Path of the executed notebook. Empty if there is an error.
    error:str
      Error message if any.

  Raises:
    RuntimeError with the error message.

  """
    # fmt: on

    return dsl.ContainerSpec(
        image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b3',
        command=[
            'python3',
            '-m',
            'google_cloud_pipeline_components.container.experimental.notebooks.executor',
        ],
        args=[
            '--project',
            project,
            '--input_notebook_file',
            input_notebook_file,
            '--output_notebook_folder',
            output_notebook_folder,
            '--execution_id',
            execution_id,
            '--location',
            location,
            '--master_type',
            master_type,
            '--container_image_uri',
            container_image_uri,
            dsl.IfPresentPlaceholder(
                input_name='accelerator_type',
                then=['--accelerator_type', accelerator_type],
            ),
            dsl.IfPresentPlaceholder(
                input_name='accelerator_core_count',
                then=['--accelerator_core_count', accelerator_core_count],
            ),
            dsl.IfPresentPlaceholder(
                input_name='labels', then=['--labels', labels]),
            dsl.IfPresentPlaceholder(
                input_name='params_yaml_file',
                then=['--params_yaml_file', params_yaml_file],
            ),
            dsl.IfPresentPlaceholder(
                input_name='parameters', then=['--parameters', parameters]),
            dsl.IfPresentPlaceholder(
                input_name='service_account',
                then=['--service_account', service_account],
            ),
            dsl.IfPresentPlaceholder(
                input_name='job_type', then=['--job_type', job_type]),
            dsl.IfPresentPlaceholder(
                input_name='kernel_spec', then=['--kernel_spec', kernel_spec]),
            dsl.IfPresentPlaceholder(
                input_name='block_pipeline',
                then=['--block_pipeline', block_pipeline],
            ),
            dsl.IfPresentPlaceholder(
                input_name='fail_pipeline',
                then=['--fail_pipeline', fail_pipeline],
            ),
            '----output-paths',
            state,
            output_notebook_file,
            gcp_resources,
            error,
        ],
    )
