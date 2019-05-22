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
from urlparse import urlparse

import boto3
from botocore.exceptions import ClientError
from sagemaker import get_execution_role

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

def id_generator(size=4, chars=string.ascii_uppercase + string.digits):
  return ''.join(random.choice(chars) for _ in range(size))