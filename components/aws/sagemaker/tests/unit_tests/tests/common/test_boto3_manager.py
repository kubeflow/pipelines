import unittest
import os

from boto3.session import Session
from unittest.mock import patch, MagicMock, ANY

from common.boto3_manager import Boto3Manager


class Boto3ManagerTestCase(unittest.TestCase):
    def test_assume_default_boto3_session(self):
        returned_session = Boto3Manager._get_boto3_session("us-east-1")

        assert isinstance(returned_session, Session)
        assert returned_session.region_name == "us-east-1"

    @patch("common.boto3_manager.DeferredRefreshableCredentials", MagicMock())
    @patch("common.boto3_manager.AssumeRoleCredentialFetcher", MagicMock())
    def test_assume_role_boto3_session(self):
        returned_session = Boto3Manager._get_boto3_session(
            "us-east-1", role_arn="abc123"
        )

        assert isinstance(returned_session, Session)
        assert returned_session.region_name == "us-east-1"

        # Bury into the internals to ensure our provider was registered correctly
        our_provider = returned_session._session._components.get_component(
            "credential_provider"
        ).providers[0]
        assert isinstance(our_provider, Boto3Manager.AssumeRoleProvider)

    def test_assumed_sagemaker_client(self):
        Boto3Manager._get_boto3_session = MagicMock()

        mock_sm_client = MagicMock()
        # Mock the client("SageMaker", ...) return value
        Boto3Manager._get_boto3_session.return_value.client.return_value = (
            mock_sm_client
        )

        client = Boto3Manager.get_sagemaker_client(
            "v1.0.0", "us-east-1", assume_role_arn="abc123"
        )
        assert client == mock_sm_client

        Boto3Manager._get_boto3_session.assert_called_once_with("us-east-1", "abc123")
        Boto3Manager._get_boto3_session.return_value.client.assert_called_once_with(
            "sagemaker", endpoint_url=None, config=ANY, region_name="us-east-1"
        )
