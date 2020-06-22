import json
import unittest

from unittest.mock import patch, call, Mock, MagicMock, mock_open
from botocore.exceptions import ClientError
from datetime import datetime

from workteam.src import workteam
from common import _utils
from . import test_utils


required_args = [
  '--region', 'us-west-2',
  '--team_name', 'test-team',
  '--description', 'fake team'
]

class WorkTeamTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = workteam.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    workteam._utils = MagicMock()
    workteam._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    workteam._utils.create_workteam.return_value = 'arn:aws:sagemaker:us-east-1:999999999999:work-team'

    with patch('builtins.open', mock_open()) as file_open:
      workteam.main(required_args)

    # Check if correct requests were created and triggered
    workteam._utils.create_workteam.assert_called()

    # Check the file outputs
    file_open.assert_has_calls([
      call('/tmp/workteam_arn.txt', 'w')
    ])

    file_open().write.assert_has_calls([
      call('arn:aws:sagemaker:us-east-1:999999999999:work-team')
    ])

  def test_workteam(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args)
    response = _utils.create_workteam(mock_client, vars(mock_args))

    mock_client.create_workteam.assert_called_once_with(
      Description='fake team',
      MemberDefinitions=[{'CognitoMemberDefinition': {'UserPool': '', 'UserGroup': '', 'ClientId': ''}}], Tags=[],
      WorkteamName='test-team'
    )

  def test_sagemaker_exception_in_workteam(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "workteam")
    mock_client.create_workteam.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      _utils.create_workteam(mock_client, vars(mock_args))

  def test_get_workteam_output_from_job(self):
    mock_client = MagicMock()
    mock_client.create_workteam.return_value = {"WorkteamArn": "fake-arn"}

    self.assertEqual(_utils.create_workteam(mock_client, vars(self.parser.parse_args(required_args))), 'fake-arn')

  def test_pass_most_arguments(self):
    arguments = required_args + ['--sns_topic', 'fake-topic', '--tags', '{"fake_key": "fake_value"}']
    response = _utils.create_workteam_request(vars(self.parser.parse_args(arguments)))

    self.assertEqual(response, {'WorkteamName': 'test-team',
                                'MemberDefinitions': [{'CognitoMemberDefinition': {'UserPool': '', 'UserGroup': '', 'ClientId': ''}}],
                                'Description': 'fake team',
                                'NotificationConfiguration' : {'NotificationTopicArn': 'fake-topic'},
                                'Tags': [{'Key': 'fake_key', 'Value': 'fake_value'}]
                                })