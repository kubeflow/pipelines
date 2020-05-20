import json
import unittest

from unittest.mock import patch, call, Mock, MagicMock, mock_open
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

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    ground_truth._utils = MagicMock()
    ground_truth._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    ground_truth._utils.get_labeling_job_outputs.return_value = ('s3://fake-bucket/output', 'arn:aws:sagemaker:us-east-1:999999999999:labeling-job')

    with patch('builtins.open', mock_open()) as file_open:
      ground_truth.main(required_args)

    # Check if correct requests were created and triggered
    ground_truth._utils.create_labeling_job.assert_called()
    ground_truth._utils.wait_for_labeling_job.assert_called()
    ground_truth._utils.get_labeling_job_outputs.assert_called()

    # Check the file outputs
    file_open.assert_has_calls([
      call('/tmp/output_manifest_location.txt', 'w'),
      call('/tmp/active_learning_model_arn.txt', 'w')
    ], any_order=True)

    file_open().write.assert_has_calls([
      call('s3://fake-bucket/output'),
      call('arn:aws:sagemaker:us-east-1:999999999999:labeling-job')
    ], any_order=False)

  def test_ground_truth(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args)
    response = _utils.create_labeling_job(mock_client, vars(mock_args))

    mock_client.create_labeling_job.assert_called_once_with(
      HumanTaskConfig={'WorkteamArn': None, 'UiConfig': {'UiTemplateS3Uri': 's3://fake-bucket/ui_template'},
                       'PreHumanTaskLambdaArn': '', 'TaskTitle': 'fake-image-labelling-work',
                       'TaskDescription': 'fake job', 'NumberOfHumanWorkersPerDataObject': 1,
                       'TaskTimeLimitInSeconds': 180,
                       'AnnotationConsolidationConfig': {'AnnotationConsolidationLambdaArn': ''}},
      InputConfig={'DataSource': {'S3DataSource': {'ManifestS3Uri': 's3://fake-bucket/manifest'}}},
      LabelAttributeName='test_job', LabelCategoryConfigS3Uri='', LabelingJobName='test_job',
      OutputConfig={'S3OutputPath': 's3://fake-bucket/output', 'KmsKeyId': ''},
      RoleArn='arn:aws:iam::123456789012:user/Development/product_1234/*', Tags=[]
    )

    self.assertEqual(response, 'test_job')

  def test_sagemaker_exception_in_ground_truth(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "ground_truth")
    mock_client.create_labeling_job.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      _utils.get_labeling_job_outputs(mock_client, vars(mock_args))

  def test_wait_for_labeling_job_creation(self):
    mock_client = MagicMock()
    mock_client.describe_labeling_job.side_effect = [
      {"LabelingJobStatus": "InProgress"},
      {"LabelingJobStatus": "Completed"},
      {"LabelingJobStatus": "Should not be called"}
    ]

    _utils.wait_for_labeling_job(mock_client, 'test-batch', 0)
    self.assertEqual(mock_client.describe_labeling_job.call_count, 2)

  def test_wait_for_labeling_job_creation(self):
    mock_client = MagicMock()
    mock_client.describe_labeling_job.side_effect = [
      {"LabelingJobStatus": "InProgress"},
      {"LabelingJobStatus": "Failed"},
      {"LabelingJobStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_labeling_job(mock_client, 'test-batch', 0)
    self.assertEqual(mock_client.describe_labeling_job.call_count, 2)

  def test_get_labeling_job_output_from_job(self):
    mock_client = MagicMock()
    mock_client.describe_labeling_job.return_value = {"LabelingJobOutput": {
                                                          "OutputDatasetS3Uri": "s3://path/",
                                                          "FinalActiveLearningModelArn": "fake-arn"
                                                      }}

    output_manifest, active_learning_model_arn = _utils.get_labeling_job_outputs(mock_client, 'labeling-job', True)
    self.assertEqual(output_manifest, 's3://path/')
    self.assertEqual(active_learning_model_arn, 'fake-arn')