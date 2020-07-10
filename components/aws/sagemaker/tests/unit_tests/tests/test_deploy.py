import unittest

from unittest.mock import patch, call, Mock, MagicMock, mock_open
from botocore.exceptions import ClientError

from deploy.src import deploy
from common import _utils


required_args = [
  '--region', 'us-west-2',
  '--model_name_1', 'model-test',
  '--endpoint_name_output_path', '/tmp/output'
]

class DeployTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = deploy.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_main(self):
    # Mock out all of utils except parser
    deploy._utils = MagicMock()
    deploy._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    deploy._utils.deploy_model.return_value = 'test-endpoint-name'

    deploy.main(required_args)

    # Check if correct requests were created and triggered
    deploy._utils.deploy_model.assert_called()
    deploy._utils.wait_for_endpoint_creation.assert_called()

    # Check the file outputs
    deploy._utils.write_output.assert_has_calls([
      call('/tmp/output', 'test-endpoint-name')
    ])

  def test_deploy_model(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + ['--endpoint_name', 'test-endpoint-name', '--endpoint_config_name', 'test-endpoint-config-name'])
    response = _utils.deploy_model(mock_client, vars(mock_args))

    mock_client.create_endpoint_config.assert_called_once_with(
      EndpointConfigName='test-endpoint-config-name',
      ProductionVariants=[
        {'VariantName': 'variant-name-1', 'ModelName': 'model-test', 'InitialInstanceCount': 1,
         'InstanceType': 'ml.m4.xlarge', 'InitialVariantWeight': 1.0}
      ],
      Tags=[]
    )

    self.assertEqual(response, 'test-endpoint-name')

  def test_sagemaker_exception_in_deploy_model(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "deploy_model")
    mock_client.create_endpoint_config.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
      _utils.deploy_model(mock_client, vars(mock_args))

  def test_model_name_exception(self):
    mock_client = MagicMock()
    mock_args = vars(self.parser.parse_args(required_args))
    mock_args['model_name_1'] = None

    with self.assertRaises(Exception):
      _utils.create_endpoint_config_request(mock_args)

  def test_create_endpoint_exception(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "deploy_model")
    mock_client.create_endpoint.side_effect = mock_exception

    with self.assertRaises(Exception):
      _utils.create_endpoint(mock_client, 'us-east-1', 'fake-endpoint', 'fake-endpoint-config', {})

  def test_wait_for_endpoint_creation(self):
    mock_client = MagicMock()
    mock_client.describe_endpoint.side_effect = [
      {"EndpointStatus": "Creating", "EndpointArn": "fake_arn"},
      {"EndpointStatus": "InService", "EndpointArn": "fake_arn"},
      {"EndpointStatus": "Should not be called", "EndpointArn": "fake_arn"}
    ]

    _utils.wait_for_endpoint_creation(mock_client, 'test-endpoint')
    self.assertEqual(mock_client.describe_endpoint.call_count, 2)

  def test_wait_for_failed_job(self):
    mock_client = MagicMock()
    mock_client.describe_endpoint.side_effect = [
      {"EndpointStatus": "Creating", "EndpointArn": "fake_arn"},
      {"EndpointStatus": "Failed", "FailureReason": "SYSTEM FAILURE"},
      {"EndpointStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_endpoint_creation(mock_client, 'test-endpoint')

    self.assertEqual(mock_client.describe_endpoint.call_count, 2)

  def test_get_endpoint_name_from_job(self):
    mock_client = MagicMock()

    # if we don't pass --endpoint_name argument then endpoint name is constructed using --model_name_1
    self.assertEqual(_utils.deploy_model(mock_client, vars(self.parser.parse_args(required_args))), 'Endpoint-test')

  def test_pass_most_args(self):
    arguments =  [
      '--region', 'us-west-2',
      '--endpoint_url', 'fake-url',
      '--model_name_1', 'model-test-1',
      '--accelerator_type_1', 'ml.eia1.medium',
      '--model_name_2', 'model-test-2',
      '--accelerator_type_2', 'ml.eia1.medium',
      '--model_name_3', 'model-test-3',
      '--accelerator_type_3', 'ml.eia1.medium',
      '--resource_encryption_key', 'fake-key',
      '--endpoint_config_tags', '{"fake_config_key": "fake_config_value"}',
      '--endpoint_tags', '{"fake_key": "fake_value"}'
    ]

    response = _utils.create_endpoint_config_request(vars(self.parser.parse_args(arguments)))
    self.assertEqual(response, {'EndpointConfigName': 'EndpointConfig-test-1',
                                'KmsKeyId': 'fake-key',
                                'ProductionVariants': [
                                   {'InitialInstanceCount': 1,
                                    'AcceleratorType': 'ml.eia1.medium',
                                    'InitialVariantWeight': 1.0,
                                    'InstanceType': 'ml.m4.xlarge',
                                    'ModelName': 'model-test-1',
                                    'VariantName': 'variant-name-1'
                                    },
                                   {'InitialInstanceCount': 1,
                                    'AcceleratorType': 'ml.eia1.medium',
                                    'InitialVariantWeight': 1.0,
                                    'InstanceType': 'ml.m4.xlarge',
                                    'ModelName': 'model-test-2',
                                    'VariantName': 'variant-name-2'
                                    },
                                   {'InitialInstanceCount': 1,
                                    'AcceleratorType': 'ml.eia1.medium',
                                    'InitialVariantWeight': 1.0,
                                    'InstanceType': 'ml.m4.xlarge',
                                    'ModelName': 'model-test-3',
                                    'VariantName': 'variant-name-3'
                                    }
                                  ],
                                'Tags': [{'Key': 'fake_config_key', 'Value': 'fake_config_value'}]
                               })

  def test_tag_in_create_endpoint(self):
    mock_client = MagicMock()
    _utils.create_endpoint(mock_client, 'us-east-1', 'fake-endpoint', 'fake-endpoint-config', {"fake_key": "fake_value"})

    mock_client.create_endpoint.assert_called_once_with(
        EndpointConfigName='fake-endpoint-config',
        EndpointName='fake-endpoint',
        Tags=[{'Key': 'fake_key', 'Value': 'fake_value'}]
    )

