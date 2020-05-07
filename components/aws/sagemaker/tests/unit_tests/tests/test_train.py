import json
import unittest

from unittest.mock import patch, Mock, MagicMock
from botocore.exceptions import ClientError
from datetime import datetime

from train.src import train
from common import _utils
from . import test_utils

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

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_reasonable_required_args(self):
    response = _utils.create_training_job_request(vars(self.parser.parse_args(required_args)))

    # Ensure all of the optional arguments have reasonable default values
    self.assertFalse(response['EnableManagedSpotTraining'])
    self.assertDictEqual(response['HyperParameters'], {})
    self.assertNotIn('VpcConfig', response)
    self.assertNotIn('MetricDefinitions', response)
    self.assertEqual(response['Tags'], [])
    self.assertEqual(response['AlgorithmSpecification']['TrainingInputMode'], 'File')
    self.assertEqual(response['OutputDataConfig']['S3OutputPath'], 'test-path')

  def test_metric_definitions(self):
    metric_definition_args = self.parser.parse_args(required_args + ['--metric_definitions', '{"metric1": "regexval1", "metric2": "regexval2"}'])
    response = _utils.create_training_job_request(vars(metric_definition_args))

    self.assertIn('MetricDefinitions', response['AlgorithmSpecification'])
    response_metric_definitions = response['AlgorithmSpecification']['MetricDefinitions']

    self.assertEqual(response_metric_definitions, [{
      'Name': "metric1",
      'Regex': "regexval1"
    }, {
      'Name': "metric2",
      'Regex': "regexval2"
    }])

  def test_invalid_instance_type(self):
    invalid_instance_args = required_args + ['--instance_type', 'invalid-instance']

    with self.assertRaises(SystemExit):
      self.parser.parse_args(invalid_instance_args)

  def test_valid_hyperparameters(self):
    hyperparameters_str = '{"hp1": "val1", "hp2": "val2", "hp3": "val3"}'

    good_args = self.parser.parse_args(required_args + ['--hyperparameters', hyperparameters_str])
    response = _utils.create_training_job_request(vars(good_args))
    
    self.assertIn('hp1', response['HyperParameters'])
    self.assertIn('hp2', response['HyperParameters'])
    self.assertIn('hp3', response['HyperParameters'])
    self.assertEqual(response['HyperParameters']['hp1'], "val1")
    self.assertEqual(response['HyperParameters']['hp2'], "val2")
    self.assertEqual(response['HyperParameters']['hp3'], "val3")

  def test_empty_hyperparameters(self):
    hyperparameters_str = '{}'

    good_args = self.parser.parse_args(required_args + ['--hyperparameters', hyperparameters_str])
    response = _utils.create_training_job_request(vars(good_args))
    
    self.assertEqual(response['HyperParameters'], {})

  def test_object_hyperparameters(self):
    hyperparameters_str = '{"hp1": {"innerkey": "innerval"}}'

    invalid_args = self.parser.parse_args(required_args + ['--hyperparameters', hyperparameters_str])
    with self.assertRaises(Exception):
      _utils.create_training_job_request(vars(invalid_args))

  def test_vpc_configuration(self):
    required_vpc_args = self.parser.parse_args(required_args + ['--vpc_security_group_ids', 'sg1,sg2', '--vpc_subnets', 'subnet1,subnet2'])
    response = _utils.create_training_job_request(vars(required_vpc_args))
    
    self.assertIn('VpcConfig', response)
    self.assertIn('sg1', response['VpcConfig']['SecurityGroupIds'])
    self.assertIn('sg2', response['VpcConfig']['SecurityGroupIds'])
    self.assertIn('subnet1', response['VpcConfig']['Subnets'])
    self.assertIn('subnet2', response['VpcConfig']['Subnets'])

  def test_training_mode(self):
    required_vpc_args = self.parser.parse_args(required_args + ['--training_input_mode', 'Pipe'])
    response = _utils.create_training_job_request(vars(required_vpc_args))

    self.assertEqual(response['AlgorithmSpecification']['TrainingInputMode'], 'Pipe')

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

  def test_empty_string(self):
    good_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/"}'])
    response = _utils.create_training_job_request(vars(good_args))
    test_utils.check_empty_string_values(response)