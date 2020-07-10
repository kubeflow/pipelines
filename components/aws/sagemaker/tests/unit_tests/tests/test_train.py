import unittest
import os
import signal

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

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
  '--model_artifact_path', 'test-path',
  '--model_artifact_url_output_path', '/tmp/model_artifact_url_output_path',
  '--job_name_output_path', '/tmp/job_name_output_path',
  '--training_image_output_path', '/tmp/training_image_output_path',
]

class TrainTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = train.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    train._utils = MagicMock()
    train._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    train._utils.create_training_job.return_value = 'job-name'
    train._utils.get_image_from_job.return_value = 'training-image'
    train._utils.get_model_artifacts_from_job.return_value = 'model-artifacts'

    train.main(required_args)

    # Check if correct requests were created and triggered
    train._utils.create_training_job.assert_called()
    train._utils.wait_for_training_job.assert_called()
    train._utils.print_logs_for_job.assert_called()

    # Check the file outputs
    train._utils.write_output.assert_has_calls([
      call('/tmp/model_artifact_url_output_path', 'model-artifacts'),
      call('/tmp/job_name_output_path', 'job-name'),
      call('/tmp/training_image_output_path', 'training-image')
    ])

  def test_create_training_job(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + ['--job_name', 'test-job'])
    response = _utils.create_training_job(mock_client, vars(mock_args))

    mock_client.create_training_job.assert_called_once_with(
      AlgorithmSpecification={'TrainingImage': 'test-image', 'TrainingInputMode': 'File'},
      EnableInterContainerTrafficEncryption=False,
      EnableManagedSpotTraining=False,
      EnableNetworkIsolation=True,
      HyperParameters={},
      InputDataConfig=[{'ChannelName': 'train',
                        'DataSource': {'S3DataSource': {'S3Uri': 's3://fake-bucket/data', 'S3DataType': 'S3Prefix', 'S3DataDistributionType': 'FullyReplicated'}},
                        'ContentType': '',
                        'CompressionType': 'None',
                        'RecordWrapperType': 'None',
                        'InputMode': 'File'
                      }],
      OutputDataConfig={'KmsKeyId': '', 'S3OutputPath': 'test-path'},
      ResourceConfig={'InstanceType': 'ml.m4.xlarge', 'InstanceCount': 1, 'VolumeSizeInGB': 50, 'VolumeKmsKeyId': ''},
      RoleArn='arn:aws:iam::123456789012:user/Development/product_1234/*',
      StoppingCondition={'MaxRuntimeInSeconds': 3600},
      Tags=[],
      TrainingJobName='test-job'
    )
    self.assertEqual(response, 'test-job')

  def test_main_stop_training_job(self):
    train._utils = MagicMock()
    train._utils.create_training_job.return_value = 'job-name'

    try:
      os.kill(os.getpid(), signal.SIGTERM)
    finally:
      train._utils.stop_training_job.assert_called_once_with(ANY, 'job-name')
      train._utils.get_image_from_job.assert_not_called()

  def test_utils_stop_training_job(self):
    mock_sm_client = MagicMock()
    mock_sm_client.stop_training_job.return_value = None

    response = _utils.stop_training_job(mock_sm_client, 'FakeJobName')

    mock_sm_client.stop_training_job.assert_called_once_with(
        TrainingJobName='FakeJobName'
    )
    self.assertEqual(response, None)

  def test_sagemaker_exception_in_create_training_job(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "create_training_job")
    mock_client.create_training_job.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      response = _utils.create_training_job(mock_client, vars(mock_args))

  def test_wait_for_training_job(self):
    mock_client = MagicMock()
    mock_client.describe_training_job.side_effect = [
      {"TrainingJobStatus": "Starting"},
      {"TrainingJobStatus": "InProgress"},
      {"TrainingJobStatus": "Downloading"},
      {"TrainingJobStatus": "Completed"},
      {"TrainingJobStatus": "Should not be called"}
    ]

    _utils.wait_for_training_job(mock_client, 'training-job', 0)
    self.assertEqual(mock_client.describe_training_job.call_count, 4)

  def test_wait_for_failed_job(self):
    mock_client = MagicMock()
    mock_client.describe_training_job.side_effect = [
      {"TrainingJobStatus": "Starting"},
      {"TrainingJobStatus": "InProgress"},
      {"TrainingJobStatus": "Downloading"},
      {"TrainingJobStatus": "Failed", "FailureReason": "Something broke lol"},
      {"TrainingJobStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_training_job(mock_client, 'training-job', 0)

    self.assertEqual(mock_client.describe_training_job.call_count, 4)

  def test_get_model_artifacts_from_job(self):
    mock_client = MagicMock()
    mock_client.describe_training_job.return_value = {"ModelArtifacts": {"S3ModelArtifacts": "s3://path/"}}

    self.assertEqual(_utils.get_model_artifacts_from_job(mock_client, 'training-job'), 's3://path/')

  def test_get_image_from_defined_job(self):
    mock_client = MagicMock()
    mock_client.describe_training_job.return_value = {"AlgorithmSpecification": {"TrainingImage": "training-image-url"}}

    self.assertEqual(_utils.get_image_from_job(mock_client, 'training-job'), "training-image-url")

  def test_get_image_from_algorithm_job(self):
    mock_client = MagicMock()
    mock_client.describe_training_job.return_value = {"AlgorithmSpecification": {"AlgorithmName": "my-algorithm"}}
    mock_client.describe_algorithm.return_value = {"TrainingSpecification": {"TrainingImage": "training-image-url"}}

    self.assertEqual(_utils.get_image_from_job(mock_client, 'training-job'), "training-image-url")

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

  def test_no_defined_image(self):
    # Pass the image to pass the parser
    no_image_args = required_args.copy()
    image_index = no_image_args.index('--image')
    # Cut out --image and it's associated value
    no_image_args = no_image_args[:image_index] + no_image_args[image_index+2:]

    parsed_args = self.parser.parse_args(no_image_args)

    with self.assertRaises(Exception):
      _utils.create_training_job_request(vars(parsed_args))

  def test_first_party_algorithm(self):
    algorithm_name_args = self.parser.parse_args(required_args + ['--algorithm_name', 'first-algorithm'])

    # Should not throw an exception
    response = _utils.create_training_job_request(vars(algorithm_name_args))
    self.assertIn('TrainingImage', response['AlgorithmSpecification'])
    self.assertNotIn('AlgorithmName', response['AlgorithmSpecification'])

  def test_known_algorithm_key(self):
    # This passes an algorithm that is a known NAME of an algorithm
    known_algorithm_args = required_args + ['--algorithm_name', 'seq2seq modeling']
    image_index = required_args.index('--image')
    # Cut out --image and it's associated value
    known_algorithm_args = known_algorithm_args[:image_index] + known_algorithm_args[image_index+2:]

    parsed_args = self.parser.parse_args(known_algorithm_args)

    # Patch get_image_uri
    _utils.get_image_uri = MagicMock()
    _utils.get_image_uri.return_value = "seq2seq-url"

    response = _utils.create_training_job_request(vars(parsed_args))

    _utils.get_image_uri.assert_called_with('us-west-2', 'seq2seq')
    self.assertEqual(response['AlgorithmSpecification']['TrainingImage'], "seq2seq-url")

  def test_known_algorithm_value(self):
    # This passes an algorithm that is a known SageMaker algorithm name
    known_algorithm_args = required_args + ['--algorithm_name', 'seq2seq']
    image_index = required_args.index('--image')
    # Cut out --image and it's associated value
    known_algorithm_args = known_algorithm_args[:image_index] + known_algorithm_args[image_index+2:]

    parsed_args = self.parser.parse_args(known_algorithm_args)

    # Patch get_image_uri
    _utils.get_image_uri = MagicMock()
    _utils.get_image_uri.return_value = "seq2seq-url"

    response = _utils.create_training_job_request(vars(parsed_args))

    _utils.get_image_uri.assert_called_with('us-west-2', 'seq2seq')
    self.assertEqual(response['AlgorithmSpecification']['TrainingImage'], "seq2seq-url")

  def test_unknown_algorithm(self):
    known_algorithm_args = required_args + ['--algorithm_name', 'unknown algorithm']
    image_index = required_args.index('--image')
    # Cut out --image and it's associated value
    known_algorithm_args = known_algorithm_args[:image_index] + known_algorithm_args[image_index+2:]

    parsed_args = self.parser.parse_args(known_algorithm_args)

    # Patch get_image_uri
    _utils.get_image_uri = MagicMock()
    _utils.get_image_uri.return_value = "unknown-url"

    response = _utils.create_training_job_request(vars(parsed_args))

    # Should just place the algorithm name in regardless
    _utils.get_image_uri.assert_not_called()
    self.assertEqual(response['AlgorithmSpecification']['AlgorithmName'], "unknown algorithm")

  def test_no_channels(self):
    no_channels_args = required_args.copy()
    channels_index = required_args.index('--channels')
    # Replace the value after the flag with an empty list
    no_channels_args[channels_index + 1] = '[]'
    parsed_args = self.parser.parse_args(no_channels_args)

    with self.assertRaises(Exception):
      _utils.create_training_job_request(vars(parsed_args))


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

  def test_tags(self):
    args = self.parser.parse_args(required_args + ['--tags', '{"key1": "val1", "key2": "val2"}'])
    response = _utils.create_training_job_request(vars(args))
    self.assertIn({'Key': 'key1', 'Value': 'val1'}, response['Tags'])
    self.assertIn({'Key': 'key2', 'Value': 'val2'}, response['Tags'])
