# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import argparse
from time import gmtime, strftime
import time
import string
import random
import json
import yaml

import boto3
from botocore.exceptions import ClientError
from sagemaker.amazon.amazon_estimator import get_image_uri

import logging
logging.getLogger().setLevel(logging.INFO)

# Mappings are extracted from the first table in https://docs.aws.amazon.com/sagemaker/latest/dg/sagemaker-algo-docker-registry-paths.html
built_in_algos = {
    'blazingtext': 'blazingtext',
    'deepar forecasting': 'forecasting-deepar',
    'factorization machines': 'factorization-machines',
    'image classification': 'image-classification',
    'ip insights': 'ipinsights',
    'k-means': 'kmeans',
    'k-nearest neighbors': 'knn',
    'k-nn': 'knn',
    'lda': 'lda',
    'linear learner': 'linear-learner',
    'neural topic model': 'ntm',
    'object2vec': 'object2vec',
    'object detection': 'object-detection',
    'pca': 'pca',
    'random cut forest': 'randomcutforest',
    'semantic segmentation': 'semantic-segmentation',
    'sequence to sequence': 'seq2seq',
    'seq2seq modeling': 'seq2seq',
    'xgboost': 'xgboost'
}

# Get current directory to open templates
__cwd__ = os.path.realpath(os.path.join(os.getcwd(), os.path.dirname(__file__)))

def add_default_client_arguments(parser):
    parser.add_argument('--region', type=str.strip, required=True, help='The region where the training job launches.')
    parser.add_argument('--endpoint_url', type=str.strip, required=False, help='The URL to use when communicating with the Sagemaker service.')

def get_sagemaker_client(region, endpoint_url=None):
    """Builds a client to the AWS SageMaker API."""
    client = boto3.client('sagemaker', region_name=region, endpoint_url=endpoint_url)
    return client

def create_training_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_training_job
    with open(os.path.join(__cwd__, 'train.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    job_name = args['job_name'] if args['job_name'] else 'TrainingJob-' + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

    request['TrainingJobName'] = job_name
    request['RoleArn'] = args['role']
    request['HyperParameters'] = args['hyperparameters']
    request['AlgorithmSpecification']['TrainingInputMode'] = args['training_input_mode']

    ### Update training image (for BYOC and built-in algorithms) or algorithm resource name
    if not args['image'] and not args['algorithm_name']:
        logging.error('Please specify training image or algorithm name.')
        raise Exception('Could not create job request')
    if args['image'] and args['algorithm_name']:
        logging.error('Both image and algorithm name inputted, only one should be specified. Proceeding with image.')

    if args['image']:
        request['AlgorithmSpecification']['TrainingImage'] = args['image']
        request['AlgorithmSpecification'].pop('AlgorithmName')
    else:
        # TODO: Adjust this implementation to account for custom algorithm resources names that are the same as built-in algorithm names
        algo_name = args['algorithm_name'].lower().strip()
        if algo_name in built_in_algos.keys():
            request['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], built_in_algos[algo_name])
            request['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        # Just to give the user more leeway for built-in algorithm name inputs
        elif algo_name in built_in_algos.values():
            request['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], algo_name)
            request['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        else:
            request['AlgorithmSpecification']['AlgorithmName'] = args['algorithm_name']
            request['AlgorithmSpecification'].pop('TrainingImage')
    
    ### Update metric definitions
    if args['metric_definitions']:
        for key, val in args['metric_definitions'].items():
            request['AlgorithmSpecification']['MetricDefinitions'].append({'Name': key, 'Regex': val})
    else:
        request['AlgorithmSpecification'].pop('MetricDefinitions')

    ### Update or pop VPC configs
    if args['vpc_security_group_ids'] and args['vpc_subnets']:
        request['VpcConfig']['SecurityGroupIds'] = [args['vpc_security_group_ids']]
        request['VpcConfig']['Subnets'] = [args['vpc_subnets']]
    else:
        request.pop('VpcConfig')

    ### Update input channels, must have at least one specified
    if len(args['channels']) > 0:
        request['InputDataConfig'] = args['channels']
        # Max number of input channels/data locations is 20, but currently only 8 data location parameters are exposed separately.
        #   Source: Input data configuration description in the SageMaker create training job form
        for i in range(1, len(args['channels']) + 1):
            if args['data_location_' + str(i)]:
                request['InputDataConfig'][i-1]['DataSource']['S3DataSource']['S3Uri'] = args['data_location_' + str(i)]
    else:
        logging.error("Must specify at least one input channel.")
        raise Exception('Could not create job request')

    request['OutputDataConfig']['S3OutputPath'] = args['model_artifact_path']
    request['OutputDataConfig']['KmsKeyId'] = args['output_encryption_key']
    request['ResourceConfig']['InstanceType'] = args['instance_type']
    request['ResourceConfig']['VolumeKmsKeyId'] = args['resource_encryption_key']
    request['EnableNetworkIsolation'] = args['network_isolation']
    request['EnableInterContainerTrafficEncryption'] = args['traffic_encryption']

    ### Update InstanceCount, VolumeSizeInGB, and MaxRuntimeInSeconds if input is non-empty and > 0, otherwise use default values
    if args['instance_count']:
        request['ResourceConfig']['InstanceCount'] = args['instance_count']

    if args['volume_size']:
        request['ResourceConfig']['VolumeSizeInGB'] = args['volume_size']

    if args['max_run_time']:
        request['StoppingCondition']['MaxRuntimeInSeconds'] = args['max_run_time']

    enable_spot_instance_support(request, args)

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_training_job(client, args):
  """Create a Sagemaker training job."""
  request = create_training_job_request(args)
  try:
      client.create_training_job(**request)
      training_job_name = request['TrainingJobName']
      logging.info("Created Training Job with name: " + training_job_name)
      logging.info("Training job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}"
        .format(args['region'], args['region'], training_job_name))
      logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix"
        .format(args['region'], args['region'], training_job_name))
      return training_job_name
  except ClientError as e:
      raise Exception(e.response['Error']['Message'])


def wait_for_training_job(client, training_job_name):
  while(True):
    response = client.describe_training_job(TrainingJobName=training_job_name)
    status = response['TrainingJobStatus']
    if status == 'Completed':
      logging.info("Training job ended with status: " + status)
      break
    if status == 'Failed':
      message = response['FailureReason']
      logging.info('Training failed with the following error: {}'.format(message))
      raise Exception('Training job failed')
    logging.info("Training job is still in status: " + status)
    time.sleep(30)


def get_model_artifacts_from_job(client, job_name):
  info = client.describe_training_job(TrainingJobName=job_name)
  model_artifact_url = info['ModelArtifacts']['S3ModelArtifacts']
  return model_artifact_url


def get_image_from_job(client, job_name):
    info = client.describe_training_job(TrainingJobName=job_name)
    try:
        image = info['AlgorithmSpecification']['TrainingImage']
    except:
        algorithm_name = info['AlgorithmSpecification']['AlgorithmName']
        image = client.describe_algorithm(AlgorithmName=algorithm_name)['TrainingSpecification']['TrainingImage']

    return image


def create_model(client, args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_model
    with open(os.path.join(__cwd__, 'model.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    request['ModelName'] = args['model_name']
    request['PrimaryContainer']['Environment'] = args['environment']

    if args['secondary_containers']:
        request['Containers'] = args['secondary_containers']
        request.pop('PrimaryContainer')
    else:
        request.pop('Containers')
        ### Update primary container and handle input errors
        if args['container_host_name']:
            request['PrimaryContainer']['ContainerHostname'] = args['container_host_name']
        else:
            request['PrimaryContainer'].pop('ContainerHostname')

        if (args['image'] or args['model_artifact_url']) and args['model_package']:
            logging.error("Please specify an image AND model artifact url, OR a model package name.")
            raise Exception("Could not make create model request.")
        elif args['model_package']:
            request['PrimaryContainer']['ModelPackageName'] = args['model_package']
            request['PrimaryContainer'].pop('Image')
            request['PrimaryContainer'].pop('ModelDataUrl')
        else:
            if args['image'] and args['model_artifact_url']:
                request['PrimaryContainer']['Image'] = args['image']
                request['PrimaryContainer']['ModelDataUrl'] = args['model_artifact_url']
                request['PrimaryContainer'].pop('ModelPackageName')
            else:
                logging.error("Please specify an image AND model artifact url.")
                raise Exception("Could not make create model request.")

    request['ExecutionRoleArn'] = args['role']
    request['EnableNetworkIsolation'] = args['network_isolation']

    ### Update or pop VPC configs
    if args['vpc_security_group_ids'] and args['vpc_subnets']:
        request['VpcConfig']['SecurityGroupIds'] = [args['vpc_security_group_ids']]
        request['VpcConfig']['Subnets'] = [args['vpc_subnets']]
    else:
        request.pop('VpcConfig')

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    create_model_response = client.create_model(**request)

    logging.info("Model Config Arn: " + create_model_response['ModelArn'])
    return create_model_response['ModelArn']


def deploy_model(client, args):
  endpoint_config_name = create_endpoint_config(client, args)
  endpoint_name = create_endpoint(client, args['region'], args['endpoint_name'], endpoint_config_name, args['endpoint_tags'])
  return endpoint_name


def create_endpoint_config(client, args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint_config
    with open(os.path.join(__cwd__, 'endpoint_config.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    endpoint_config_name = args['endpoint_config_name'] if args['endpoint_config_name'] else 'EndpointConfig' + args['model_name_1'][args['model_name_1'].index('-'):]
    request['EndpointConfigName'] = endpoint_config_name

    if args['resource_encryption_key']:
        request['KmsKeyId'] = args['resource_encryption_key']
    else:
        request.pop('KmsKeyId')

    if not args['model_name_1']:
        logging.error("Must specify at least one model (model name) to host.")
        raise Exception("Could not create endpoint config.")

    for i in range(len(request['ProductionVariants']), 0, -1):
        if args['model_name_' + str(i)]:
            request['ProductionVariants'][i-1]['ModelName'] = args['model_name_' + str(i)]
            if args['variant_name_' + str(i)]:
                request['ProductionVariants'][i-1]['VariantName'] = args['variant_name_' + str(i)]
            if args['initial_instance_count_' + str(i)]:
                request['ProductionVariants'][i-1]['InitialInstanceCount'] = args['initial_instance_count_' + str(i)]
            if args['instance_type_' + str(i)]:
                request['ProductionVariants'][i-1]['InstanceType'] = args['instance_type_' + str(i)]
            if args['initial_variant_weight_' + str(i)]:
                request['ProductionVariants'][i-1]['InitialVariantWeight'] = args['initial_variant_weight_' + str(i)]
            if args['accelerator_type_' + str(i)]:
                request['ProductionVariants'][i-1]['AcceleratorType'] = args['accelerator_type_' + str(i)]
            else:
                request['ProductionVariants'][i-1].pop('AcceleratorType')
        else:
            request['ProductionVariants'].pop(i-1)

    ### Update tags
    for key, val in args['endpoint_config_tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    try:
        create_endpoint_config_response = client.create_endpoint_config(**request)
        logging.info("Endpoint configuration in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpointConfig/{}"
            .format(args['region'], args['region'], endpoint_config_name))
        logging.info("Endpoint Config Arn: " + create_endpoint_config_response['EndpointConfigArn'])
        return endpoint_config_name
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def create_endpoint(client, region, endpoint_name, endpoint_config_name, endpoint_tags):
  ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint
  endpoint_name = endpoint_name if endpoint_name else 'Endpoint' + endpoint_config_name[endpoint_config_name.index('-'):]

  ### Update tags
  tags=[]
  for key, val in endpoint_tags.items():
      tags.append({'Key': key, 'Value': val})

  try:
      create_endpoint_response = client.create_endpoint(
          EndpointName=endpoint_name,
          EndpointConfigName=endpoint_config_name,
          Tags=tags)
      logging.info("Created endpoint with name: " + endpoint_name)
      logging.info("Endpoint in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpoints/{}"
          .format(region, region, endpoint_name))
      logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/Endpoints/{};streamFilter=typeLogStreamPrefix"
          .format(region, region, endpoint_name))
      return endpoint_name
  except ClientError as e:
      raise Exception(e.response['Error']['Message'])


def wait_for_endpoint_creation(client, endpoint_name):
  status = client.describe_endpoint(EndpointName=endpoint_name)['EndpointStatus']
  logging.info("Status: " + status)

  try:
    client.get_waiter('endpoint_in_service').wait(EndpointName=endpoint_name)
  finally:
    resp = client.describe_endpoint(EndpointName=endpoint_name)
    status = resp['EndpointStatus']
    logging.info("Endpoint Arn: " + resp['EndpointArn'])
    logging.info("Create endpoint ended with status: " + status)

    if status != 'InService':
      message = client.describe_endpoint(EndpointName=endpoint_name)['FailureReason']
      logging.info('Create endpoint failed with the following error: {}'.format(message))
      raise Exception('Endpoint creation did not succeed')


def create_transform_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_transform_job
    with open(os.path.join(__cwd__, 'transform.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    job_name = args['job_name'] if args['job_name'] else 'BatchTransform' + args['model_name'][args['model_name'].index('-'):]

    request['TransformJobName'] = job_name
    request['ModelName'] = args['model_name']

    if args['max_concurrent']:
        request['MaxConcurrentTransforms'] = args['max_concurrent']

    if args['max_payload'] or args['max_payload'] == 0:
        request['MaxPayloadInMB'] = args['max_payload']

    if args['batch_strategy']:
        request['BatchStrategy'] = args['batch_strategy']
    else:
        request.pop('BatchStrategy')

    request['Environment'] = args['environment']

    if args['data_type']:
        request['TransformInput']['DataSource']['S3DataSource']['S3DataType'] = args['data_type']

    request['TransformInput']['DataSource']['S3DataSource']['S3Uri'] = args['input_location']
    request['TransformInput']['ContentType'] = args['content_type']

    if args['compression_type']:
        request['TransformInput']['CompressionType'] = args['compression_type']

    if args['split_type']:
        request['TransformInput']['SplitType'] = args['split_type']

    request['TransformOutput']['S3OutputPath'] = args['output_location']
    request['TransformOutput']['Accept'] = args['accept']
    request['TransformOutput']['KmsKeyId'] = args['output_encryption_key']

    if args['assemble_with']:
        request['TransformOutput']['AssembleWith'] = args['assemble_with']
    else:
        request['TransformOutput'].pop('AssembleWith')

    request['TransformResources']['InstanceType'] = args['instance_type']
    request['TransformResources']['InstanceCount'] = args['instance_count']
    request['TransformResources']['VolumeKmsKeyId'] = args['resource_encryption_key']
    request['DataProcessing']['InputFilter'] = args['input_filter']
    request['DataProcessing']['OutputFilter'] = args['output_filter']

    if args['join_source']:
        request['DataProcessing']['JoinSource'] = args['join_source']

    ### Update tags
    if not args['tags'] is None:
        for key, val in args['tags'].items():
            request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_transform_job(client, args):
  request = create_transform_job_request(args)
  try:
      client.create_transform_job(**request)
      batch_job_name = request['TransformJobName']
      logging.info("Created Transform Job with name: " + batch_job_name)
      logging.info("Transform job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/transform-jobs/{}"
        .format(args['region'], args['region'], batch_job_name))
      logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TransformJobs;prefix={};streamFilter=typeLogStreamPrefix"
        .format(args['region'], args['region'], batch_job_name))
      return batch_job_name
  except ClientError as e:
      raise Exception(e.response['Error']['Message'])


def wait_for_transform_job(client, batch_job_name):
  ### Wait until the job finishes
  while(True):
    response = client.describe_transform_job(TransformJobName=batch_job_name)
    status = response['TransformJobStatus']
    if status == 'Completed':
      logging.info("Transform job ended with status: " + status)
      break
    if status == 'Failed':
      message = response['FailureReason']
      logging.info('Transform failed with the following error: {}'.format(message))
      raise Exception('Transform job failed')
    logging.info("Transform job is still in status: " + status)
    time.sleep(30)


def create_hyperparameter_tuning_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_hyper_parameter_tuning_job
    with open(os.path.join(__cwd__, 'hpo.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    ### Create a hyperparameter tuning job
    request['HyperParameterTuningJobName'] = args['job_name'] if args['job_name'] else "HPOJob-" + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

    request['HyperParameterTuningJobConfig']['Strategy'] = args['strategy']
    request['HyperParameterTuningJobConfig']['HyperParameterTuningJobObjective']['Type'] = args['metric_type']
    request['HyperParameterTuningJobConfig']['HyperParameterTuningJobObjective']['MetricName'] = args['metric_name']
    request['HyperParameterTuningJobConfig']['ResourceLimits']['MaxNumberOfTrainingJobs'] = args['max_num_jobs']
    request['HyperParameterTuningJobConfig']['ResourceLimits']['MaxParallelTrainingJobs'] = args['max_parallel_jobs']
    request['HyperParameterTuningJobConfig']['ParameterRanges']['IntegerParameterRanges'] = args['integer_parameters']
    request['HyperParameterTuningJobConfig']['ParameterRanges']['ContinuousParameterRanges'] = args['continuous_parameters']
    request['HyperParameterTuningJobConfig']['ParameterRanges']['CategoricalParameterRanges'] = args['categorical_parameters']
    request['HyperParameterTuningJobConfig']['TrainingJobEarlyStoppingType'] = args['early_stopping_type']

    request['TrainingJobDefinition']['StaticHyperParameters'] = args['static_parameters']
    request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingInputMode'] = args['training_input_mode']

    ### Update training image (for BYOC) or algorithm resource name
    if not args['image'] and not args['algorithm_name']:
        logging.error('Please specify training image or algorithm name.')
        raise Exception('Could not create job request')
    if args['image'] and args['algorithm_name']:
        logging.error('Both image and algorithm name inputted, only one should be specified. Proceeding with image.')

    if args['image']:
        request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = args['image']
        request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
    else:
        # TODO: Adjust this implementation to account for custom algorithm resources names that are the same as built-in algorithm names
        algo_name = args['algorithm_name'].lower().strip()
        if algo_name in built_in_algos.keys():
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], built_in_algos[algo_name])
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        # To give the user more leeway for built-in algorithm name inputs
        elif algo_name in built_in_algos.values():
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], algo_name)
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        else:
            request['TrainingJobDefinition']['AlgorithmSpecification']['AlgorithmName'] = args['algorithm_name']
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('TrainingImage')

    ### Update metric definitions
    if args['metric_definitions']:
        for key, val in args['metric_definitions'].items():
            request['TrainingJobDefinition']['AlgorithmSpecification']['MetricDefinitions'].append({'Name': key, 'Regex': val})
    else:
        request['TrainingJobDefinition']['AlgorithmSpecification'].pop('MetricDefinitions')

    ### Update or pop VPC configs
    if args['vpc_security_group_ids'] and args['vpc_subnets']:
        request['TrainingJobDefinition']['VpcConfig']['SecurityGroupIds'] = [args['vpc_security_group_ids']]
        request['TrainingJobDefinition']['VpcConfig']['Subnets'] = [args['vpc_subnets']]
    else:
        request['TrainingJobDefinition'].pop('VpcConfig')

    ### Update input channels, must have at least one specified
    if len(args['channels']) > 0:
        request['TrainingJobDefinition']['InputDataConfig'] = args['channels']
        # Max number of input channels/data locations is 20, but currently only 8 data location parameters are exposed separately.
        #   Source: Input data configuration description in the SageMaker create hyperparameter tuning job form
        for i in range(1, len(args['channels']) + 1):
            if args['data_location_' + str(i)]:
                request['TrainingJobDefinition']['InputDataConfig'][i-1]['DataSource']['S3DataSource']['S3Uri'] = args['data_location_' + str(i)]
    else:
        logging.error("Must specify at least one input channel.")
        raise Exception('Could not make job request')

    request['TrainingJobDefinition']['OutputDataConfig']['S3OutputPath'] = args['output_location']
    request['TrainingJobDefinition']['OutputDataConfig']['KmsKeyId'] = args['output_encryption_key']
    request['TrainingJobDefinition']['ResourceConfig']['InstanceType'] = args['instance_type']
    request['TrainingJobDefinition']['ResourceConfig']['VolumeKmsKeyId'] = args['resource_encryption_key']
    request['TrainingJobDefinition']['EnableNetworkIsolation'] = args['network_isolation']
    request['TrainingJobDefinition']['EnableInterContainerTrafficEncryption'] = args['traffic_encryption']
    request['TrainingJobDefinition']['RoleArn'] = args['role']

    ### Update InstanceCount, VolumeSizeInGB, and MaxRuntimeInSeconds if input is non-empty and > 0, otherwise use default values
    if args['instance_count']:
        request['TrainingJobDefinition']['ResourceConfig']['InstanceCount'] = args['instance_count']

    if args['volume_size']:
        request['TrainingJobDefinition']['ResourceConfig']['VolumeSizeInGB'] = args['volume_size']

    if args['max_run_time']:
        request['TrainingJobDefinition']['StoppingCondition']['MaxRuntimeInSeconds'] = args['max_run_time']

    ### Update or pop warm start configs
    if args['warm_start_type'] and args['parent_hpo_jobs']:
        request['WarmStartConfig']['WarmStartType'] = args['warm_start_type']
        parent_jobs = [n.strip() for n in args['parent_hpo_jobs'].split(',')]
        for i in range(len(parent_jobs)):
            request['WarmStartConfig']['ParentHyperParameterTuningJobs'].append({'HyperParameterTuningJobName': parent_jobs[i]})
    else:
        if args['warm_start_type'] or args['parent_hpo_jobs']:
            if not args['warm_start_type']:
                logging.error('Must specify warm start type as either "IdenticalDataAndAlgorithm" or "TransferLearning".')
            if not args['parent_hpo_jobs']:
                logging.error("Must specify at least one parent hyperparameter tuning job")
            raise Exception('Could not make job request')
        request.pop('WarmStartConfig')

    enable_spot_instance_support(request['TrainingJobDefinition'], args)

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_hyperparameter_tuning_job(client, args):
    """Create a Sagemaker HPO job"""
    request = create_hyperparameter_tuning_job_request(args)
    try:
        job_arn = client.create_hyper_parameter_tuning_job(**request)
        hpo_job_name = request['HyperParameterTuningJobName']
        logging.info("Created Hyperparameter Training Job with name: " + hpo_job_name)
        logging.info("HPO job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/hyper-tuning-jobs/{}"
            .format(args['region'], args['region'], hpo_job_name))
        logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix"
            .format(args['region'], args['region'], hpo_job_name))
        return hpo_job_name
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def wait_for_hyperparameter_training_job(client, hpo_job_name):
    ### Wait until the job finishes
    while(True):
        response = client.describe_hyper_parameter_tuning_job(HyperParameterTuningJobName=hpo_job_name)
        status = response['HyperParameterTuningJobStatus']
        if status == 'Completed':
            logging.info("Hyperparameter tuning job ended with status: " + status)
            break
        if status == 'Failed':
            message = response['FailureReason']
            logging.error('Hyperparameter tuning failed with the following error: {}'.format(message))
            raise Exception('Hyperparameter tuning job failed')
        logging.info("Hyperparameter tuning job is still in status: " + status)
        time.sleep(30)


def get_best_training_job_and_hyperparameters(client, hpo_job_name):
    ### Get and return best training job and its hyperparameters, without the objective metric
    info = client.describe_hyper_parameter_tuning_job(HyperParameterTuningJobName=hpo_job_name)
    best_job = info['BestTrainingJob']['TrainingJobName']
    training_info = client.describe_training_job(TrainingJobName=best_job)
    train_hyperparameters = training_info['HyperParameters']
    train_hyperparameters.pop('_tuning_objective_metric')
    return best_job, train_hyperparameters


def create_workteam(client, args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_workteam
    """Create a workteam"""
    with open(os.path.join(__cwd__, 'workteam.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    request['WorkteamName'] = args['team_name']
    request['Description'] = args['description']

    if args['sns_topic']:
    	request['NotificationConfiguration']['NotificationTopicArn'] = args['sns_topic']
    else:
        request.pop('NotificationConfiguration')

    for group in [n.strip() for n in args['user_groups'].split(',')]:
        request['MemberDefinitions'].append({'CognitoMemberDefinition': {'UserPool': args['user_pool'], 'UserGroup': group, 'ClientId': args['client_id']}})

    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    try:
        response = client.create_workteam(**request)
        portal = client.describe_workteam(WorkteamName=args['team_name'])['Workteam']['SubDomain']
        logging.info("Labeling portal: " + portal)
        return response['WorkteamArn']
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def create_labeling_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
    with open(os.path.join(__cwd__, 'gt.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    # Mapping are extracted from ARNs listed in https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
    algorithm_arn_map = {'us-west-2': '081040173940',
              'us-east-1': '432418664414',
              'us-east-2': '266458841044',
              'eu-west-1': '568282634449',
              'ap-northeast-1': '477331159723',
              'ap-southeast-1': '454466003867'}

    task_map = {'bounding box': 'BoundingBox',
              'image classification': 'ImageMultiClass',
              'semantic segmentation': 'SemanticSegmentation',
              'text classification': 'TextMultiClass'}

    auto_labeling_map = {'bounding box': 'object-detection',
              'image classification': 'image-classification',
              'text classification': 'text-classification'}

    task = args['task_type'].lower()

    request['LabelingJobName'] = args['job_name'] if args['job_name'] else "LabelingJob-" + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

    if args['label_attribute_name']:
        name_check = args['label_attribute_name'].split('-')[-1]
        if task == 'semantic segmentation' and name_check == 'ref' or task != 'semantic segmentation' and name_check != 'metadata' and name_check != 'ref':
            request['LabelAttributeName'] = args['label_attribute_name']
        else:
            logging.error('Invalid label attribute name. If task type is semantic segmentation, name must end in "-ref". Else, name must not end in "-ref" or "-metadata".')
    else:
        request['LabelAttributeName'] = args['job_name']

    request['InputConfig']['DataSource']['S3DataSource']['ManifestS3Uri'] = args['manifest_location']
    request['OutputConfig']['S3OutputPath'] = args['output_location']
    request['OutputConfig']['KmsKeyId'] = args['output_encryption_key']
    request['RoleArn'] = args['role']
    request['LabelCategoryConfigS3Uri'] = args['label_category_config']

    ### Update or pop stopping conditions
    if not args['max_human_labeled_objects'] and not args['max_percent_objects']:
        request.pop('StoppingConditions')
    else:
        if args['max_human_labeled_objects']:
            request['StoppingConditions']['MaxHumanLabeledObjectCount'] = args['max_human_labeled_objects']
        else:
            request['StoppingConditions'].pop('MaxHumanLabeledObjectCount')
        if args['max_percent_objects']:
            request['StoppingConditions']['MaxPercentageOfInputDatasetLabeled'] = args['max_percent_objects']
        else:
            request['StoppingConditions'].pop('MaxPercentageOfInputDatasetLabeled')

    ### Update or pop automatic labeling configs
    if args['enable_auto_labeling']:
        if task == 'image classification' or task == 'bounding box' or task == 'text classification':
            labeling_algorithm_arn = 'arn:aws:sagemaker:{}:027400017018:labeling-job-algorithm-specification/image-classification'.format(args['region'], auto_labeling_map[task])
            request['LabelingJobAlgorithmsConfig']['LabelingJobAlgorithmSpecificationArn'] = labeling_algorithm_arn
            if args['initial_model_arn']:
                request['LabelingJobAlgorithmsConfig']['InitialActiveLearningModelArn'] = args['initial_model_arn']
            else:
                request['LabelingJobAlgorithmsConfig'].pop('InitialActiveLearningModelArn')
            request['LabelingJobAlgorithmsConfig']['LabelingJobResourceConfig']['VolumeKmsKeyId'] = args['resource_encryption_key']
        else:
            logging.error("Automated data labeling not available for semantic segmentation or custom algorithms. Proceeding without automated data labeling.")
    else:
        request.pop('LabelingJobAlgorithmsConfig')

    ### Update pre-human and annotation consolidation task lambda functions
    if task == 'image classification' or task == 'bounding box' or task == 'text classification' or task == 'semantic segmentation':
        prehuman_arn = 'arn:aws:lambda:{}:{}:function:PRE-{}'.format(args['region'], algorithm_arn_map[args['region']], task_map[task])
        acs_arn = 'arn:aws:lambda:{}:{}:function:ACS-{}'.format(args['region'], algorithm_arn_map[args['region']], task_map[task])
        request['HumanTaskConfig']['PreHumanTaskLambdaArn'] = prehuman_arn
        request['HumanTaskConfig']['AnnotationConsolidationConfig']['AnnotationConsolidationLambdaArn'] = acs_arn
    elif task == 'custom' or task == '':
        if args['pre_human_task_function'] and args['post_human_task_function']:
            request['HumanTaskConfig']['PreHumanTaskLambdaArn'] = args['pre_human_task_function']
            request['HumanTaskConfig']['AnnotationConsolidationConfig']['AnnotationConsolidationLambdaArn'] = args['post_human_task_function']
        else:
            logging.error("Must specify pre-human task lambda arn and annotation consolidation post-human task lambda arn.")
    else:
        logging.error("Task type must be Bounding Box, Image Classification, Semantic Segmentation, Text Classification, or Custom.")

    request['HumanTaskConfig']['UiConfig']['UiTemplateS3Uri'] = args['ui_template']
    request['HumanTaskConfig']['TaskTitle'] = args['title']
    request['HumanTaskConfig']['TaskDescription'] = args['description']
    request['HumanTaskConfig']['NumberOfHumanWorkersPerDataObject'] = args['num_workers_per_object']
    request['HumanTaskConfig']['TaskTimeLimitInSeconds'] = args['time_limit']

    if args['task_availibility']:
        request['HumanTaskConfig']['TaskAvailabilityLifetimeInSeconds'] = args['task_availibility']
    else:
        request['HumanTaskConfig'].pop('TaskAvailabilityLifetimeInSeconds')

    if args['max_concurrent_tasks']:
        request['HumanTaskConfig']['MaxConcurrentTaskCount'] = args['max_concurrent_tasks']
    else:
        request['HumanTaskConfig'].pop('MaxConcurrentTaskCount')

    if args['task_keywords']:
    	for word in [n.strip() for n in args['task_keywords'].split(',')]:
            request['HumanTaskConfig']['TaskKeywords'].append(word)
    else:
	    request['HumanTaskConfig'].pop('TaskKeywords')

    ### Update worker configurations
    if args['worker_type'].lower() == 'public':
        if args['no_adult_content']:
            request['InputConfig']['DataAttributes']['ContentClassifiers'].append('FreeOfAdultContent')
        if args['no_ppi']:
            request['InputConfig']['DataAttributes']['ContentClassifiers'].append('FreeOfPersonallyIdentifiableInformation')

        request['HumanTaskConfig']['WorkteamArn'] = 'arn:aws:sagemaker:{}:394669845002:workteam/public-crowd/default'.format(args['region'])

        dollars = int(args['workforce_task_price'])
        cents = int(100 * (args['workforce_task_price'] - dollars))
        tenth_of_cents = int((args['workforce_task_price'] * 1000) - (dollars * 1000) - (cents * 10))
        request['HumanTaskConfig']['PublicWorkforceTaskPrice']['AmountInUsd']['Dollars'] = dollars
        request['HumanTaskConfig']['PublicWorkforceTaskPrice']['AmountInUsd']['Cents'] = cents
        request['HumanTaskConfig']['PublicWorkforceTaskPrice']['AmountInUsd']['TenthFractionsOfACent'] = tenth_of_cents
    else:
        request['InputConfig'].pop('DataAttributes')
        request['HumanTaskConfig']['WorkteamArn'] = args['workteam_arn']
        request['HumanTaskConfig'].pop('PublicWorkforceTaskPrice')

    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_labeling_job(client, args):
    """Create a SageMaker Ground Truth job"""
    request = create_labeling_job_request(args)
    try:
        client.create_labeling_job(**request)
        gt_job_name = request['LabelingJobName']
        logging.info("Created Ground Truth Labeling Job with name: " + gt_job_name)
        logging.info("Ground Truth job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/groundtruth?region={}#/labeling-jobs/details/{}"
            .format(args['region'], args['region'], gt_job_name))
        return gt_job_name
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def wait_for_labeling_job(client, labeling_job_name):
    ### Wait until the job finishes
    status = 'InProgress'
    while(status == 'InProgress'):
        response = client.describe_labeling_job(LabelingJobName=labeling_job_name)
        status = response['LabelingJobStatus']
        if status == 'Failed':
            message = response['FailureReason']
            logging.info('Labeling failed with the following error: {}'.format(message))
            raise Exception('Labeling job failed')
        logging.info("Labeling job is still in status: " + status)
        time.sleep(30)

    if status == 'Completed':
        logging.info("Labeling job ended with status: " + status)
    else:
        raise Exception('Labeling job stopped')


def get_labeling_job_outputs(client, labeling_job_name, auto_labeling):
    ### Get and return labeling job outputs
    info = client.describe_labeling_job(LabelingJobName=labeling_job_name)
    output_manifest = info['LabelingJobOutput']['OutputDatasetS3Uri']
    if auto_labeling:
        active_learning_model_arn = info['LabelingJobOutput']['FinalActiveLearningModelArn']
    else:
        active_learning_model_arn = ''
    return output_manifest, active_learning_model_arn

def enable_spot_instance_support(training_job_config, args):
    if args['spot_instance']:
        training_job_config['EnableManagedSpotTraining'] = args['spot_instance']
        if args['max_wait_time'] >= training_job_config['StoppingCondition']['MaxRuntimeInSeconds']:
            training_job_config['StoppingCondition']['MaxWaitTimeInSeconds'] = args['max_wait_time']
        else:
            logging.error("Max wait time must be greater than or equal to max run time.")
            raise Exception('Could not create job request.')
        
        if args['checkpoint_config'] and 'S3Uri' in args['checkpoint_config']:
            training_job_config['CheckpointConfig'] = args['checkpoint_config']
        else:
            logging.error("EnableManagedSpotTraining requires checkpoint config with an S3 uri.")
            raise Exception('Could not create job request.')
    else:
        # Remove any artifacts that require spot instance support
        del training_job_config['StoppingCondition']['MaxWaitTimeInSeconds']
        del training_job_config['CheckpointConfig']


def id_generator(size=4, chars=string.ascii_uppercase + string.digits):
  return ''.join(random.choice(chars) for _ in range(size))


def str_to_bool(s):
  if s.lower().strip() == 'true':
    return True
  elif s.lower().strip() == 'false':
    return False
  else:
    raise argparse.ArgumentTypeError('"True" or "False" expected.')

def str_to_int(s):
  if s:
    return int(s)
  else:
    return 0

def str_to_float(s):
  if s:
    return float(s)
  else:
    return 0.0

def str_to_json_dict(s):
  if s != '':
      return json.loads(s)
  else:
      return {}

def str_to_json_list(s):
  if s != '':
      return json.loads(s)
  else:
      return []
