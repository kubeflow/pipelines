import unittest

from unittest.mock import patch, call, Mock, MagicMock, mock_open
from botocore.exceptions import ClientError

from model.src import create_model
from common import _utils


required_args = [
  '--region', 'us-west-2',
  '--model_name', 'model_test',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--image', 'test-image',
  '--model_artifact_url', 's3://fake-bucket/model_artifact',
  '--model_name_output_path', '/tmp/output'
]

class ModelTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = create_model.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    create_model._utils = MagicMock()
    create_model._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    create_model._utils.create_model.return_value = 'model_test'

    create_model.main(required_args)

    # Check if correct requests were created and triggered
    create_model._utils.create_model.assert_called()

    # Check the file outputs
    create_model._utils.write_output.assert_has_calls([
      call('/tmp/output', 'model_test')
    ])

  def test_create_model(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args)
    response = _utils.create_model(mock_client, vars(mock_args))

    mock_client.create_model.assert_called_once_with(
      EnableNetworkIsolation=True,
      ExecutionRoleArn='arn:aws:iam::123456789012:user/Development/product_1234/*',
      ModelName='model_test',
      PrimaryContainer={'Image': 'test-image', 'ModelDataUrl': 's3://fake-bucket/model_artifact', 'Environment': {}},
      Tags=[]
    )


  def test_sagemaker_exception_in_create_model(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "create_model")
    mock_client.create_model.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      _utils.create_model(mock_client, vars(mock_args))

  def test_secondary_containers(self):
    arguments = required_args + ['--secondary_containers', '["fake-container"]']
    response = _utils.create_model_request(vars(self.parser.parse_args(arguments)))

    self.assertEqual(response['Containers'], ['fake-container'])

  def test_pass_most_arguments(self):
    arguments = required_args + ['--container_host_name', 'fake-host',
                                 '--tags', '{"fake_key": "fake_value"}',
                                 '--vpc_security_group_ids', 'fake-ids',
                                 '--vpc_subnets', 'fake-subnets']
    response = _utils.create_model_request(vars(self.parser.parse_args(arguments)))
    print(response)
    self.assertEqual(response, {'ModelName': 'model_test',
                                'PrimaryContainer': {'ContainerHostname': 'fake-host',
                                                     'Image': 'test-image',
                                                     'ModelDataUrl': 's3://fake-bucket/model_artifact',
                                                     'Environment': {}
                                                    },
                                'ExecutionRoleArn': 'arn:aws:iam::123456789012:user/Development/product_1234/*',
                                'Tags': [{'Key': 'fake_key', 'Value': 'fake_value'}],
                                'VpcConfig': {'SecurityGroupIds': ['fake-ids'], 'Subnets': ['fake-subnets']},
                                'EnableNetworkIsolation': True
                               })

  def test_image_model_package(self):
    arguments = [ '--region', 'us-west-2',
                  '--model_name', 'model_test',
                  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*'
    ]

    # does not error out
    _utils.create_model_request(vars(self.parser.parse_args(arguments + ['--image', 'test-image',
                                                                         '--model_artifact_url', 's3://fake-bucket/model_artifact'
                                                                         ])))
    # does not error out
    _utils.create_model_request(vars(self.parser.parse_args(arguments + ['--model_package', 'fake-package'])))

    with self.assertRaises(Exception):
      _utils.create_model_request(vars(self.parser.parse_args(arguments)))
    with self.assertRaises(Exception):
      _utils.create_model_request(vars(self.parser.parse_args(arguments+ ['--image', 'test-image'])))
    with self.assertRaises(Exception):
      _utils.create_model_request(vars(self.parser.parse_args(arguments+ ['--model_artifact_url', 's3://fake-bucket/model_artifact'])))