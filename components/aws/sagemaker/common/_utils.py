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
from distutils.util import strtobool
import time
import string
import random
import json
import yaml
import re
import json
from pathlib2 import Path
from enum import Enum, auto

import boto3
from boto3.session import Session

from botocore.config import Config
from botocore.credentials import (
    AssumeRoleCredentialFetcher,
    CredentialResolver,
    DeferredRefreshableCredentials,
    JSONFileCache
)
from botocore.exceptions import ClientError
from botocore.session import Session as BotocoreSession

from sagemaker.image_uris import retrieve

import logging
logging.getLogger().setLevel(logging.INFO)

# this error message is used in integration tests
CW_ERROR_MESSAGE = 'Error in fetching CloudWatch logs for SageMaker job'

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


def nullable_string_argument(value):
    value = value.strip()
    if not value:
        return None
    return value


def add_default_client_arguments(parser):
    parser.add_argument('--region', type=str, required=True, help='The region where the training job launches.')
    parser.add_argument('--endpoint_url', type=nullable_string_argument, required=False, help='The URL to use when communicating with the SageMaker service.')
    parser.add_argument('--assume_role', type=nullable_string_argument, required=False, help='The ARN of an IAM role to assume when connecting to SageMaker.')


def get_component_version():
    """Get component version from the first line of License file"""
    component_version = 'NULL'

    # Get license file using known common directory
    license_file_path = os.path.abspath(os.path.join(__cwd__, '../THIRD-PARTY-LICENSES.txt'))
    with open(license_file_path, 'r') as license_file:
        version_match = re.search('Amazon SageMaker Components for Kubeflow Pipelines; version (([0-9]+[.])+[0-9]+)',
                        license_file.readline())
        if version_match is not None:
            component_version = version_match.group(1)

    return component_version


def print_log_header(header_len, title=""):
    logging.info(f"{title:*^{header_len}}")

def print_logs_for_job(cw_client, log_grp, job_name):
    """Gets the CloudWatch logs for SageMaker jobs"""
    try:
        logging.info('\n******************** CloudWatch logs for {} {} ********************\n'.format(log_grp, job_name))

        log_streams = cw_client.describe_log_streams(
            logGroupName=log_grp,
            logStreamNamePrefix=job_name + '/'
        )['logStreams']

        for log_stream in log_streams:
            logging.info('\n***** {} *****\n'.format(log_stream['logStreamName']))
            response = cw_client.get_log_events(
                logGroupName=log_grp,
                logStreamName=log_stream['logStreamName']
            )
            for event in response['events']:
                logging.info(event['message'])

        logging.info('\n******************** End of CloudWatch logs for {} {} ********************\n'.format(log_grp, job_name))
    except Exception as e:
        logging.error(CW_ERROR_MESSAGE)
        logging.error(e)


class AssumeRoleProvider(object):
    METHOD = 'assume-role'

    def __init__(self, fetcher):
        self._fetcher = fetcher

    def load(self):
        return DeferredRefreshableCredentials(
            self._fetcher.fetch_credentials,
            self.METHOD
        )

def get_boto3_session(region, role_arn=None):
    """Creates a boto3 session, optionally assuming a role"""

    # By default return a basic session
    if role_arn is None:
        return Session(region_name=region)

    # The following assume role example was taken from
    # https://github.com/boto/botocore/issues/761#issuecomment-426037853

    # Create a session used to assume role
    assume_session = BotocoreSession()
    fetcher = AssumeRoleCredentialFetcher(
        assume_session.create_client,
        assume_session.get_credentials(),
        role_arn,
        extra_args={
            'DurationSeconds': 3600, # 1 hour assume assume by default
        },
        cache=JSONFileCache()
    )
    role_session = BotocoreSession()
    role_session.register_component(
        'credential_provider',
        CredentialResolver([AssumeRoleProvider(fetcher)])
    )
    return Session(region_name=region, botocore_session=role_session)

def get_sagemaker_client(region, endpoint_url=None, assume_role_arn=None):
    """Builds a client to the AWS SageMaker API."""
    session = get_boto3_session(region, assume_role_arn)
    session_config = Config(
        user_agent='sagemaker-on-kubeflow-pipelines-v{}'.format(get_component_version())
    )
    client = session.client('sagemaker', endpoint_url=endpoint_url, config=session_config)
    return client


def get_cloudwatch_client(region, assume_role_arn=None):
    session = get_boto3_session(region, assume_role_arn)
    client = session.client('logs')
    return client


def create_training_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_training_job
    with open(os.path.join(__cwd__, 'train.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    job_name = args['job_name'] if args['job_name'] else 'TrainingJob-' + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

    request['TrainingJobName'] = job_name
    request['RoleArn'] = args['role']
    request['HyperParameters'] = create_hyperparameters(args['hyperparameters'])
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
            request['AlgorithmSpecification']['TrainingImage'] = retrieve(built_in_algos[algo_name], args['region'])
            request['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        # Just to give the user more leeway for built-in algorithm name inputs
        elif algo_name in built_in_algos.values():
            request['AlgorithmSpecification']['TrainingImage'] = retrieve(algo_name, args['region'])
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
        request['VpcConfig']['SecurityGroupIds'] = args['vpc_security_group_ids'].split(',')
        request['VpcConfig']['Subnets'] = args['vpc_subnets'].split(',')
    else:
        request.pop('VpcConfig')

    ### Update input channels, must have at least one specified
    if len(args['channels']) > 0:
        request['InputDataConfig'] = args['channels']
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

    ### Update DebugHookConfig and DebugRuleConfigurations
    if args['debug_hook_config']:
        request['DebugHookConfig'] = args['debug_hook_config']
    else:
        request.pop('DebugHookConfig')

    if args['debug_rule_config']:
        request['DebugRuleConfigurations'] = args['debug_rule_config']
    else:
        request.pop('DebugRuleConfigurations')

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_training_job(client, args):
  """Create a SageMaker training job."""
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


def wait_for_training_job(client, training_job_name, poll_interval=30):
    while(True):
        response = client.describe_training_job(TrainingJobName=training_job_name)
        status = response['TrainingJobStatus']
        if status == 'Completed':
            logging.info("Training job ended with status: " + status)
            break
        if status == 'Failed':
            message = response['FailureReason']
            logging.info(f'Training failed with the following error: {message}')
            raise Exception('Training job failed')
        logging.info("Training job is still in status: " + status)
        time.sleep(poll_interval)


def wait_for_debug_rules(client, training_job_name, poll_interval=30):
    first_poll = True
    while(True):
        response = client.describe_training_job(TrainingJobName=training_job_name)
        if 'DebugRuleEvaluationStatuses' not in response:
            break
        if first_poll:
            logging.info("Polling for status of all debug rules:")
            first_poll = False
        if DebugRulesStatus.from_describe(response) != DebugRulesStatus.INPROGRESS:
            logging.info("Rules have ended with status:\n")
            print_debug_rule_status(response, True)
            break
        print_debug_rule_status(response)
        time.sleep(poll_interval)


class DebugRulesStatus(Enum):
    COMPLETED = auto()
    ERRORED = auto()
    INPROGRESS = auto()

    @classmethod
    def from_describe(self, response):
        has_error = False
        for debug_rule in response['DebugRuleEvaluationStatuses']:
            if debug_rule['RuleEvaluationStatus'] == "Error":
                has_error = True
            if debug_rule['RuleEvaluationStatus'] == "InProgress":
                return DebugRulesStatus.INPROGRESS
        if has_error:
            return DebugRulesStatus.ERRORED
        else:
            return DebugRulesStatus.COMPLETED


def print_debug_rule_status(response, last_print=False):
    """
    Example of DebugRuleEvaluationStatuses:
    response['DebugRuleEvaluationStatuses'] =
        [{
            "RuleConfigurationName": "VanishingGradient",
            "RuleEvaluationStatus": "IssuesFound",
            "StatusDetails": "There was an issue."
        }]

    If last_print is False:
    INFO:root: - LossNotDecreasing: InProgress
    INFO:root: - Overtraining: NoIssuesFound
    ERROR:root:- CustomGradientRule: Error

    If last_print is True:
    INFO:root: - LossNotDecreasing: IssuesFound
    INFO:root:   - RuleEvaluationConditionMet: Evaluation of the rule LossNotDecreasing at step 10 resulted in the condition being met
    """
    for debug_rule in response['DebugRuleEvaluationStatuses']:
        line_ending = "\n" if last_print else ""
        if 'StatusDetails' in debug_rule:
            status_details = f"- {debug_rule['StatusDetails'].rstrip()}{line_ending}"
            line_ending = ""
        else:
            status_details = ""
        rule_status = f"- {debug_rule['RuleConfigurationName']}: {debug_rule['RuleEvaluationStatus']}{line_ending}"
        if debug_rule['RuleEvaluationStatus'] == "Error":
            log = logging.error
            status_padding = 1
        else:
            log = logging.info
            status_padding = 2

        log(f"{status_padding * ' '}{rule_status}")
        if last_print and status_details:
            log(f"{(status_padding + 2) * ' '}{status_details}")
    print_log_header(50)


def get_model_artifacts_from_job(client, job_name):
  info = client.describe_training_job(TrainingJobName=job_name)
  model_artifact_url = info['ModelArtifacts']['S3ModelArtifacts']
  return model_artifact_url


def get_image_from_job(client, job_name):
    info = client.describe_training_job(TrainingJobName=job_name)
    if 'TrainingImage' in info['AlgorithmSpecification']:
        image = info['AlgorithmSpecification']['TrainingImage']
    else:
        algorithm_name = info['AlgorithmSpecification']['AlgorithmName']
        image = client.describe_algorithm(AlgorithmName=algorithm_name)['TrainingSpecification']['TrainingImage']

    return image


def stop_training_job(client, job_name):
    response = client.describe_training_job(TrainingJobName=job_name)
    if response["TrainingJobStatus"] == "InProgress":
        try:
            client.stop_training_job(TrainingJobName=job_name)
            return job_name
        except ClientError as e:
            raise Exception(e.response['Error']['Message'])


def create_model(client, args):
    request = create_model_request(args)
    try:
        create_model_response = client.create_model(**request)
        logging.info("Model Config Arn: " + create_model_response['ModelArn'])
        return create_model_response['ModelArn']
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def create_model_request(args):
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

        if args['model_package']:
            request['PrimaryContainer']['ModelPackageName'] = args['model_package']
            request['PrimaryContainer'].pop('Image')
            request['PrimaryContainer'].pop('ModelDataUrl')
        elif args['image'] and args['model_artifact_url']:
            request['PrimaryContainer']['Image'] = args['image']
            request['PrimaryContainer']['ModelDataUrl'] = args['model_artifact_url']
            request['PrimaryContainer'].pop('ModelPackageName')
        else:
            logging.error("Please specify an image AND model artifact url, OR a model package name.")
            raise Exception("Could not make create model request.")

    request['ExecutionRoleArn'] = args['role']
    request['EnableNetworkIsolation'] = args['network_isolation']

    ### Update or pop VPC configs
    if args['vpc_security_group_ids'] and args['vpc_subnets']:
        request['VpcConfig']['SecurityGroupIds'] = args['vpc_security_group_ids'].split(',')
        request['VpcConfig']['Subnets'] = args['vpc_subnets'].split(',')
    else:
        request.pop('VpcConfig')

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request

def endpoint_name_exists(client, endpoint_name):
  try:
      endpoint_name = client.describe_endpoint(EndpointName=endpoint_name)['EndpointName']
      logging.info("Endpoint exists: " + endpoint_name)
      return True
  except ClientError as e:
      logging.debug("Endpoint does not exist")
  return False

def endpoint_config_name_exists(client, endpoint_config_name):
  try:
      config_name = client.describe_endpoint_config(EndpointConfigName=endpoint_config_name)['EndpointConfigName']
      logging.info("Endpoint Config exists: " + config_name)
      return True
  except ClientError as e:
      logging.info("Endpoint Config does not exist:" + endpoint_config_name)
  return False

def update_deployed_model(client, args):
    endpoint_config_name = create_endpoint_config(client, args)
    endpoint_name = update_endpoint(client, args['region'], args['endpoint_name'], endpoint_config_name)
    return endpoint_name

def deploy_model(client, args):
    args['update_endpoint'] = False
    endpoint_config_name = create_endpoint_config(client, args)
    endpoint_name = create_endpoint(client, args['region'], args['endpoint_name'], endpoint_config_name, args['endpoint_tags'])
    return endpoint_name

def get_endpoint_config(client, endpoint_name):
    endpoint_config_name = None
    try:
        endpoint_config_name = client.describe_endpoint(EndpointName=endpoint_name)['EndpointConfigName']
        logging.info("Current Endpoint Config Name: " + endpoint_config_name)
    except ClientError as e:
        logging.info("Endpoint Config does not exist")
        ## This is not an error, end point may not exist
    return endpoint_config_name

def delete_endpoint_config(client, endpoint_config_name):
    try:
        client.delete_endpoint_config(EndpointConfigName=endpoint_config_name)
        return True
    except ClientError as e:
        logging.info("Endpoint config may not exist to be deleted: " + e.response['Error']['Message'])
    return False

def create_endpoint_config_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_endpoint_config
    with open(os.path.join(__cwd__, 'endpoint_config.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    if not args['model_name_1']:
        logging.error("Must specify at least one model (model name) to host.")
        raise Exception("Could not create endpoint config.")

    endpoint_config_name = args['endpoint_config_name']
    request['EndpointConfigName'] = endpoint_config_name

    if args['resource_encryption_key']:
        request['KmsKeyId'] = args['resource_encryption_key']
    else:
        request.pop('KmsKeyId')

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

    return request

def get_endpoint_config_name(client, args):
    ## boto3 documentation says to update an endpoint, a new EndPointConfig has to be created
    ## and the one currently in use should NOT be deleted. Appending a random number to resolve conflict
    endpoint_config_name = args['endpoint_config_name'] if args['endpoint_config_name'] else 'EndpointConfig' + "-" + id_generator(8)
    if args['update_endpoint'] and endpoint_config_name_exists(client, endpoint_config_name):
        endpoint_config_name = endpoint_config_name + "-" + id_generator(8)
        logging.info("Changed endpoint_config_name to: " + endpoint_config_name)
    return endpoint_config_name

def create_endpoint_config(client, args):
    args['endpoint_config_name'] = get_endpoint_config_name(client, args)
    request = create_endpoint_config_request(args)
    try:
        create_endpoint_config_response = client.create_endpoint_config(**request)
        logging.info("Endpoint configuration in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpointConfig/{}"
            .format(args['region'], args['region'], request['EndpointConfigName']))
        logging.info("Endpoint Config Arn: " + create_endpoint_config_response['EndpointConfigArn'])
        return request['EndpointConfigName']
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])

def update_endpoint(client, region, endpoint_name, endpoint_config_name):
  try:
      update_endpoint_response = client.update_endpoint(
          EndpointName=endpoint_name,
          EndpointConfigName=endpoint_config_name,
          )
      logging.info("Updating endpoint with name: " + endpoint_name)
      logging.info("Endpoint in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/endpoints/{}"
          .format(region, region, endpoint_name))
      logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/Endpoints/{};streamFilter=typeLogStreamPrefix"
          .format(region, region, endpoint_name))
      return endpoint_name
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

    if status != 'InService':
      message = resp['FailureReason']
      logging.info('Create/Update endpoint failed with the following error: {}'.format(message))
      raise Exception('Endpoint creation did not succeed')

    logging.info("Endpoint Arn: " + resp['EndpointArn'])
    logging.info("Create/Update endpoint ended with status: " + status)

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


def wait_for_transform_job(client, batch_job_name, poll_interval=30):
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
    time.sleep(poll_interval)


def stop_transform_job(client, job_name):
    try:
        client.stop_transform_job(TransformJobName=job_name)
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


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

    request['TrainingJobDefinition']['StaticHyperParameters'] = create_hyperparameters(args['static_parameters'])
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
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = retrieve(built_in_algos[algo_name], args['region'])
            request['TrainingJobDefinition']['AlgorithmSpecification'].pop('AlgorithmName')
            logging.warning('Algorithm name is found as an Amazon built-in algorithm. Using built-in algorithm.')
        # To give the user more leeway for built-in algorithm name inputs
        elif algo_name in built_in_algos.values():
            request['TrainingJobDefinition']['AlgorithmSpecification']['TrainingImage'] = retrieve(algo_name, args['region'])
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
        request['TrainingJobDefinition']['VpcConfig']['SecurityGroupIds'] = args['vpc_security_group_ids'].split(',')
        request['TrainingJobDefinition']['VpcConfig']['Subnets'] = args['vpc_subnets'].split(',')
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

    enable_spot_instance_support(request['TrainingJobDefinition'], args)

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_hyperparameter_tuning_job(client, args):
    """Create a SageMaker HPO job"""
    request = create_hyperparameter_tuning_job_request(args)
    try:
        client.create_hyper_parameter_tuning_job(**request)
        hpo_job_name = request['HyperParameterTuningJobName']
        logging.info("Created Hyperparameter Tuning Job with name: " + hpo_job_name)
        logging.info("HPO job in SageMaker: https://{}.console.aws.amazon.com/sagemaker/home?region={}#/hyper-tuning-jobs/{}"
            .format(args['region'], args['region'], hpo_job_name))
        logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/TrainingJobs;prefix={};streamFilter=typeLogStreamPrefix"
            .format(args['region'], args['region'], hpo_job_name))
        return hpo_job_name
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def wait_for_hyperparameter_training_job(client, hpo_job_name, poll_interval=30):
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
        time.sleep(poll_interval)


def get_best_training_job_and_hyperparameters(client, hpo_job_name):
    ### Get and return best training job and its hyperparameters, without the objective metric
    info = client.describe_hyper_parameter_tuning_job(HyperParameterTuningJobName=hpo_job_name)
    best_job = info['BestTrainingJob']['TrainingJobName']
    training_info = client.describe_training_job(TrainingJobName=best_job)
    train_hyperparameters = training_info['HyperParameters']
    train_hyperparameters.pop('_tuning_objective_metric')
    return best_job, train_hyperparameters


def stop_hyperparameter_tuning_job(client, job_name):
    try:
        client.stop_hyper_parameter_tuning_job(HyperParameterTuningJobName=job_name)
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def create_workteam(client, args):
    try:
        request = create_workteam_request(args)
        response = client.create_workteam(**request)
        portal = client.describe_workteam(WorkteamName=args['team_name'])['Workteam']['SubDomain']
        logging.info("Labeling portal: " + portal)
        return response['WorkteamArn']
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])

def create_workteam_request(args):
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

    return request


def create_labeling_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
    with open(os.path.join(__cwd__, 'gt.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    # Mapping are extracted from ARNs listed in https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_labeling_job
    algorithm_arn_map = {'us-west-2': '081040173940',
              'us-east-1': '432418664414',
              'us-east-2': '266458841044',
              'ca-central-1': '918755190332',
              'eu-west-1': '568282634449',
              'eu-west-2': '487402164563',
              'eu-central-1': '203001061592',
              'ap-northeast-1': '477331159723',
              'ap-northeast-2': '845288260483',
              'ap-south-1': '565803892007',
              'ap-southeast-1': '377565633583',
              'ap-southeast-2': '454466003867'}

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
    
    ### Update or pop label category config s3 uri
    if not args['label_category_config']:
        request.pop('LabelCategoryConfigS3Uri')
    else:
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


def wait_for_labeling_job(client, labeling_job_name, poll_interval=30):
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
        time.sleep(poll_interval)

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
        active_learning_model_arn = ' '
    return output_manifest, active_learning_model_arn


def stop_labeling_job(client, job_name):
    try:
        client.stop_labeling_job(LabelingJobName=job_name)
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def create_hyperparameters(hyperparam_args):
    # Validate all values are strings
    for key, value in hyperparam_args.items():
        if not isinstance(value, str):
            raise Exception(f"Could not parse hyperparameters. Value for {key} was not a string.")

    return hyperparam_args

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

def create_processing_job_request(args):
    ### Documentation: https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/sagemaker.html#SageMaker.Client.create_processing_job
    with open(os.path.join(__cwd__, 'process.template.yaml'), 'r') as f:
        request = yaml.safe_load(f)

    job_name = args['job_name'] if args['job_name'] else 'ProcessingJob-' + strftime("%Y%m%d%H%M%S", gmtime()) + '-' + id_generator()

    request['ProcessingJobName'] = job_name
    request['RoleArn'] = args['role']

    ### Update processing container settings
    request['AppSpecification']['ImageUri'] = args['image']

    if args['container_entrypoint']:
        request['AppSpecification']['ContainerEntrypoint'] = args['container_entrypoint']
    else:
        request['AppSpecification'].pop('ContainerEntrypoint')
    if args['container_arguments']:
        request['AppSpecification']['ContainerArguments'] = args['container_arguments']
    else:
        request['AppSpecification'].pop('ContainerArguments')

    ### Update or pop VPC configs
    if args['vpc_security_group_ids'] and args['vpc_subnets']:
        request['NetworkConfig']['VpcConfig']['SecurityGroupIds'] = args['vpc_security_group_ids'].split(',')
        request['NetworkConfig']['VpcConfig']['Subnets'] = args['vpc_subnets'].split(',')
    else:
        request['NetworkConfig'].pop('VpcConfig')
    request['NetworkConfig']['EnableNetworkIsolation'] = args['network_isolation']
    request['NetworkConfig']['EnableInterContainerTrafficEncryption'] = args['traffic_encryption']

    ### Update input channels, not a required field
    if args['input_config']:
        request['ProcessingInputs'] = args['input_config']
    else:
        request.pop('ProcessingInputs')

    ### Update output channels, must have at least one specified
    if len(args['output_config']) > 0:
        request['ProcessingOutputConfig']['Outputs'] = args['output_config']
    else:
        logging.error("Must specify at least one output channel.")
        raise Exception('Could not create job request')

    if args['output_encryption_key']:
        request['ProcessingOutputConfig']['KmsKeyId'] = args['output_encryption_key']
    else:
        request['ProcessingOutputConfig'].pop('KmsKeyId')

    ### Update cluster config resources
    request['ProcessingResources']['ClusterConfig']['InstanceType'] = args['instance_type']
    request['ProcessingResources']['ClusterConfig']['InstanceCount'] = args['instance_count']
    request['ProcessingResources']['ClusterConfig']['VolumeSizeInGB'] = args['volume_size']

    if args['resource_encryption_key']:
        request['ProcessingResources']['ClusterConfig']['VolumeKmsKeyId'] = args['resource_encryption_key']
    else:
        request['ProcessingResources']['ClusterConfig'].pop('VolumeKmsKeyId')

    if args['max_run_time']:
        request['StoppingCondition']['MaxRuntimeInSeconds'] = args['max_run_time']
    else:
        request['StoppingCondition']['MaxRuntimeInSeconds'].pop('max_run_time')

    request['Environment'] = args['environment']

    ### Update tags
    for key, val in args['tags'].items():
        request['Tags'].append({'Key': key, 'Value': val})

    return request


def create_processing_job(client, args):
  """Create a SageMaker processing job."""
  request = create_processing_job_request(args)
  try:
      client.create_processing_job(**request)
      processing_job_name = request['ProcessingJobName']
      logging.info("Created Processing Job with name: " + processing_job_name)
      logging.info("CloudWatch logs: https://{}.console.aws.amazon.com/cloudwatch/home?region={}#logStream:group=/aws/sagemaker/ProcessingJobs;prefix={};streamFilter=typeLogStreamPrefix"
        .format(args['region'], args['region'], processing_job_name))
      return processing_job_name
  except ClientError as e:
      raise Exception(e.response['Error']['Message'])


def wait_for_processing_job(client, processing_job_name, poll_interval=30):
  while(True):
    response = client.describe_processing_job(ProcessingJobName=processing_job_name)
    status = response['ProcessingJobStatus']
    if status == 'Completed':
      logging.info("Processing job ended with status: " + status)
      break
    if status == 'Failed':
      message = response['FailureReason']
      logging.info('Processing failed with the following error: {}'.format(message))
      raise Exception('Processing job failed')
    logging.info("Processing job is still in status: " + status)
    time.sleep(poll_interval)

def get_processing_job_outputs(client, processing_job_name):
    """Map the S3 outputs of a processing job to a dictionary object."""
    response = client.describe_processing_job(ProcessingJobName=processing_job_name)
    outputs = {}
    for output in response['ProcessingOutputConfig']['Outputs']:
        outputs[output['OutputName']] = output['S3Output']['S3Uri']

    return outputs


def stop_processing_job(client, job_name):
    try:
        client.stop_processing_job(ProcessingJobName=job_name)
    except ClientError as e:
        raise Exception(e.response['Error']['Message'])


def id_generator(size=4, chars=string.ascii_uppercase + string.digits):
  return ''.join(random.choice(chars) for _ in range(size))

def yaml_or_json_str(str):
  if str == "" or str == None:
    return None
  try:
    return json.loads(str)
  except:
    return yaml.safe_load(str)

def str_to_bool(str):
    # This distutils function returns an integer representation of the boolean
    # rather than a True/False value. This simply hard casts it.
    return bool(strtobool(str))

def write_output(output_path, output_value, json_encode=False):
    """Write an output value to the associated path, dumping as a JSON object
    if specified.
    Arguments:
    - output_path: The file path of the output.
    - output_value: The output value to write to the file.
    - json_encode: True if the value should be encoded as a JSON object.
    """

    write_value = json.dumps(output_value) if json_encode else output_value 

    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    Path(output_path).write_text(write_value)
