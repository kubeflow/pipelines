import json
import unittest
import os
import signal

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

from process.src import process
from common import _utils


required_args = [
  '--region', 'us-west-2',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--image', 'test-image',
  '--instance_type', 'ml.m4.xlarge',
  '--instance_count', '1',
  '--input_config', json.dumps([{
    'InputName': "dataset-input",
    'S3Input': {
      'S3Uri': "s3://my-bucket/dataset.csv",
      'LocalPath': "/opt/ml/processing/input",
      'S3DataType': "S3Prefix",
      'S3InputMode': "File"
    }
  }]),
  '--output_config', json.dumps([{
    'OutputName': "training-outputs",
    'S3Output': {
      'S3Uri': "s3://my-bucket/outputs/train.csv",
      'LocalPath': "/opt/ml/processing/output/train",
      'S3UploadMode': "Continuous"
    }
  }]),
  '--job_name_output_path', '/tmp/job_name_output_path',
  '--output_artifacts_output_path', '/tmp/output_artifacts_output_path'
]

class ProcessTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = process.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    process._utils = MagicMock()
    process._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    process._utils.create_processing_job.return_value = 'job-name'
    process._utils.get_processing_job_outputs.return_value = mock_outputs = {'val1': 's3://1', 'val2': 's3://2'}

    process.main(required_args)

    # Check if correct requests were created and triggered
    process._utils.create_processing_job.assert_called()
    process._utils.wait_for_processing_job.assert_called()
    process._utils.print_logs_for_job.assert_called()

    # Check the file outputs
    process._utils.write_output.assert_has_calls([
      call('/tmp/job_name_output_path', 'job-name'),
      call('/tmp/output_artifacts_output_path', mock_outputs, json_encode=True)
    ])

  def test_create_processing_job(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + ['--job_name', 'test-job'])
    response = _utils.create_processing_job(mock_client, vars(mock_args))

    mock_client.create_processing_job.assert_called_once_with(
      AppSpecification={"ImageUri": "test-image"},
      Environment={},
      NetworkConfig={
        "EnableInterContainerTrafficEncryption": False,
        "EnableNetworkIsolation": True,
      },
      ProcessingInputs=[
        {
          "InputName": "dataset-input",
          "S3Input": {
            "S3Uri": "s3://my-bucket/dataset.csv",
            "LocalPath": "/opt/ml/processing/input",
            "S3DataType": "S3Prefix",
            "S3InputMode": "File"
          },
        }
      ],
      ProcessingJobName="test-job",
      ProcessingOutputConfig={
        "Outputs": [
          {
            "OutputName": "training-outputs",
            "S3Output": {
              "S3Uri": "s3://my-bucket/outputs/train.csv",
              "LocalPath": "/opt/ml/processing/output/train",
              "S3UploadMode": "Continuous"
            },
          }
        ]
      },
      ProcessingResources={
        "ClusterConfig": {
          "InstanceType": "ml.m4.xlarge",
          "InstanceCount": 1,
          "VolumeSizeInGB": 30,
        }
      },
      RoleArn="arn:aws:iam::123456789012:user/Development/product_1234/*",
      StoppingCondition={"MaxRuntimeInSeconds": 86400},
      Tags=[],
    )
    self.assertEqual(response, 'test-job')

  def test_main_stop_processing_job(self):
    process._utils = MagicMock()
    process._utils.create_processing_job.return_value = 'job-name'

    try:
      os.kill(os.getpid(), signal.SIGTERM)
    finally:
      process._utils.stop_processing_job.assert_called_once_with(ANY, 'job-name')
      process._utils.get_processing_job_outputs.assert_not_called()

  def test_utils_stop_processing_job(self):
    mock_sm_client = MagicMock()
    mock_sm_client.stop_processing_job.return_value = None

    response = _utils.stop_processing_job(mock_sm_client, 'FakeJobName')

    mock_sm_client.stop_processing_job.assert_called_once_with(
        ProcessingJobName='FakeJobName'
    )
    self.assertEqual(response, None)

  def test_sagemaker_exception_in_create_processing_job(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "create_processing_job")
    mock_client.create_processing_job.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      response = _utils.create_processing_job(mock_client, vars(mock_args))

  def test_wait_for_processing_job(self):
    mock_client = MagicMock()
    mock_client.describe_processing_job.side_effect = [
      {"ProcessingJobStatus": "Starting"},
      {"ProcessingJobStatus": "InProgress"},
      {"ProcessingJobStatus": "Downloading"},
      {"ProcessingJobStatus": "Completed"},
      {"ProcessingJobStatus": "Should not be called"}
    ]

    _utils.wait_for_processing_job(mock_client, 'processing-job', 0)
    self.assertEqual(mock_client.describe_processing_job.call_count, 4)

  def test_wait_for_failed_job(self):
    mock_client = MagicMock()
    mock_client.describe_processing_job.side_effect = [
      {"ProcessingJobStatus": "Starting"},
      {"ProcessingJobStatus": "InProgress"},
      {"ProcessingJobStatus": "Downloading"},
      {"ProcessingJobStatus": "Failed", "FailureReason": "Something broke lol"},
      {"ProcessingJobStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_processing_job(mock_client, 'processing-job', 0)

    self.assertEqual(mock_client.describe_processing_job.call_count, 4)

  def test_reasonable_required_args(self):
    response = _utils.create_processing_job_request(vars(self.parser.parse_args(required_args)))

    # Ensure all of the optional arguments have reasonable default values
    self.assertNotIn('VpcConfig', response['NetworkConfig'])
    self.assertEqual(response['Tags'], [])
    ## TODO

  def test_no_defined_image(self):
    # Pass the image to pass the parser
    no_image_args = required_args.copy()
    image_index = no_image_args.index('--image')
    # Cut out --image and it's associated value
    no_image_args = no_image_args[:image_index] + no_image_args[image_index+2:]

    with self.assertRaises(SystemExit):
      parsed_args = self.parser.parse_args(no_image_args)

  def test_container_entrypoint(self):
    entrypoint, arguments = ['/bin/bash'], ['arg1', 'arg2']

    container_args = self.parser.parse_args(required_args + ['--container_entrypoint', json.dumps(entrypoint),
      '--container_arguments', json.dumps(arguments)])
    response = _utils.create_processing_job_request(vars(container_args))

    self.assertEqual(response['AppSpecification']['ContainerEntrypoint'], entrypoint)
    self.assertEqual(response['AppSpecification']['ContainerArguments'], arguments)

  def test_environment_variables(self):
    env_vars = {
      'key1': 'val1',
      'key2': 'val2'
    }

    environment_args = self.parser.parse_args(required_args + ['--environment', json.dumps(env_vars)])
    response = _utils.create_processing_job_request(vars(environment_args))

    self.assertEqual(response['Environment'], env_vars)

  def test_vpc_configuration(self):
    required_vpc_args = self.parser.parse_args(required_args + ['--vpc_security_group_ids', 'sg1,sg2', '--vpc_subnets', 'subnet1,subnet2'])
    response = _utils.create_processing_job_request(vars(required_vpc_args))

    self.assertIn('VpcConfig', response['NetworkConfig'])
    self.assertIn('sg1', response['NetworkConfig']['VpcConfig']['SecurityGroupIds'])
    self.assertIn('sg2', response['NetworkConfig']['VpcConfig']['SecurityGroupIds'])
    self.assertIn('subnet1', response['NetworkConfig']['VpcConfig']['Subnets'])
    self.assertIn('subnet2', response['NetworkConfig']['VpcConfig']['Subnets'])

  def test_tags(self):
    args = self.parser.parse_args(required_args + ['--tags', '{"key1": "val1", "key2": "val2"}'])
    response = _utils.create_processing_job_request(vars(args))
    self.assertIn({'Key': 'key1', 'Value': 'val1'}, response['Tags'])
    self.assertIn({'Key': 'key2', 'Value': 'val2'}, response['Tags'])

  def test_get_processing_job_output(self):
    mock_client = MagicMock()
    mock_client.describe_processing_job.return_value = {
      'ProcessingOutputConfig': {
        'Outputs': [{
            'OutputName': 'train',
            'S3Output': {
              'S3Uri': 's3://train'
            }
          },{
            'OutputName': 'valid',
            'S3Output': {
              'S3Uri': 's3://valid'
            }
          }
        ]
      }
    }

    response = _utils.get_processing_job_outputs(mock_client, 'processing-job')

    self.assertIn('train', response)
    self.assertIn('valid', response)
    self.assertEqual(response['train'], 's3://train')
    self.assertEqual(response['valid'], 's3://valid')
