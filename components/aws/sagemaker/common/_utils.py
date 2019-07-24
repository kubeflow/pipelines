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
import os
import subprocess
from time import gmtime, strftime
import time
import string
import random
import json
import yaml
from urlparse import urlparse

import boto3
from botocore.exceptions import ClientError
from sagemaker import get_execution_role
from sagemaker.amazon.amazon_estimator import get_image_uri

import logging
logging.getLogger().setLevel(logging.INFO)

def get_client(region=None):
    """Builds a client to the AWS SageMaker API."""
    client = boto3.client('sagemaker', region_name=region)
    return client

def get_sagemaker_role():
  return get_execution_role()

def create_training_job(client, image, instance_type, instance_count, volume_size, data_location, output_location, role):
  """Create a Sagemaker training job."""
  job_name = 'TrainingJob-' + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

  create_training_params = \
  {
      "AlgorithmSpecification": {
          "TrainingImage": image,
          "TrainingInputMode": "File"
      },
      "RoleArn": role,
      "OutputDataConfig": {
          "S3OutputPath": output_location
      },
      "ResourceConfig": {
          "InstanceCount": instance_count,
          "InstanceType": instance_type,
          "VolumeSizeInGB": volume_size
      },
      "TrainingJobName": job_name,
      "HyperParameters": {
          "k": "10",
          "feature_dim": "784",
          "mini_batch_size": "500"
      },
      "StoppingCondition": {
          "MaxRuntimeInSeconds": 60 * 60
      },
      "InputDataConfig": [
          {
              "ChannelName": "train",
              "DataSource": {
                  "S3DataSource": {
                      "S3DataType": "S3Prefix",
                      "S3Uri": data_location,
                      "S3DataDistributionType": "FullyReplicated"
                  }
              },
              "CompressionType": "None",
              "RecordWrapperType": "None"
          }
      ]
  }
  client.create_training_job(**create_training_params)
  return job_name


def deploy_model(client, model_name):
  endpoint_config_name = create_endpoint_config(client, model_name)
  endpoint_name = create_endpoint(client, endpoint_config_name)
  return endpoint_name


def get_model_artifacts_from_job(client, job_name):
  info = client.describe_training_job(TrainingJobName=job_name)
  model_artifact_url = info['ModelArtifacts']['S3ModelArtifacts']
  return model_artifact_url

def create_model(client, model_artifact_url, model_name, image, role):
  primary_container = {
      'Image': image,
      'ModelDataUrl': model_artifact_url
  }

  create_model_response = client.create_model(
      ModelName = model_name,
      ExecutionRoleArn = role,
      PrimaryContainer = primary_container)

  print("Model Config Arn: " + create_model_response['ModelArn'])
  return create_model_response['ModelArn']

def create_endpoint_config(client, model_name):
  endpoint_config_name = 'EndpointConfig' + model_name[model_name.index('-'):]
  print(endpoint_config_name)
  create_endpoint_config_response = client.create_endpoint_config(
    EndpointConfigName = endpoint_config_name,
    ProductionVariants=[{
        'InstanceType':'ml.m4.xlarge',
        'InitialInstanceCount':1,
        'ModelName': model_name,
        'VariantName':'AllTraffic'}])

  print("Endpoint Config Arn: " + create_endpoint_config_response['EndpointConfigArn'])
  return endpoint_config_name


def create_endpoint(client, endpoint_config_name):
  endpoint_name = 'Endpoint' + endpoint_config_name[endpoint_config_name.index('-'):]
  print(endpoint_name)
  create_endpoint_response = client.create_endpoint(
      EndpointName=endpoint_name,
      EndpointConfigName=endpoint_config_name)
  print(create_endpoint_response['EndpointArn'])

  resp = client.describe_endpoint(EndpointName=endpoint_name)
  status = resp['EndpointStatus']
  print("Status: " + status)

  try:
    client.get_waiter('endpoint_in_service').wait(EndpointName=endpoint_name)
  finally:
    resp = client.describe_endpoint(EndpointName=endpoint_name)
    status = resp['EndpointStatus']
    print("Arn: " + resp['EndpointArn'])
    print("Create endpoint ended with status: " + status)
    return endpoint_name

    if status != 'InService':
      message = client.describe_endpoint(EndpointName=endpoint_name)['FailureReason']
      print('Create endpoint failed with the following error: {}'.format(message))
      raise Exception('Endpoint creation did not succeed')

def wait_for_training_job(client, job_name):
  status = client.describe_training_job(TrainingJobName=job_name)['TrainingJobStatus']
  print(status)
  try:
    client.get_waiter('training_job_completed_or_stopped').wait(TrainingJobName=job_name)
  finally:
    status = client.describe_training_job(TrainingJobName=job_name)['TrainingJobStatus']
    print("Training job ended with status: " + status)
    if status == 'Failed':
        message = client.describe_training_job(TrainingJobName=job_name)['FailureReason']
        print('Training failed with the following error: {}'.format(message))
        raise Exception('Training job failed')

def create_transform_job(client, model_name, input_location, output_location):
  batch_job_name = 'BatchTransform' + model_name[model_name.index('-'):]

  ### Create a transform job
  request = \
  {
      "TransformJobName": batch_job_name,
      "ModelName": model_name,
      "MaxConcurrentTransforms": 4,
      "MaxPayloadInMB": 6,
      "BatchStrategy": "MultiRecord",
      "TransformOutput": {
          "S3OutputPath": output_location
      },
      "TransformInput": {
          "DataSource": {
              "S3DataSource": {
                  "S3DataType": "S3Prefix",
                  "S3Uri": input_location
              }
          },
          "ContentType": "text/csv",
          "SplitType": "Line",
          "CompressionType": "None"
      },
      "TransformResources": {
          "InstanceType": "ml.m4.xlarge",
          "InstanceCount": 1
      }
  }

  client.create_transform_job(**request)

  print("Created Transform job with name: ", batch_job_name)
  return batch_job_name


def wait_for_transform_job(client, batch_job_name):
  ### Wait until the job finishes
  while(True):
    response = client.describe_transform_job(TransformJobName=batch_job_name)
    status = response['TransformJobStatus']
    if  status == 'Completed':
      print("Transform job ended with status: " + status)
      break
    if status == 'Failed':
      message = response['FailureReason']
      print('Transform failed with the following error: {}'.format(message))
      raise Exception('Transform job failed')
    print("Transform job is still in status: " + status)
    time.sleep(30)

def print_tranformation_job_result(output_location):
  ### Fetch the transform output
  bucket = urlparse(output_location).netloc
  output_key = "{}/valid_data.csv.out".format(urlparse(output_location).path.lstrip('/'))
  s3_client = boto3.client('s3')
  s3_client.download_file(bucket, output_key, 'valid-result')
  with open('valid-result') as f:
    results = f.readlines()
  print("Sample transform result: {}".format(results[0]))


def create_hyperparameter_tuning_job_request(args):
    with open('/app/common/hpo.template.yaml', 'r') as f:
        request = yaml.safe_load(f)

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

    ### Create a hyperparameter tuning job
    request['HyperParameterTuningJobName'] = args['job_name']

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
        # TODO: determine if users can make custom algorithm resources that have the same name as built-in algorithm names
        algo_name = args['algorithm_name'].lower().strip()
        if algo_name in built_in_algos.keys():
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], built_in_algos[algo_name])
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
        # Just to give the user more leeway for built-in algorithm name inputs
        elif algo_name in built_in_algos.values():
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = get_image_uri(args['region'], algo_name)
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
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
        logging.info("URL: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/hyper-tuning-jobs/{}".format(args['region'], args['region'], hpo_job_name))
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
