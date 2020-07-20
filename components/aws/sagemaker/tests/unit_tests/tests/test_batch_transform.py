import unittest
import os
import signal

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

from batch_transform.src import batch_transform
from common import _utils


# TODO : Errors out if model_name doesn't contain '-'
# fix model_name '-' bug

required_args = [
  '--region', 'us-west-2',
  '--model_name', 'model-test',
  '--input_location', 's3://fake-bucket/data',
  '--output_location', 's3://fake-bucket/output',
  '--instance_type', 'ml.c5.18xlarge',
  '--output_location_output_path', '/tmp/output'
]

class BatchTransformTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = batch_transform.create_parser()
    cls.parser = parser


  def test_create_parser(self):
    self.assertIsNotNone(self.parser)


  def test_main(self):
    # Mock out all of utils except parser
    batch_transform._utils = MagicMock()
    batch_transform._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    batch_transform._utils.create_transform_job.return_value = 'test-batch-job'

    batch_transform.main(required_args)

    # Check if correct requests were created and triggered
    batch_transform._utils.create_transform_job.assert_called()
    batch_transform._utils.wait_for_transform_job.assert_called()
    batch_transform._utils.print_logs_for_job.assert_called()


    # Check the file outputs
    batch_transform._utils.write_output.assert_has_calls([
      call('/tmp/output', 's3://fake-bucket/output')
    ])


  def test_batch_transform(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + ['--job_name', 'test-batch-job'])
    response = _utils.create_transform_job(mock_client, vars(mock_args))

    mock_client.create_transform_job.assert_called_once_with(
      DataProcessing={'InputFilter': '', 'OutputFilter': '', 'JoinSource': 'None'},
      Environment={},
      MaxConcurrentTransforms=0,
      MaxPayloadInMB=6,
      ModelName='model-test',
      Tags=[],
      TransformInput={'DataSource': {'S3DataSource': {'S3DataType': 'S3Prefix', 'S3Uri': 's3://fake-bucket/data'}},
                      'ContentType': '', 'CompressionType': 'None', 'SplitType': 'None'},
      TransformJobName='test-batch-job',
      TransformOutput={'S3OutputPath': 's3://fake-bucket/output', 'Accept': None, 'KmsKeyId': ''},
      TransformResources={'InstanceType': 'ml.c5.18xlarge', 'InstanceCount': None, 'VolumeKmsKeyId': ''}
    )

    self.assertEqual(response, 'test-batch-job')


  def test_pass_all_arguments(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + [
                          '--job_name', 'test-batch-job',
                          '--max_concurrent', '5',
                          '--max_payload', '100',
                          '--batch_strategy', 'MultiRecord',
                          '--data_type', 'S3Prefix',
                          '--compression_type', 'Gzip',
                          '--split_type', 'RecordIO',
                          '--assemble_with', 'Line',
                          '--join_source', 'Input',
                          '--tags', '{"fake_key": "fake_value"}'
                ])
    response = _utils.create_transform_job(mock_client, vars(mock_args))

    mock_client.create_transform_job.assert_called_once_with(
      BatchStrategy='MultiRecord',
      DataProcessing={'InputFilter': '', 'OutputFilter': '',
                      'JoinSource': 'Input'},
      Environment={},
      MaxConcurrentTransforms=5,
      MaxPayloadInMB=100,
      ModelName='model-test',
      Tags=[{'Key': 'fake_key', 'Value': 'fake_value'}],
      TransformInput={
          'DataSource': {'S3DataSource': {'S3DataType': 'S3Prefix',
                         'S3Uri': 's3://fake-bucket/data'}},
          'ContentType': '',
          'CompressionType': 'Gzip',
          'SplitType': 'RecordIO',
          },
      TransformJobName='test-batch-job',
      TransformOutput={
          'S3OutputPath': 's3://fake-bucket/output',
          'Accept': None,
          'AssembleWith': 'Line',
          'KmsKeyId': '',
          },
      TransformResources={'InstanceType': 'ml.c5.18xlarge',
                          'InstanceCount': None, 'VolumeKmsKeyId': ''}
    )

  def test_main_stop_tranform_job(self):
    batch_transform._utils = MagicMock()
    batch_transform._utils.create_transform_job.return_value = 'test-batch-job'

    try:
      os.kill(os.getpid(), signal.SIGTERM)
    finally:
      batch_transform._utils.stop_transform_job.assert_called_once_with(ANY, 'test-batch-job')
      batch_transform._utils.print_logs_for_job.assert_not_called()

  def test_utils_stop_transform_job(self):
    mock_sm_client = MagicMock()
    mock_sm_client.stop_transform_job.return_value = None

    response = _utils.stop_transform_job(mock_sm_client, 'FakeJobName')

    mock_sm_client.stop_transform_job.assert_called_once_with(
        TransformJobName='FakeJobName'
    )
    self.assertEqual(response, None)

  def test_sagemaker_exception_in_batch_transform(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "batch_transform")
    mock_client.create_transform_job.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      _utils.create_transform_job(mock_client, vars(mock_args))


  def test_wait_for_transform_job_creation(self):
    mock_client = MagicMock()
    mock_client.describe_transform_job.side_effect = [
      {"TransformJobStatus": "InProgress"},
      {"TransformJobStatus": "Completed"},
      {"TransformJobStatus": "Should not be called"}
    ]

    _utils.wait_for_transform_job(mock_client, 'test-batch', 0)
    self.assertEqual(mock_client.describe_transform_job.call_count, 2)


  def test_wait_for_failed_job(self):
    mock_client = MagicMock()
    mock_client.describe_transform_job.side_effect = [
      {"TransformJobStatus": "InProgress"},
      {"TransformJobStatus": "Failed", "FailureReason": "SYSTEM FAILURE"},
      {"TransformJobStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_transform_job(mock_client, 'test-batch', 0)

    self.assertEqual(mock_client.describe_transform_job.call_count, 2)


