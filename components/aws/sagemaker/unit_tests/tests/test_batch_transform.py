import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from batch_transform.src import batch_transform
from common import _utils

required_args = [
  '--region', 'us-west-2',
  '--model_name', 'test_model',
  '--input_location', 's3://fake-bucket/data',
  '--output_location', 's3://fake-bucket/output',
  '--instance_type', 'ml.c5.18xlarge',
  '--output_location_file', 'tmp/'
]

class BatchTransformTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = batch_transform.create_parser()
    cls.parser = parser

  def test_sample(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_hyperparameter_tuning_job_request(vars(args))
    self.assertEqual(response['TransformOutput']['S3OutputPath'], 's3://fake-uri/output')