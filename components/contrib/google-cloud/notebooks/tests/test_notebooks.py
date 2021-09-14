import json
from logging import raiseExceptions
import os
import time
import pytest
import unittest
from unittest import mock
from google.cloud import notebooks
from google.cloud.notebooks import Execution
from google.cloud import aiplatform as vertex_ai
from google.cloud.aiplatform import CustomJob
from google.cloud import aiplatform_v1beta1 as vertex_ai_beta
from google.cloud.aiplatform.compat.types import job_state
from types import SimpleNamespace

from src import notebook_executor

_MOCK_PROJECT = 'mock-project'
_MOCK_NOTEBOOK_FILE = 'gs://mock-bucket/mock-input-notebook-file.ipynb'
_MOCK_OUTPUT_NOTEBOOK_FOLDER = 'gs://mock-output-notebook-folder'
_MOCK_EXECUTION_ID = 'mock-execution-id'

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
    state, notebook_output_file, error = notebook_executor.execute_notebook(
      project_id=_MOCK_PROJECT,
      input_notebook_file=_MOCK_NOTEBOOK_FILE,
      output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
      execution_id=_MOCK_EXECUTION_ID,
      block_pipeline=block_pipeline
    )
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

    if fail_pipeline:
      with pytest.raises(RuntimeError):
        notebook_executor.execute_notebook(
          project_id=_MOCK_PROJECT,
          input_notebook_file=_MOCK_NOTEBOOK_FILE,
          output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
          execution_id=_MOCK_EXECUTION_ID,
          block_pipeline=True,
          fail_pipeline=fail_pipeline
        )
    else:
      _, _, error = notebook_executor.execute_notebook(
          project_id=_MOCK_PROJECT,
          input_notebook_file=_MOCK_NOTEBOOK_FILE,
          output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
          execution_id=_MOCK_EXECUTION_ID,
          block_pipeline=True,
          fail_pipeline=fail_pipeline
        )
      assert error == 'Execution finished with state: FAILED'



