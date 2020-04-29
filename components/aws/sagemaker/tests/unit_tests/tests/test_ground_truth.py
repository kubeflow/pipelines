import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from ground_truth.src import ground_truth
from common import _utils
from . import test_utils


required_args = [
  '--region', 'us-west-2',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--job_name', 'test_job',
  '--manifest_location', 's3://fake-bucket/manifest',
  '--output_location', 's3://fake-bucket/output',
  '--task_type', 'fake-task',
  '--worker_type', 'fake_worker',
  '--ui_template', 's3://fake-bucket/ui_template',
  '--title', 'fake-image-labelling-work',
  '--description', 'fake job',
  '--num_workers_per_object', '1',
  '--time_limit', '180',
]

class GroundTruthTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = ground_truth.create_parser()
    cls.parser = parser

  def test_sample(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_labeling_job_request(vars(args))
    self.assertEqual(response['LabelingJobName'], 'test_job')

  def test_empty_string(self):
    args = self.parser.parse_args(required_args)
    response = _utils.create_labeling_job_request(vars(args))
    test_utils.check_empty_string_values(response)
