import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from model.src import create_model
from common import _utils
from . import test_utils


required_args = [
  '--region', 'us-west-2',
  '--model_name', 'model_test',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--image', 'test-image',
  '--model_artifact_url', 's3://fake-bucket/model_artifact'
]

class ModelTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = create_model.create_parser()
    cls.parser = parser

  def test_sample(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_model_request(vars(args))
    self.assertEqual(response['ModelName'], 'model_test')

  def test_empty_string(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_model_request(vars(args))
    test_utils.check_empty_string_values(response)