import unittest
import os
import signal

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

from hyperparameter_tuning.src import hyperparameter_tuning as hpo
from common import _utils


required_args = [
  '--region', 'us-west-2',
  '--role', 'arn:aws:iam::123456789012:user/Development/product_1234/*',
  '--image', 'test-image',
  '--metric_name', 'test-metric',
  '--metric_type', 'Maximize',
  '--channels', '[{"ChannelName": "train", "DataSource": {"S3DataSource":{"S3Uri": "s3://fake-bucket/data","S3DataType":"S3Prefix","S3DataDistributionType": "FullyReplicated"}},"ContentType":"","CompressionType": "None","RecordWrapperType":"None","InputMode": "File"}]',
  '--output_location', 'test-output-location',
  '--max_num_jobs', '5',
  '--max_parallel_jobs', '2',
  '--hpo_job_name_output_path', '/tmp/hpo_job_name_output_path',
  '--model_artifact_url_output_path', '/tmp/model_artifact_url_output_path',
  '--best_job_name_output_path', '/tmp/best_job_name_output_path',
  '--best_hyperparameters_output_path', '/tmp/best_hyperparameters_output_path',
  '--training_image_output_path', '/tmp/training_image_output_path'
]

class HyperparameterTestCase(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    parser = hpo.create_parser()
    cls.parser = parser

  def test_create_parser(self):
    self.assertIsNotNone(self.parser)

  def test_spot_bad_args(self):
    no_max_wait_args = self.parser.parse_args(required_args + ['--spot_instance', 'True'])
    no_checkpoint_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600'])
    no_s3_uri_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '3600', '--checkpoint_config', '{}'])

    for arg in [no_max_wait_args, no_checkpoint_args, no_s3_uri_args]:
      with self.assertRaises(Exception):
        _utils.create_hyperparameter_tuning_job_request(vars(arg))

  def test_spot_lesser_wait_time(self):
    args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '86399', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/", "LocalPath": "local-path"}'])
    with self.assertRaises(Exception):
      _utils.create_hyperparameter_tuning_job_request(vars(args))

  def test_spot_good_args(self):
    good_args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '86400', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/"}'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(good_args))
    self.assertTrue(response['TrainingJobDefinition']['EnableManagedSpotTraining'])
    self.assertEqual(response['TrainingJobDefinition']['StoppingCondition']['MaxWaitTimeInSeconds'], 86400)
    self.assertEqual(response['TrainingJobDefinition']['CheckpointConfig']['S3Uri'], 's3://fake-uri/')

  def test_spot_local_path(self):
    args = self.parser.parse_args(required_args + ['--spot_instance', 'True', '--max_wait_time', '86400', '--checkpoint_config', '{"S3Uri": "s3://fake-uri/", "LocalPath": "local-path"}'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(args))
    self.assertEqual(response['TrainingJobDefinition']['CheckpointConfig']['S3Uri'], 's3://fake-uri/')
    self.assertEqual(response['TrainingJobDefinition']['CheckpointConfig']['LocalPath'], 'local-path')

  def test_main(self):
    # Mock out all of utils except parser
    hpo._utils = MagicMock()
    hpo._utils.add_default_client_arguments = _utils.add_default_client_arguments

    # Set some static returns
    hpo._utils.create_hyperparameter_tuning_job.return_value = 'job-name'
    hpo._utils.get_best_training_job_and_hyperparameters.return_value = 'best_job', {"key_1": "best_hp_1"}
    hpo._utils.get_image_from_job.return_value = 'training-image'
    hpo._utils.get_model_artifacts_from_job.return_value = 'model-artifacts'

    hpo.main(required_args)

    # Check if correct requests were created and triggered
    hpo._utils.create_hyperparameter_tuning_job.assert_called()
    hpo._utils.wait_for_hyperparameter_training_job.assert_called()

    # Check the file outputs
    hpo._utils.write_output.assert_has_calls([
      call('/tmp/hpo_job_name_output_path', 'job-name'),
      call('/tmp/model_artifact_url_output_path', 'model-artifacts'),
      call('/tmp/best_job_name_output_path', 'best_job'),
      call('/tmp/best_hyperparameters_output_path', {"key_1": "best_hp_1"}, json_encode=True),
      call('/tmp/training_image_output_path', 'training-image')
    ])
  
  def test_create_hyperparameter_tuning_job(self):
    mock_client = MagicMock()
    mock_args = self.parser.parse_args(required_args + ['--job_name', 'test-job'])
    response = _utils.create_hyperparameter_tuning_job(mock_client, vars(mock_args))

    mock_client.create_hyper_parameter_tuning_job.assert_called_once_with(
        HyperParameterTuningJobConfig={'Strategy': 'Bayesian', 
                                       'HyperParameterTuningJobObjective': {'Type': 'Maximize', 'MetricName': 'test-metric'}, 
                                       'ResourceLimits': {'MaxNumberOfTrainingJobs': 5, 'MaxParallelTrainingJobs': 2}, 
                                       'ParameterRanges': {'IntegerParameterRanges': [], 'ContinuousParameterRanges': [], 'CategoricalParameterRanges': []}, 
                                       'TrainingJobEarlyStoppingType': 'Off'
                                      }, 
        HyperParameterTuningJobName='test-job', 
        Tags=[], 
        TrainingJobDefinition={'StaticHyperParameters': {}, 
                               'AlgorithmSpecification': {'TrainingImage': 'test-image', 'TrainingInputMode': 'File'}, 
                               'RoleArn': 'arn:aws:iam::123456789012:user/Development/product_1234/*', 
                               'InputDataConfig': [{'ChannelName': 'train', 
                                                    'DataSource': {'S3DataSource': {'S3Uri': 's3://fake-bucket/data', 
                                                                                    'S3DataType': 'S3Prefix', 
                                                                                    'S3DataDistributionType': 'FullyReplicated'}},
                                                    'ContentType': '', 
                                                    'CompressionType': 'None', 
                                                    'RecordWrapperType': 'None', 
                                                    'InputMode': 'File'}], 
                               'OutputDataConfig': {'KmsKeyId': '', 'S3OutputPath': 'test-output-location'}, 
                               'ResourceConfig': {'InstanceType': 'ml.m4.xlarge', 'InstanceCount': 1, 'VolumeSizeInGB': 30, 'VolumeKmsKeyId': ''},
                               'StoppingCondition': {'MaxRuntimeInSeconds': 86400}, 
                               'EnableNetworkIsolation': True, 
                               'EnableInterContainerTrafficEncryption': False, 
                               'EnableManagedSpotTraining': False}
    )

    self.assertEqual(response, 'test-job')

  def test_main_stop_hyperparameter_tuning_job(self):
    hpo._utils = MagicMock()
    hpo._utils.create_processing_job.return_value = 'job-name'

    try:
      os.kill(os.getpid(), signal.SIGTERM)
    finally:
      hpo._utils.stop_hyperparameter_tuning_job.assert_called_once_with(ANY, 'job-name')
      hpo._utils.get_best_training_job_and_hyperparameters.assert_not_called()

  def test_utils_stop_hyper_parameter_tuning_job(self):
    mock_sm_client = MagicMock()
    mock_sm_client.stop_hyper_parameter_tuning_job.return_value = None

    response = _utils.stop_hyperparameter_tuning_job(mock_sm_client, 'FakeJobName')

    mock_sm_client.stop_hyper_parameter_tuning_job.assert_called_once_with(
        HyperParameterTuningJobName='FakeJobName'
    )
    self.assertEqual(response, None)

  def test_sagemaker_exception_in_create_hyperparameter_tuning_job(self):
    mock_client = MagicMock()
    mock_exception = ClientError({"Error": {"Message": "SageMaker broke"}}, "create_hyperparameter_tuning_job")
    mock_client.create_hyper_parameter_tuning_job.side_effect = mock_exception
    mock_args = self.parser.parse_args(required_args)

    with self.assertRaises(Exception):
       _utils.create_hyperparameter_tuning_job(mock_client, vars(mock_args))
 
  def test_wait_for_hyperparameter_tuning_job(self):
    mock_client = MagicMock()
    mock_client.describe_hyper_parameter_tuning_job.side_effect = [
      {"HyperParameterTuningJobStatus": "InProgress"},
      {"HyperParameterTuningJobStatus": "Completed"},
      {"HyperParameterTuningJobStatus": "Should not be called"}
    ]

    _utils.wait_for_hyperparameter_training_job(mock_client,'hyperparameter-tuning-job',0)
    self.assertEqual(mock_client.describe_hyper_parameter_tuning_job.call_count, 2)
  
  def test_wait_for_failed_job(self):
    mock_client = MagicMock()
    mock_client.describe_hyper_parameter_tuning_job.side_effect = [
      {"HyperParameterTuningJobStatus": "InProgress"},
      {"HyperParameterTuningJobStatus": "Failed", "FailureReason": "Something broke lol"},
      {"HyperParameterTuningJobStatus": "Should not be called"}
    ]

    with self.assertRaises(Exception):
      _utils.wait_for_hyperparameter_training_job(mock_client, 'training-job', 0)

    self.assertEqual(mock_client.describe_hyper_parameter_tuning_job.call_count,2)

  def test_get_image_from_algorithm_job(self):
    mock_client = MagicMock()
    mock_client.describe_hyper_parameter_tuning_job.return_value = {"TrainingJobDefinition": {"AlgorithmSpecification": {"AlgorithmName": "my-algorithm"}}}
    mock_client.describe_algorithm.return_value = {"TrainingSpecification": {"TrainingImage": "training-image-url"}}

    self.assertEqual(_utils.get_image_from_job(mock_client, 'training-job'), "training-image-url")

  def test_best_training_job(self):
    mock_client = MagicMock()
    mock_client.describe_hyper_parameter_tuning_job.return_value = {'BestTrainingJob': {'TrainingJobName': 'best_training_job'}}
    mock_client.describe_training_job.return_value = {"HyperParameters": {"hp": "val", '_tuning_objective_metric': 'remove_me'}}
    name, params =_utils.get_best_training_job_and_hyperparameters(mock_client, "mock-hpo-job")
    self.assertEqual("best_training_job", name)
    self.assertEqual("val", params["hp"])
    
  def test_warm_start_and_parents_args(self):
    # specifying both params 
    good_args = self.parser.parse_args(required_args + ['--warm_start_type', 'TransferLearning'] + ['--parent_hpo_jobs', 'A,B,C'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(good_args))
    self.assertIn('WarmStartConfig', response)
    self.assertIn('ParentHyperParameterTuningJobs', response['WarmStartConfig'])
    self.assertIn('WarmStartType', response['WarmStartConfig'])
    self.assertEqual(response['WarmStartConfig']['ParentHyperParameterTuningJobs'][0]['HyperParameterTuningJobName'], 'A')
    self.assertEqual(response['WarmStartConfig']['ParentHyperParameterTuningJobs'][1]['HyperParameterTuningJobName'], 'B')
    self.assertEqual(response['WarmStartConfig']['ParentHyperParameterTuningJobs'][2]['HyperParameterTuningJobName'], 'C')
    self.assertEqual(response['WarmStartConfig']['WarmStartType'], 'TransferLearning')
 
  def test_either_warm_start_or_parents_args(self):
    # It will generate an exception if either warm_start_type or parent hpo jobs is being passed
    missing_parent_hpo_jobs_args = self.parser.parse_args(required_args + ['--warm_start_type', 'TransferLearning'])
    with self.assertRaises(Exception):
      _utils.create_hyperparameter_tuning_job_request(vars(missing_parent_hpo_jobs_args))

    missing_warm_start_type_args = self.parser.parse_args(required_args + ['--parent_hpo_jobs', 'A,B,C']) 
    with self.assertRaises(Exception):
      _utils.create_hyperparameter_tuning_job_request(vars(missing_warm_start_type_args))
     

  def test_reasonable_required_args(self):
    response = _utils.create_hyperparameter_tuning_job_request(vars(self.parser.parse_args(required_args)))

    # Ensure all of the optional arguments have reasonable default values
    self.assertFalse(response['TrainingJobDefinition']['EnableManagedSpotTraining'])
    self.assertDictEqual(response['TrainingJobDefinition']['StaticHyperParameters'], {})
    self.assertNotIn('VpcConfig', response['TrainingJobDefinition'])
    self.assertNotIn('MetricDefinitions', response['TrainingJobDefinition'])
    self.assertEqual(response['Tags'], [])
    self.assertEqual(response['TrainingJobDefinition']['AlgorithmSpecification']['TrainingInputMode'], 'File')
    self.assertEqual(response['TrainingJobDefinition']['OutputDataConfig']['S3OutputPath'], 'test-output-location')

  def test_metric_definitions(self):
    metric_definition_args = self.parser.parse_args(required_args + ['--metric_definitions', '{"metric1": "regexval1", "metric2": "regexval2"}'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(metric_definition_args))

    self.assertIn('MetricDefinitions', response['TrainingJobDefinition']['AlgorithmSpecification'])
    response_metric_definitions = response['TrainingJobDefinition']['AlgorithmSpecification']['MetricDefinitions']

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
      _utils.create_hyperparameter_tuning_job_request(vars(parsed_args))

  def test_first_party_algorithm(self):
    algorithm_name_args = self.parser.parse_args(required_args + ['--algorithm_name', 'first-algorithm'])

    # Should not throw an exception
    response = _utils.create_hyperparameter_tuning_job_request(vars(algorithm_name_args))
    self.assertIn('TrainingJobDefinition', response)
    self.assertIn('TrainingImage', response['TrainingJobDefinition']['AlgorithmSpecification'])
    self.assertNotIn('AlgorithmName', response['TrainingJobDefinition']['AlgorithmSpecification'])

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

    response = _utils.create_hyperparameter_tuning_job_request(vars(parsed_args))

    _utils.get_image_uri.assert_called_with('us-west-2', 'seq2seq')
    self.assertEqual(response['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'], "seq2seq-url")


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

    response = _utils.create_hyperparameter_tuning_job_request(vars(parsed_args))

    _utils.get_image_uri.assert_called_with('us-west-2', 'seq2seq')
    self.assertEqual(response['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'], "seq2seq-url")


  def test_unknown_algorithm(self):
    known_algorithm_args = required_args + ['--algorithm_name', 'unknown algorithm']
    image_index = required_args.index('--image')
    # Cut out --image and it's associated value
    known_algorithm_args = known_algorithm_args[:image_index] + known_algorithm_args[image_index+2:]

    parsed_args = self.parser.parse_args(known_algorithm_args)

    # Patch get_image_uri
    _utils.get_image_uri = MagicMock()
    _utils.get_image_uri.return_value = "unknown-url"

    response = _utils.create_hyperparameter_tuning_job_request(vars(parsed_args))

    # Should just place the algorithm name in regardless
    _utils.get_image_uri.assert_not_called()
    self.assertEqual(response['TrainingJobDefinition']['AlgorithmSpecification']['AlgorithmName'], "unknown algorithm")

  def test_no_channels(self):
    no_channels_args = required_args.copy()
    channels_index = required_args.index('--channels')
    # Replace the value after the flag with an empty list
    no_channels_args[channels_index + 1] = '[]'
    parsed_args = self.parser.parse_args(no_channels_args)

    with self.assertRaises(Exception):
      _utils.create_hyperparameter_tuning_job_request(vars(parsed_args))

  def test_tags(self):
    args = self.parser.parse_args(required_args + ['--tags', '{"key1": "val1", "key2": "val2"}'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(args))
    self.assertIn({'Key': 'key1', 'Value': 'val1'}, response['Tags'])
    self.assertIn({'Key': 'key2', 'Value': 'val2'}, response['Tags'])

  def test_valid_hyperparameters(self):
    hyperparameters_str = '{"hp1": "val1", "hp2": "val2", "hp3": "val3"}'
    categorical_params = '[{"Name" : "categorical", "Values": ["A", "B"]}]'
    integer_params = '[{"MaxValue": "integer_val1", "MinValue": "integer_val2", "Name": "integer", "ScalingType": "test_integer"}]'
    continuous_params = '[{"MaxValue": "continuous_val1", "MinValue": "continuous_val2", "Name": "continuous", "ScalingType": "test_continuous"}]'
    good_args = self.parser.parse_args(required_args + ['--static_parameters', hyperparameters_str] + ['--integer_parameters', integer_params] + ['--continuous_parameters', continuous_params] + ['--categorical_parameters', categorical_params])
    response = _utils.create_hyperparameter_tuning_job_request(vars(good_args))
    
    self.assertIn('hp1', response['TrainingJobDefinition']['StaticHyperParameters'])
    self.assertIn('hp2', response['TrainingJobDefinition']['StaticHyperParameters'])
    self.assertIn('hp3', response['TrainingJobDefinition']['StaticHyperParameters'])
    self.assertEqual(response['TrainingJobDefinition']['StaticHyperParameters']['hp1'], "val1")
    self.assertEqual(response['TrainingJobDefinition']['StaticHyperParameters']['hp2'], "val2")
    self.assertEqual(response['TrainingJobDefinition']['StaticHyperParameters']['hp3'], "val3")
  
    self.assertIn('ParameterRanges', response['HyperParameterTuningJobConfig'])
    self.assertIn('IntegerParameterRanges', response['HyperParameterTuningJobConfig']['ParameterRanges'])
    self.assertIn('ContinuousParameterRanges', response['HyperParameterTuningJobConfig']['ParameterRanges'])
    self.assertIn('CategoricalParameterRanges', response['HyperParameterTuningJobConfig']['ParameterRanges'])
    self.assertIn('Name', response['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'][0])
    self.assertIn('Values', response['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'][0])
    self.assertIn('MaxValue', response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0])
    self.assertIn('MinValue', response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0])
    self.assertIn('Name', response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0])
    self.assertIn('ScalingType', response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0])
    self.assertIn('MaxValue', response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0])
    self.assertIn('MinValue', response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0])
    self.assertIn('Name', response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0])
    self.assertIn('ScalingType', response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0])

    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'][0]['Name'], "categorical")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'][0]["Values"][0], "A")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'][0]["Values"][1], "B")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0]['MaxValue'], "integer_val1")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0]['MinValue'], "integer_val2")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0]['Name'], "integer")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'][0]['ScalingType'], "test_integer")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0]['MaxValue'], "continuous_val1")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0]['MinValue'], "continuous_val2")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0]['Name'], "continuous")
    self.assertEqual(response['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'][0]['ScalingType'], "test_continuous")


  def test_empty_hyperparameters(self):
    hyperparameters_str = '{}'

    good_args = self.parser.parse_args(required_args + ['--static_parameters', hyperparameters_str])
    response = _utils.create_hyperparameter_tuning_job_request(vars(good_args))
    
    self.assertEqual(response['TrainingJobDefinition']['StaticHyperParameters'], {}) 

  def test_object_hyperparameters(self):
    hyperparameters_str = '{"hp1": {"innerkey": "innerval"}}'

    invalid_args = self.parser.parse_args(required_args + ['--static_parameters', hyperparameters_str])
    with self.assertRaises(Exception):
      _utils.create_hyperparameter_tuning_job_request(vars(invalid_args))

  def test_vpc_configuration(self):
    required_vpc_args = self.parser.parse_args(required_args + ['--vpc_security_group_ids', 'sg1,sg2', '--vpc_subnets', 'subnet1,subnet2'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(required_vpc_args))
    
    self.assertIn('TrainingJobDefinition', response)
    self.assertIn('VpcConfig', response['TrainingJobDefinition'])
    self.assertIn('sg1', response['TrainingJobDefinition']['VpcConfig']['SecurityGroupIds'])
    self.assertIn('sg2', response['TrainingJobDefinition']['VpcConfig']['SecurityGroupIds'])
    self.assertIn('subnet1', response['TrainingJobDefinition']['VpcConfig']['Subnets'])
    self.assertIn('subnet2', response['TrainingJobDefinition']['VpcConfig']['Subnets'])

  def test_training_mode(self):
    required_vpc_args = self.parser.parse_args(required_args + ['--training_input_mode', 'Pipe'])
    response = _utils.create_hyperparameter_tuning_job_request(vars(required_vpc_args))

    self.assertEqual(response['TrainingJobDefinition']['AlgorithmSpecification']['TrainingInputMode'], 'Pipe')
