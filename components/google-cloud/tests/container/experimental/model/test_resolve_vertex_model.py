"""Tests for resolving a google.VertexModel artifact."""

import json
import os
from unittest import mock

import google.auth
from google.cloud import aiplatform
from google_cloud_pipeline_components.container.experimental.model.resolve_vertex_model import main
from google_cloud_pipeline_components.proto import gcp_resources_pb2
from google.cloud.aiplatform.metadata.schema.google import artifact_schema

from google.protobuf import json_format
import unittest

PROJECT = 'test-project'
LOCATION = 'test-location'
MODEL_NAME = f'projects/{PROJECT}/locations/{LOCATION}/models/1234'
MODEL_VERSION = '456'
MODEL_NAME_WITH_VERSION = f'projects/{PROJECT}/locations/{LOCATION}/models/1234@{MODEL_VERSION}'

VERTEX_MODEL_ARTIFACT = artifact_schema.VertexModel(
    vertex_model_name=MODEL_NAME)
VERTEX_MODEL_ARTIFACT_WITH_VERSION = artifact_schema.VertexModel(
    vertex_model_name=MODEL_NAME_WITH_VERSION)


def mock_api_call(test_func):

  @mock.patch.object(google.auth, 'default', autospec=True)
  @mock.patch.object(
      aiplatform.gapic.ModelServiceClient, 'get_model', autospec=True)
  def mocked_test(self, mock_get_model, mock_auth):
    mock_creds = mock.Mock(spec=google.auth.credentials.Credentials)
    mock_creds.token = 'token'
    mock_auth.return_value = [mock_creds, 'project']
    test_func(self, mock_get_model)

  return mocked_test


class ResolveVertexModelTest(unittest.TestCase):

  def setUp(self):
    super(ResolveVertexModelTest, self).setUp()
    self._gcp_resources = os.path.join(
        os.getenv('TEST_UNDECLARED_OUTPUTS_DIR'), 'gcp_resources')
    self._project = PROJECT
    self._location = LOCATION
    self._model_name = f'projects/{self._project}/locations/{self._location}/models/1234'
    self._api_uri_prefix = f'https://{self._location}-aiplatform.googleapis.com/ui/'

    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock_api_call
  def test_resolve_model_artifact(self, mock_get_model):
    get_model_response = mock.Mock()
    get_model_response.name = MODEL_NAME
    get_model_response.version_id = MODEL_VERSION
    mock_get_model.return_value = get_model_response

    main([
        '--model_artifact_resource_name', VERTEX_MODEL_ARTIFACT.uri,
        '--gcp_resources', self._gcp_resources
    ])
    mock_get_model.assert_called_with(mock.ANY, parent=self._model_name)
