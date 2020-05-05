import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from deploy.src import deploy
from common import _utils
from . import test_utils


required_args = [
  '--region', 'us-west-2',
  '--model_name_1', 'model-test'
]

class DeployTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = deploy.create_parser()
    cls.parser = parser

  def test_sample(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_endpoint_config_request(vars(args))
    self.assertEqual(response['EndpointConfigName'], 'EndpointConfig-test')

  def test_empty_string(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_endpoint_config_request(vars(args))
    test_utils.check_empty_string_values(response)