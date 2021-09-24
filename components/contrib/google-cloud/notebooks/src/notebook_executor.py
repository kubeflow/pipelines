#!/usr/bin/env python
# Copyright 2021 Google LLC
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

"""Kubeflow Pipeline component for running notebooks as a step using.

Uses the Notebooks Executor API:
https://cloud.google.com/notebooks/docs/reference/rest#rest-resource:-v1.projects.locations.executions.
"""

from typing import NamedTuple, Optional  # pylint: disable=unused-import
from kfp.components import create_component_from_func


def execute_notebook(
    project_id: str,
    input_notebook_file: str,
    output_notebook_folder: str,
    execution_id: str,
    location: Optional[str] = 'us-central1',
    master_type: Optional[str] = 'n1-standard-4',
    accelerator_type: Optional[str] = None,
    accelerator_core_count: Optional[str] = '0',
    labels: Optional[str] = 'src=notebooks_executor_api',
    container_image_uri:
    Optional[str] = 'gcr.io/deeplearning-platform-release/base-cpu:latest',
    params_yaml_file: Optional[str] = None,
    parameters: Optional[str] = None,
    block_pipeline: Optional[bool] = True,
    fail_pipeline: Optional[bool] = True
) -> NamedTuple(
    'Outputs',
    [
        ('state', str),
        ('output_notebook_file', str),
        ('error', str),
    ],
):
  """Executes a notebook using the Notebooks Executor API.

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
  import time  # pylint: disable=g-import-not-at-top
  from typing import NamedTuple  # pylint: disable=g-import-not-at-top,redefined-outer-name,reimported
  from google.cloud import notebooks
  from google.cloud import aiplatform as vertex_ai
  from google.cloud import aiplatform_v1beta1 as vertex_ai_beta
  from google.cloud.aiplatform.compat.types import job_state
  from google.cloud.notebooks import Execution

  _STATES_COMPLETED = (
      job_state.JobState.JOB_STATE_SUCCEEDED,
      job_state.JobState.JOB_STATE_FAILED,
      job_state.JobState.JOB_STATE_CANCELLED,
      job_state.JobState.JOB_STATE_PAUSED,
      Execution.State.SUCCEEDED,
      Execution.State.FAILED,
      Execution.State.CANCELLING,
      Execution.State.CANCELLED,
  )

  _STATES_ERROR = (
      job_state.JobState.JOB_STATE_FAILED,
      job_state.JobState.JOB_STATE_CANCELLED,
      Execution.State.FAILED,
      Execution.State.CANCELLED
  )

  _STATES_JOB_URI = (
    Execution.State.PREPARING,
    Execution.State.RUNNING,
  )

  client_notebooks = notebooks.NotebookServiceClient()
  client_vertexai_jobs = vertex_ai_beta.JobServiceClient(
      client_options={'api_endpoint': f'{location}-aiplatform.googleapis.com'})

  # ----------------------------------
  # Helpers
  # ----------------------------------
  def _check_prefix(s, prefix='gs://'):
    """Adds the prefix gs:// to a GCS path when missing."""
    if not s.startswith(prefix):
      s = f'{prefix}{s}'
    return s

  def _handle_error(error_response):
    """Build the error logic.

    Manages how errors behave based on the fail_pipeline pipeline parameter. Can
    either fails the pipeline by raising an error or silently return the error.

    Args:
      error_response: tuple Object to return when not failing the pipeline.

    Returns:
      A tuple matching error_response with the error message generally at the
      last index if fail_pipeline is False.

    Raises:
      RuntimeError: with the error message if fail_pipeline is True.
    """
    if fail_pipeline:
      error_message = error_response[-1]
      raise RuntimeError(error_message)
    return error_response

  def _build_execution():
    """Builds the body object for the Notebooks Executor API."""
    betpl = {}
    betpl['master_type'] = master_type
    betpl['accelerator_config'] = {}
    betpl['input_notebook_file'] = _check_prefix(input_notebook_file)
    betpl['container_image_uri'] = container_image_uri
    betpl['output_notebook_folder'] = _check_prefix(output_notebook_folder)
    if labels:
      betpl['labels'] = dict(l.split('=') for l in labels.split(','))
    if accelerator_type:
      betpl['accelerator_config']['type'] = accelerator_type
      betpl['accelerator_config']['core_count'] = accelerator_core_count
    if params_yaml_file:
      betpl['params_yaml_file'] = params_yaml_file
    if parameters:
      betpl['parameters'] = parameters
    body = {}
    body['execution_template'] = betpl
    body['description'] = f'Executor for notebook {input_notebook_file}'
    return body

  # ----------------------------------
  # Component logic
  # ----------------------------------

  # Creates execution.
  execution = _build_execution()
  try:
    print('Try create_execution()...')
    execution_create_operation = client_notebooks.create_execution(
      parent=f'projects/{project_id}/locations/{location}',
      execution_id=execution_id,
      execution=execution)
  except Exception as e:
    response = ('', '', f'create_execution() failed: {e}')
    _handle_error(response)
    return response

  # Gets initial execution
  try:
    print('Try get_execution()...')
    execution = client_notebooks.get_execution(
        name=f'projects/{project_id}/locations/{location}/executions/{execution_id}')
  except Exception as e:
    response = ('', '', f'get_execution() failed: {e}')
    _handle_error(response)
    return response

  if not block_pipeline:
    print('Not blocking pipeline...')
    return (Execution.State(execution.state).name, execution.output_notebook_file, '')

  # Waits for execution to finish.
  print('Blocking pipeline...')
  execution_state = ''
  execution_job_uri = ''
  while True:
    try:
      execution = client_notebooks.get_execution(
        name=f'projects/{project_id}/locations/{location}/executions/{execution_id}')
      execution_state = getattr(execution, 'state', '')
      print(f'execution.state is {Execution.State(execution_state).name}')
    except Exception as e:
      response = ('', '', f'get_execution() for blocking pipeline failed: {e}')
      _handle_error(response)
      return response

    # Job URI is not available when state is INITIALIZING.
    if execution_state in _STATES_JOB_URI and not execution_job_uri:
      execution_job_uri = getattr(execution, 'job_uri', '')
      print(f'execution.job_uri is {execution_job_uri}')

    if execution_state in _STATES_COMPLETED:
      break
    time.sleep(30)

  # For some reason, execution.get and job.get might not return the same state
  # so we check if we can get the error message using the AI Plaform API.
  # It is only possible if there is job_uri available though.
  if execution_state in _STATES_ERROR:
    if execution_job_uri:
      while True:
        try:
          custom_job = client_vertexai_jobs.get_custom_job(name=execution_job_uri)
        except Exception as e:
          response = ('', '', f'get_custom_job() failed: {e}')
          _handle_error(response)
          return response

        custom_job_state = getattr(custom_job, 'state', None)
        if custom_job_state in _STATES_COMPLETED:
          break
        time.sleep(30)

      # == to `if state in _JOB_ERROR_STATES`
      custom_job_error = getattr(custom_job, 'error', None)
      if custom_job_error:
        response = ('', '', f'Error {custom_job_error.code}: {custom_job_error.message}')
        _handle_error((None, response))
        return response

    # The job might be successful but we need to address that the execution
    # had a problem. The previous loop was in hope to find the error message,
    # we didn't have any so we return the execution state as the message.
    response = ('', '', f'Execution finished with state: {Execution.State(execution_state).name}')
    _handle_error((None, response))
    return response

  return (Execution.State(execution_state).name, execution.output_notebook_file, '')

if __name__ == '__main__':
  notebooks_executor_op = create_component_from_func(
    execute_notebook,
    base_image='python:3.8',
    packages_to_install=['google-cloud-notebooks','google-cloud-aiplatform'],
    output_component_file='component.yaml'
  )