# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Module for executing notebooks using the Notebooks Executor API."""

import json
import time
from google.api_core import gapic_v1
from google.cloud import aiplatform_v1beta1 as vertex_ai_beta
from google.cloud import notebooks
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
    Execution.State.CANCELLED,
)

_STATES_JOB_URI = (
    Execution.State.PREPARING,
    Execution.State.RUNNING,
)


def build_execution_template(args):
  """Builds the body object for the Notebooks Executor API."""

  def _check_prefix(s, prefix='gs://'):
    """Adds the prefix gs:// to a GCS path when missing."""
    if not s.startswith(prefix):
      s = f'{prefix}{s}'
    return s

  # Does not provide defaults to match with the API's design.
  # Default values are set in component.yaml.
  if (not getattr(args, 'master_type', None) or
      not getattr(args, 'input_notebook_file', None) or
      not getattr(args, 'container_image_uri', None) or
      not getattr(args, 'output_notebook_folder', None) or
      not getattr(args, 'job_type', None) or
      not getattr(args, 'kernel_spec', None)):
    raise AttributeError('Missing a required argument for the API.')

  betpl = {}
  betpl['master_type'] = args.master_type
  betpl['accelerator_config'] = {}
  betpl['input_notebook_file'] = _check_prefix(args.input_notebook_file)
  betpl['container_image_uri'] = args.container_image_uri
  betpl['output_notebook_folder'] = _check_prefix(args.output_notebook_folder)
  if getattr(args, 'labels', None):
    betpl['labels'] = dict(l.split('=') for l in args.labels.split(','))
  if getattr(args, 'accelerator_type', None):
    betpl['accelerator_config']['type'] = args.accelerator_type
    betpl['accelerator_config']['core_count'] = args.accelerator_core_count
  if getattr(args, 'params_yaml_file', None):
    betpl['params_yaml_file'] = args.params_yaml_file
  if getattr(args, 'parameters', None):
    betpl['parameters'] = args.parameters
  if getattr(args, 'service_account', None):
    betpl['service_account'] = args.service_account
  if getattr(args, 'job_type', None):
    betpl['job_type'] = args.job_type
  if getattr(args, 'kernel_spec', None):
    betpl['kernel_spec'] = args.kernel_spec
  body = {}
  body['execution_template'] = betpl
  body['description'] = f'Executor for notebook {args.input_notebook_file}'
  return body


def build_response(state='',
                   output_notebook_file='',
                   gcp_resources='',
                   error=''):
  return (state, output_notebook_file, gcp_resources, error)


def handle_error(raise_error, error_response):
  """Builds the error logic.

  Manages how errors behave based on the fail_pipeline pipeline parameter. Can
  either fails the pipeline by raising an error or silently return the error.

  Args:
    raise_error: error to raise if any.
    error_response: tuple Object to return when not failing the pipeline.

  Returns:
    A tuple matching error_response with the error message generally at the
    last index if fail_pipeline is False.

  Raises:
    RuntimeError: with the error message if fail_pipeline is True.
  """
  if raise_error:
    error_message = error_response[-1]
    raise RuntimeError(error_message)
  return error_response


def execute_notebook(args):
  """Executes a notebook."""

  client_info = gapic_v1.client_info.ClientInfo(
      user_agent='google-cloud-pipeline-components',)

  client_notebooks = notebooks.NotebookServiceClient(client_info=client_info)
  client_vertexai_jobs = vertex_ai_beta.JobServiceClient(
      client_options={
          'api_endpoint': f'{args.location}-aiplatform.googleapis.com'
      },
      client_info=client_info)

  execution_parent = f'projects/{args.project}/locations/{args.location}'
  execution_fullname = f'{execution_parent}/executions/{args.execution_id}'
  execution_template = build_execution_template(args)
  gcp_resources = ''

  try:
    print('Try create_execution()...')
    _ = client_notebooks.create_execution(
        parent=execution_parent,
        execution_id=args.execution_id,
        execution=execution_template)
    gcp_resources = json.dumps({
        'resources': [{
            'resourceType':
                'type.googleapis.com/google.cloud.notebooks.v1.Execution',
            'resourceUri':
                execution_fullname
        },]
    })
  # pylint: disable=broad-except
  except Exception as e:
    response = build_response(error=f'create_execution() failed: {e}')
    handle_error(args.fail_pipeline, response)
    return response

  # Gets initial execution
  try:
    print('Try get_execution()...')
    execution = client_notebooks.get_execution(name=execution_fullname)
  # pylint: disable=broad-except
  except Exception as e:
    response = build_response(error=f'get_execution() failed: {e}')
    handle_error(args.fail_pipeline, response)
    return response

  if not args.block_pipeline:
    print('Not blocking pipeline...')
    return build_response(
        state=Execution.State(execution.state).name,
        output_notebook_file=execution.output_notebook_file,
        gcp_resources=gcp_resources)

  # Waits for execution to finish.
  print('Blocking pipeline...')
  execution_state = ''
  execution_job_uri = ''
  while True:
    try:
      execution = client_notebooks.get_execution(name=execution_fullname)
      execution_state = getattr(execution, 'state', '')
      print(f'execution.state is {Execution.State(execution_state).name}')
    # pylint: disable=broad-except
    except Exception as e:
      response = build_response(
          error=f'get_execution() for blocking pipeline failed: {e}')
      handle_error(args.fail_pipeline, response)
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
          custom_job = client_vertexai_jobs.get_custom_job(
              name=execution_job_uri)
        # pylint: disable=broad-except
        except Exception as e:
          response = build_response(error=f'get_custom_job() failed: {e}')
          handle_error(args.fail_pipeline, response)
          return response

        custom_job_state = getattr(custom_job, 'state', None)
        if custom_job_state in _STATES_COMPLETED:
          break
        time.sleep(30)

      # == to `if state in _JOB_ERROR_STATES`
      custom_job_error = getattr(custom_job, 'error', None)
      if custom_job_error:
        response = build_response(
            error=f'Error {custom_job_error.code}: {custom_job_error.message}')
        handle_error(args.fail_pipeline, (None, response))
        return response

    # The job might be successful but we need to address that the execution
    # had a problem. The previous loop was in hope to find the error message,
    # we didn't have any so we return the execution state as the message.
    response = build_response(
        error=f'Execution finished: {Execution.State(execution_state).name}')
    handle_error(args.fail_pipeline, (None, response))
    return response

  return build_response(
      state=Execution.State(execution_state).name,
      output_notebook_file=execution.output_notebook_file,
      gcp_resources=gcp_resources)


def main():
  def _deserialize_bool(s) -> bool:
    # pylint: disable=g-import-not-at-top
    from distutils import util
    return util.strtobool(s) == 1

  def _serialize_str(str_value: str) -> str:
    if not isinstance(str_value, str):
      raise TypeError(
          f'Value "{str_value}" has type "{type(str_value)}" instead of str.')
    return str_value

  # pylint: disable=g-import-not-at-top
  import argparse
  parser = argparse.ArgumentParser(
      prog='Notebooks Executor',
      description='Executes a notebook using the Notebooks Executor API.')
  parser.add_argument(
      '--project',
      dest='project',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--input_notebook_file',
      dest='input_notebook_file',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--output_notebook_folder',
      dest='output_notebook_folder',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--execution_id',
      dest='execution_id',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--location',
      dest='location',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--master_type',
      dest='master_type',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--container_image_uri',
      dest='container_image_uri',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--accelerator_type',
      dest='accelerator_type',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--accelerator_core_count',
      dest='accelerator_core_count',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--labels',
      dest='labels',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--params_yaml_file',
      dest='params_yaml_file',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--parameters',
      dest='parameters',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--service_account',
      dest='service_account',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--job_type',
      dest='job_type',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--kernel_spec',
      dest='kernel_spec',
      type=str,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--block_pipeline',
      dest='block_pipeline',
      type=_deserialize_bool,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--fail_pipeline',
      dest='fail_pipeline',
      type=_deserialize_bool,
      required=False,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '----output-paths', dest='_output_paths', type=str, nargs=4)

  args, _ = parser.parse_known_args()
  parsed_args = vars(parser.parse_args())
  output_files = parsed_args.pop('_output_paths', [])

  outputs = execute_notebook(args)

  output_serializers = [
      _serialize_str,
      _serialize_str,
      _serialize_str,
      _serialize_str,
  ]

  import os
  for idx, output_file in enumerate(output_files):
    try:
      os.makedirs(os.path.dirname(output_file))
    except OSError:
      pass
    with open(output_file, 'w') as f:
      f.write(output_serializers[idx](outputs[idx]))


if __name__ == '__main__':
  main()
