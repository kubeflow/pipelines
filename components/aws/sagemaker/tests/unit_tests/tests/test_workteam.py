import json
import unittest

from unittest.mock import patch, Mock, MagicMock
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

  def test_sample(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_workteam_request(vars(args))
    self.assertEqual(response['WorkteamName'], 'test-team')

  def test_empty_string(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_workteam_request(vars(args))
    test_utils.check_empty_string_values(response)