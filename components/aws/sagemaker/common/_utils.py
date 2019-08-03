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


import datetime
import subprocess
import argparse
from time import gmtime, strftime
import time
import string
import random
import json
import yaml
from urlparse import urlparse

import boto3
from botocore.exceptions import ClientError
from sagemaker.amazon.amazon_estimator import get_image_uri

import logging
logging.getLogger().setLevel(logging.INFO)

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

def get_client(region=None):
    """Builds a client to the AWS SageMaker API."""
    client = boto3.client('sagemaker', region_name=region)
    return client


def create_training_job_request(args):
    with open('/app/common/train.template.yaml', 'r') as f:
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
        # TODO: determine if users can make custom algorithm resources that have the same name as built-in algorithm names
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
    else:
        logging.error("Must specify at least one input channel.")
        raise Exception('Could not make job request')

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


def deploy_model(client, args):
  endpoint_config_name = create_endpoint_config(client, args)
  endpoint_name = create_endpoint(client, args['region'], args['endpoint_name'], endpoint_config_name, args['endpoint_tags'])
  return endpoint_name


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
    with open('/app/common/model.template.yaml', 'r') as f:
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

def create_endpoint_config(client, args):
    with open('/app/common/endpoint_config.template.yaml', 'r') as f:
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

def create_transform_job_request(args):
    with open('/app/common/transform.template.yaml', 'r') as f:
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
      logging.info("Transform job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/jobs/{}"
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

def print_tranformation_job_result(output_location):
  ### Fetch the transform output
  bucket = urlparse(output_location).netloc
  output_key = "{}/valid_data.csv.out".format(urlparse(output_location).path.lstrip('/'))
  s3_client = boto3.client('s3')
  s3_client.download_file(bucket, output_key, 'valid-result')
  with open('valid-result') as f:
    results = f.readlines()
  logging.info("Sample transform result: {}".format(results[0]))


def create_hyperparameter_tuning_job_request(args):
    with open('/app/common/hpo.template.yaml', 'r') as f:
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
