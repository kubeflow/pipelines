import json
from logging import raiseExceptions
import os
import time
import pytest
import unittest
from unittest import mock
from google.cloud import notebooks
from google.api_core import operation
from google.cloud.notebooks import Execution
from google.cloud import aiplatform as vertex_ai
from google.cloud.aiplatform import CustomJob
from google.cloud import aiplatform_v1beta1 as vertex_ai_beta
from google.cloud.aiplatform.compat.types import job_state
from types import SimpleNamespace

from google_cloud_pipeline_components.experimental.remote.notebooks import executor

_MOCK_PROJECT_ID = 'mock-project-id'
_MOCK_LOCATION = 'mock-location'
_MOCK_NOTEBOOK_FILE = 'gs://mock-bucket/mock-input-notebook-file.ipynb'
_MOCK_OUTPUT_NOTEBOOK_FOLDER = 'gs://mock-output-notebook-folder'
_MOCK_EXECUTION_ID = 'mock-execution-id'
_MOCK_MASTER_TYPE = 'mock-master-type'
_MOCK_CONTAINER_IMAGE_URI = 'gcr-mock.io/mock-project/mock-image'

@pytest.fixture
def create_execution_mock():
  with mock.patch.object(
    notebooks.NotebookServiceClient, "create_execution"
  ) as create_execution_mock:
    yield create_execution_mock

@pytest.fixture
def get_execution_mock(request):
  with mock.patch.object(
    notebooks.NotebookServiceClient, "get_execution"
  ) as get_execution_mock:
    get_execution_mock.return_value = Execution(
      state=request.param
    )
    yield get_execution_mock

@pytest.fixture
def get_custom_job_mock(request):
  with mock.patch.object(
    vertex_ai_beta.JobServiceClient, "get_custom_job"
  ) as get_custom_job_mock:
    get_custom_job_mock = SimpleNamespace(
      state=request.param,
      error='mock-error'
    )
    yield get_custom_job_mock

class TestNotebookExecutor:

  # Use master_type as a reference for all required fields.
  @pytest.mark.parametrize('input_notebook_file', [
      (None),
      (_MOCK_NOTEBOOK_FILE),
  ])
  def test_build_execution_template(self, input_notebook_file):
    mock_args = SimpleNamespace(
        project_id=_MOCK_PROJECT_ID,
        location=_MOCK_LOCATION,
        output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
        execution_id=_MOCK_EXECUTION_ID,
        master_type=_MOCK_MASTER_TYPE,
        container_image_uri=_MOCK_CONTAINER_IMAGE_URI,
    )
    if input_notebook_file:
      setattr(mock_args, 'input_notebook_file', input_notebook_file)
      template = executor.build_execution_template(mock_args)
      expected_template = {
        'description': f'Executor for notebook {_MOCK_NOTEBOOK_FILE}',
        'execution_template': {
          'accelerator_config': {},
          'input_notebook_file': _MOCK_NOTEBOOK_FILE,
          'output_notebook_folder': _MOCK_OUTPUT_NOTEBOOK_FOLDER,
          'master_type': _MOCK_MASTER_TYPE,
          'container_image_uri': _MOCK_CONTAINER_IMAGE_URI,
        }
      }
      assert template == expected_template
    else:
      with pytest.raises(AttributeError):
        _ = executor.build_execution_template(mock_args)


  @pytest.mark.parametrize('block_pipeline, expected_state, get_execution_mock', [
      (False, 'PREPARING', 'PREPARING'),
      (True, 'SUCCEEDED', 'SUCCEEDED'),
  ], indirect=['get_execution_mock'])
  def test_block_pipeline(
      self,
      create_execution_mock,
      get_execution_mock,
      block_pipeline,
      expected_state):

    mock_args = SimpleNamespace(
        project_id=_MOCK_PROJECT_ID,
        location=_MOCK_LOCATION,
        input_notebook_file=_MOCK_NOTEBOOK_FILE,
        output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
        execution_id=_MOCK_EXECUTION_ID,
        master_type=_MOCK_MASTER_TYPE,
        container_image_uri=_MOCK_CONTAINER_IMAGE_URI,
        block_pipeline=block_pipeline,
    )
    state, _, _, error = executor.execute_notebook(args=mock_args)
    assert state == expected_state
    assert error == ''

  @pytest.mark.parametrize('get_custom_job_mock, get_execution_mock, fail_pipeline', [
      ('FAILED', 'FAILED', True),
      ('FAILED', 'FAILED', False),
  ], indirect=['get_execution_mock', 'get_custom_job_mock'])
  def test_fail_pipeline(
      self,
      create_execution_mock,
      get_execution_mock,
      get_custom_job_mock,
      fail_pipeline):

    mock_args = SimpleNamespace(
        project_id=_MOCK_PROJECT_ID,
        location=_MOCK_LOCATION,
        input_notebook_file=_MOCK_NOTEBOOK_FILE,
        output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
        execution_id=_MOCK_EXECUTION_ID,
        master_type=_MOCK_MASTER_TYPE,
        container_image_uri=_MOCK_CONTAINER_IMAGE_URI,
        block_pipeline=True,
        fail_pipeline=fail_pipeline,
    )

    if fail_pipeline:
      with pytest.raises(RuntimeError):
        _, _, _, _ = executor.execute_notebook(args=mock_args)
    else:
      _, _, _, error = executor.execute_notebook(args=mock_args
      )
      assert error == 'Execution finished with state: FAILED'
