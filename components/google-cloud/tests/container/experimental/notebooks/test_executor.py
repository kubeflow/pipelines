import json
import os
import time
import unittest

from absl.testing import parameterized
from logging import raiseExceptions
from google_cloud_pipeline_components.container.experimental.notebooks import executor
from google.cloud import notebooks
from google.api_core import operation
from google.cloud.notebooks import Execution
from google.cloud import aiplatform as vertex_ai
from google.cloud.aiplatform import CustomJob
from google.cloud import aiplatform_v1beta1 as vertex_ai_beta
from google.cloud.aiplatform.compat.types import job_state
from types import SimpleNamespace
from unittest import mock

_MOCK_PROJECT_ID = 'mock-project-id'
_MOCK_LOCATION = 'mock-location'
_MOCK_NOTEBOOK_FILE = 'gs://mock-bucket/mock-input-notebook-file.ipynb'
_MOCK_OUTPUT_NOTEBOOK_FOLDER = 'gs://mock-output-notebook-folder'
_MOCK_EXECUTION_ID = 'mock-execution-id'
_MOCK_MASTER_TYPE = 'mock-master-type'
_MOCK_CONTAINER_IMAGE_URI = 'gcr-mock.io/mock-project/mock-image'

class TestNotebookExecutor(unittest.TestCase):

  def setUp(self):
    super(TestNotebookExecutor, self).setUp()
    # ('input_notebook_file')
    self._test_build_execution_template = [
      (None),
      (_MOCK_NOTEBOOK_FILE),
    ]
    # ('block_pipeline', 'expected_state', 'mock_execution_state')
    self._param_test_block_pipeline = [
      (False, Execution.State.PREPARING, Execution.State.PREPARING),
      (True, Execution.State.SUCCEEDED, Execution.State.SUCCEEDED),
    ]
    # ('mock_get_custom_job', 'mock_get_execution', 'fail_pipeline')
    self._param_test_fail_pipeline = [
      (Execution.State.FAILED, Execution.State.FAILED, True),
      (Execution.State.FAILED, Execution.State.FAILED, False),
    ]

  def test_build_execution_template(self):
    mock_args = SimpleNamespace(
        project=_MOCK_PROJECT_ID,
        location=_MOCK_LOCATION,
        output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
        execution_id=_MOCK_EXECUTION_ID,
        master_type=_MOCK_MASTER_TYPE,
        container_image_uri=_MOCK_CONTAINER_IMAGE_URI,
    )

    for input_notebook_file in self._test_build_execution_template:
      with self.subTest():
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
          with self.assertRaises(AttributeError):
            _ = executor.build_execution_template(mock_args)

  @mock.patch.object(notebooks.NotebookServiceClient, 'create_execution', autospec=True)
  @mock.patch.object(notebooks.NotebookServiceClient, 'get_execution', autospec=True)
  def test_block_pipeline(
      self,
      mock_get_execution,
      mock_create_execution):

    for block_pipeline, expected_state, mock_execution_state in self._param_test_block_pipeline:
      with self.subTest():

        notebooks_client_create = mock.Mock()
        mock_create_execution.return_value = notebooks_client_create

        notebooks_client_get = mock.Mock(state=mock_execution_state)
        mock_get_execution.return_value = notebooks_client_get

        mock_args = SimpleNamespace(
            project=_MOCK_PROJECT_ID,
            location=_MOCK_LOCATION,
            input_notebook_file=_MOCK_NOTEBOOK_FILE,
            output_notebook_folder=_MOCK_OUTPUT_NOTEBOOK_FOLDER,
            execution_id=_MOCK_EXECUTION_ID,
            master_type=_MOCK_MASTER_TYPE,
            container_image_uri=_MOCK_CONTAINER_IMAGE_URI,
            block_pipeline=block_pipeline,
        )
        state, _, _, error = executor.execute_notebook(args=mock_args)
        assert state == Execution.State(expected_state).name
        assert error == ''

  @mock.patch.object(notebooks.NotebookServiceClient, 'create_execution', autospec=True)
  @mock.patch.object(notebooks.NotebookServiceClient, 'get_execution', autospec=True)
  @mock.patch.object(vertex_ai_beta.JobServiceClient, 'get_custom_job', autospec=True)
  def test_fail_pipeline(
      self,
      mock_get_custom_job,
      mock_get_execution,
      mock_create_execution):

    for get_custom_job_mock_state, get_execution_mock_state, fail_pipeline in self._param_test_fail_pipeline:
      with self.subTest():
        notebooks_client_create = mock.Mock()
        mock_create_execution.return_value = notebooks_client_create

        notebooks_client_get = mock.Mock(state=get_execution_mock_state)
        mock_get_execution.return_value = notebooks_client_get

        vertexai_client_get = mock.Mock(
            state=get_custom_job_mock_state,
            error='mock-error')
        mock_get_custom_job.return_value = vertexai_client_get

        mock_args = SimpleNamespace(
            project=_MOCK_PROJECT_ID,
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
          with self.assertRaises(RuntimeError):
            _, _, _, _ = executor.execute_notebook(args=mock_args)
        else:
          _, _, _, error = executor.execute_notebook(args=mock_args
          )
          assert error == 'Execution finished with state: FAILED'
