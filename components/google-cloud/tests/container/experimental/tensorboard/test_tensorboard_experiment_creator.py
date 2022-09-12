"""Tests for tensorboard experiment creator."""
import os
import google.auth

from google.api_core import exceptions
from google.cloud.aiplatform.compat.services import tensorboard_service_client
from google.cloud.aiplatform.compat.types import encryption_spec as gca_encryption_spec, tensorboard as gca_tensorboard, tensorboard_experiment as gca_tensorboard_experiment
from google_cloud_pipeline_components.container.experimental.tensorboard.tensorboard_experiment_creator import main
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

from google.protobuf import json_format
import unittest
from unittest import mock

# project
_TEST_LOCATION = "us-central1"
_TEST_PROJECT_NUMBER = "119600457337"
_TEST_PARENT = f"projects/{_TEST_PROJECT_NUMBER}/locations/{_TEST_LOCATION}"
# tensorboard
_TEST_ID = "8816482365833478144"
_TEST_DISPLAY_NAME = "my_tensorboard_1234"

_TEST_TENSORBOARD_RESOURCE_NAME = f"{_TEST_PARENT}/tensorboards/{_TEST_ID}"

_TEST_TENSORBOARD_EXPERIMENT_ID = "test-experiment"
_TEST_TENSORBOARD_EXPERIMENT_RESOURCE_NAME = (
    f"{_TEST_TENSORBOARD_RESOURCE_NAME}/experiments/{_TEST_TENSORBOARD_EXPERIMENT_ID}"
)
_TEST_TENSORBOARD_EXPERIMENT_DISPLAY_NAME = "test-display-name"

# CMEK encryption
_TEST_ENCRYPTION_KEY_NAME = "key_1234"
_TEST_ENCRYPTION_SPEC = gca_encryption_spec.EncryptionSpec(
    kms_key_name=_TEST_ENCRYPTION_KEY_NAME)


class TensorboardExperimentCreatorTest(unittest.TestCase):

  def setUp(self):
    super(TensorboardExperimentCreatorTest, self).setUp()
    self._tensorboard_resource_name = _TEST_TENSORBOARD_RESOURCE_NAME
    self._tensorboard_experiment_id = _TEST_TENSORBOARD_EXPERIMENT_ID
    self._gcp_resources = os.path.join(
        os.getenv("TEST_UNDECLARED_OUTPUTS_DIR"),
        "test_file_path/test_file.txt")
    self._tensorboard_experiment_resource_name = _TEST_TENSORBOARD_EXPERIMENT_RESOURCE_NAME

  def tearDown(self):
    super(TensorboardExperimentCreatorTest, self).tearDown()
    if os.path.exists(self._gcp_resources):
      os.remove(self._gcp_resources)

  @mock.patch.object(google.auth, "default", autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "create_tensorboard_experiment",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard_experiment",
      autospec=True)
  def test_tensorboard_experiment_create_successfully(
      self, mock_get_tensorboard, mock_create_tensorboard_experiment,
      mock_get_tensorboard_experiment, mock_auth):
    creds = mock.Mock()
    creds.token = "fake_token"
    mock_auth.return_value = [creds, "project"]
    mock_get_tensorboard.return_value = gca_tensorboard.Tensorboard(
        name=_TEST_TENSORBOARD_RESOURCE_NAME,
        display_name=_TEST_DISPLAY_NAME,
        encryption_spec=_TEST_ENCRYPTION_SPEC,
    )
    mock_create_tensorboard_experiment.return_value = gca_tensorboard_experiment.TensorboardExperiment(
        name=_TEST_TENSORBOARD_EXPERIMENT_RESOURCE_NAME,
        display_name=_TEST_TENSORBOARD_EXPERIMENT_DISPLAY_NAME)
    mock_get_tensorboard_experiment.return_value = gca_tensorboard_experiment.TensorboardExperiment(
        name=_TEST_TENSORBOARD_EXPERIMENT_RESOURCE_NAME,
        display_name=_TEST_TENSORBOARD_EXPERIMENT_DISPLAY_NAME)
    main([
        "--tensorboard_resource_name", self._tensorboard_resource_name,
        "--tensorboard_experiment_id", self._tensorboard_experiment_id,
        "--gcp_resources", self._gcp_resources
    ])
    with open(self._gcp_resources) as f:
      serialized_gcp_resources = f.read()
      # Instantiate GCPResources Proto
      tensorboard_experiment_resources = json_format.Parse(
          serialized_gcp_resources, GcpResources())

      self.assertEqual(len(tensorboard_experiment_resources.resources), 1)
      self.assertEqual(
          tensorboard_experiment_resources.resources[0].resource_uri,
          self._tensorboard_experiment_resource_name)

  @mock.patch.object(google.auth, "default", autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "create_tensorboard_experiment",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard_experiment",
      autospec=True)
  def test_tensorboard_experiment_create_missing_tensorboard_name(
      self, mock_get_tensorboard, mock_create_tensorboard_experiment,
      mock_get_tensorboard_experiment, mock_auth):
    with self.assertRaises(RuntimeError):
      main([
          "--tensorboard_experiment_id", self._tensorboard_experiment_id,
          "--gcp_resources", self._gcp_resources
      ])

  @mock.patch.object(google.auth, "default", autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "create_tensorboard_experiment",
      autospec=True)
  @mock.patch.object(
      tensorboard_service_client.TensorboardServiceClient,
      "get_tensorboard_experiment",
      autospec=True)
  def test_tensorboard_experiment_create_non_existing_tensorboard(
      self, mock_get_tensorboard, mock_create_tensorboard_experiment,
      mock_get_tensorboard_experiment, mock_auth):
    mock_get_tensorboard.side_effect = exceptions.NotFound
    with self.assertRaises(RuntimeError):
      main([
          "--tensorboard_experiment_id", self._tensorboard_experiment_id,
          "--gcp_resources", self._gcp_resources
      ])
