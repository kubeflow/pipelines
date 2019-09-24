import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from train.src import train
from common import _utils

required_args = [
  '--region', 'us-west-2',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--image', 'test-image',
  '--channels', '[{"ChannelName": "train", "DataSource": {"S3DataSource":{"S3Uri": "s3://fake-bucket/data","S3DataType":"S3Prefix","S3DataDistributionType": "FullyReplicated"}},"ContentType":"","CompressionType": "None","RecordWrapperType":"None","InputMode": "File"}]',
  '--instance_type', 'ml.m4.xlarge',
  '--instance_count', '1',
  '--volume_size', '50',
  '--max_run_time', '3600',
  '--model_artifact_path', 'test-path'
]

class TrainTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = train.create_parser()
    cls.parser = parser

  def test_spot_bad_args(self):
    no_max_wait_args = self.parser.parse_args(required_args + ['--spot_instance', 'True'])
    no_checkpoint_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600'])
    no_s3_uri_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600', '--checkpoint_config', '{}'])

    for arg in [no_max_wait_args, no_checkpoint_args, no_s3_uri_args]:
      with self.assertRaises(Exception):
        _utils.create_training_job_request(vars(arg))

  def test_spot_lesser_wait_time(self):
    args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3599', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/", "LocalPath": "local-path"}'])
    with self.assertRaises(Exception):
      _utils.create_training_job_request(vars(args))    

  def test_spot_good_args(self):
    good_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/"}'])
    response = _utils.create_training_job_request(vars(good_args))
    self.assertTrue(response['EnableManagedSpotTraining'])
    self.assertEqual(response['StoppingCondition']['MaxWaitTimeInSeconds'], 3600)
    self.assertEqual(response['CheckpointConfig']['S3Uri'], 's3://fake-uri/')

  def test_spot_local_path(self):
    args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/", "LocalPath": "local-path"}'])
    response = _utils.create_training_job_request(vars(args))
    self.assertEqual(response['CheckpointConfig']['S3Uri'], 's3://fake-uri/')
    self.assertEqual(response['CheckpointConfig']['LocalPath'], 'local-path')