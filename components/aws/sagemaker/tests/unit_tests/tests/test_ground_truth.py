import unittest
import os
import signal

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

from ground_truth.src import ground_truth
from common import _utils


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
  '--output_manifest_location_output_path', '/tmp/manifest-output',
  '--active_learning_model_arn_output_path', '/tmp/model-output'
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

    ground_truth.main(required_args)

    # Check if correct requests were created and triggered
    ground_truth._utils.create_labeling_job.assert_called()
    ground_truth._utils.wait_for_labeling_job.assert_called()
    ground_truth._utils.get_labeling_job_outputs.assert_called()

    # Check the file outputs
    ground_truth._utils.write_output.assert_has_calls([
      call('/tmp/manifest-output', 's3://fake-bucket/output'),
      call('/tmp/model-output', 'arn:aws:sagemaker:us-east-1:999999999999:labeling-job')
    ])

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
      LabelAttributeName='test_job', LabelingJobName='test_job',
      OutputConfig={'S3OutputPath': 's3://fake-bucket/output', 'KmsKeyId': ''},
      RoleArn='arn:aws:iam::123456789012:user/Development/product_1234/*', Tags=[]
    )

    self.assertEqual(response, 'test_job')

  def test_main_stop_labelling_job(self):
    ground_truth._utils = MagicMock()

    try:
      os.kill(os.getpid(), signal.SIGTERM)
    finally:
      ground_truth._utils.stop_labeling_job.assert_called_once_with(ANY, 'test_job')
      ground_truth._utils.get_labeling_job_outputs.assert_not_called()

  def test_utils_stop_labeling_job(self):
    mock_sm_client = MagicMock()
    mock_sm_client.stop_labeling_job.return_value = None

    response = _utils.stop_labeling_job(mock_sm_client, 'FakeJobName')

    mock_sm_client.stop_labeling_job.assert_called_once_with(
        LabelingJobName='FakeJobName'
    )
    self.assertEqual(response, None)

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

  def test_pass_most_args(self):
    required_args = [
      '--region', 'us-west-2',
      '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
      '--job_name', 'test_job',
      '--manifest_location', 's3://fake-bucket/manifest',
      '--output_location', 's3://fake-bucket/output',
      '--task_type', 'image classification',
      '--worker_type', 'fake_worker',
      '--ui_template', 's3://fake-bucket/ui_template',
      '--title', 'fake-image-labelling-work',
      '--description', 'fake job',
      '--num_workers_per_object', '1',
      '--time_limit', '180',
    ]
    arguments = required_args + ['--label_attribute_name', 'fake-attribute',
                                 '--max_human_labeled_objects', '10',
                                 '--max_percent_objects', '50',
                                 '--enable_auto_labeling', 'True',
                                 '--initial_model_arn', 'fake-model-arn',
                                 '--task_availibility', '30',
                                 '--max_concurrent_tasks', '10',
                                 '--task_keywords', 'fake-keyword',
                                 '--worker_type', 'public',
                                 '--no_adult_content', 'True',
                                 '--no_ppi', 'True',
                                 '--tags', '{"fake_key": "fake_value"}'
                                 ]
    response = _utils.create_labeling_job_request(vars(self.parser.parse_args(arguments)))
    print(response)
    self.assertEqual(response, {'LabelingJobName': 'test_job',
                                'LabelAttributeName': 'fake-attribute',
                                'InputConfig': {'DataSource': {'S3DataSource': {'ManifestS3Uri': 's3://fake-bucket/manifest'}},
                                                'DataAttributes': {'ContentClassifiers': ['FreeOfAdultContent', 'FreeOfPersonallyIdentifiableInformation']}},
                                'OutputConfig': {'S3OutputPath': 's3://fake-bucket/output', 'KmsKeyId': ''},
                                'RoleArn': 'arn:aws:iam::123456789012:user/Development/product_1234/*',
                                'StoppingConditions': {'MaxHumanLabeledObjectCount': 10, 'MaxPercentageOfInputDatasetLabeled': 50},
                                'LabelingJobAlgorithmsConfig': {'LabelingJobAlgorithmSpecificationArn': 'arn:aws:sagemaker:us-west-2:027400017018:labeling-job-algorithm-specification/image-classification',
                                                                'InitialActiveLearningModelArn': 'fake-model-arn',
                                                                'LabelingJobResourceConfig': {'VolumeKmsKeyId': ''}},
                                'HumanTaskConfig': {'WorkteamArn': 'arn:aws:sagemaker:us-west-2:394669845002:workteam/public-crowd/default',
                                                    'UiConfig': {'UiTemplateS3Uri': 's3://fake-bucket/ui_template'},
                                                    'PreHumanTaskLambdaArn': 'arn:aws:lambda:us-west-2:081040173940:function:PRE-ImageMultiClass',
                                                    'TaskKeywords': ['fake-keyword'],
                                                    'TaskTitle': 'fake-image-labelling-work',
                                                    'TaskDescription': 'fake job',
                                                    'NumberOfHumanWorkersPerDataObject': 1,
                                                    'TaskTimeLimitInSeconds': 180,
                                                    'TaskAvailabilityLifetimeInSeconds': 30,
                                                    'MaxConcurrentTaskCount': 10,
                                                    'AnnotationConsolidationConfig': {'AnnotationConsolidationLambdaArn': 'arn:aws:lambda:us-west-2:081040173940:function:ACS-ImageMultiClass'},
                                                    'PublicWorkforceTaskPrice': {'AmountInUsd': {'Dollars': 0, 'Cents': 0, 'TenthFractionsOfACent': 0}}},
                                'Tags': [{'Key': 'fake_key', 'Value': 'fake_value'}]}
                     )

