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
import time

import boto3
from botocore.exceptions import ClientError
import json


def get_client(region=None):
    """Builds a client to the AWS EMR API."""
    client = boto3.client('emr', region_name=region)
    return client

def create_cluster(client, cluster_name, log_s3_uri, release_label, instance_type, instance_count, ec2SubnetId=None, ec2KeyName=None):
  """Create a EMR cluster."""

  instances = {
      'MasterInstanceType': instance_type,
      'SlaveInstanceType': instance_type,
      'InstanceCount': instance_count,
      'KeepJobFlowAliveWhenNoSteps':True,
      'TerminationProtected':False
  }

  if ec2SubnetId is not None:
    instances['Ec2SubnetId'] = ec2SubnetId

  if ec2KeyName is not None:
    instances['Ec2KeyName'] = ec2KeyName

  response = client.run_job_flow(
    Name=cluster_name,
    LogUri=log_s3_uri,
    ReleaseLabel=release_label,
    Applications=[
      {
        'Name': 'Spark'
      }
    ],
    BootstrapActions=[
      {
        'Name': 'Maximize Spark Default Config',
        'ScriptBootstrapAction': {
          'Path': 's3://support.elasticmapreduce/spark/maximize-spark-default-config',
        }
      },
    ],
    Instances= instances,
    VisibleToAllUsers=True,
    JobFlowRole='EMR_EC2_DefaultRole',
    ServiceRole='EMR_DefaultRole'
  )
  return response


def delete_cluster(client, jobflow_id):
  """Delete a EMR cluster. Cluster shutdowns in background"""
  client.terminate_job_flows(JobFlowIds=[jobflow_id])

def wait_for_cluster(client, jobflow_id):
  """Waiting for a new cluster to be ready."""
  while True:
    response = client.describe_cluster(ClusterId=jobflow_id)
    cluster_status = response['Cluster']['Status']
    state = cluster_status['State']

    if 'Message' in cluster_status['StateChangeReason']:
      state = cluster_status['State']
      message = cluster_status['StateChangeReason']['Message']

      if state in ['TERMINATED', 'TERMINATED', 'TERMINATED_WITH_ERRORS']:
        raise Exception(message)

      if state == 'WAITING':
        print('EMR cluster create completed')
        break

    print("Cluster state: {}, wait 15s for cluster to start up.".format(state))
    time.sleep(15)

# Check following documentation to add other job type steps. Seems python SDK only have 'HadoopJarStep' here.
# https://docs.aws.amazon.com/cli/latest/reference/emr/add-steps.html
def submit_spark_job(client, jobflow_id, job_name, jar_path, main_class, extra_args):
  """Submits single spark job to a running cluster"""

  spark_job = {
    'Name': job_name,
    'ActionOnFailure': 'CONTINUE',
    'HadoopJarStep': {
      'Jar': 'command-runner.jar'
    }
  }

  spark_args = ['spark-submit', "--deploy-mode", "cluster"]
  if main_class:
    spark_args.extend(['--class', main_class])
  spark_args.extend([jar_path])
  spark_args.extend(extra_args)

  spark_job['HadoopJarStep']['Args'] = spark_args

  try:
    response = client.add_job_flow_steps(
            JobFlowId=jobflow_id,
            Steps=[spark_job],
    )
  except ClientError as e:
    print(e.response['Error']['Message'])
    exit(1)

  step_id = response['StepIds'][0]
  print("Step Id {} has been submitted".format(step_id))
  return step_id

def wait_for_job(client, jobflow_id, step_id):
  """Waiting for a cluster step by polling it."""
  while True:
    result = client.describe_step(ClusterId=jobflow_id, StepId=step_id)
    step_status = result['Step']['Status']
    state = step_status['State']

    if state in ('CANCELLED', 'FAILED', 'INTERRUPTED'):
      err_msg = 'UNKNOWN'
      if 'FailureDetails' in step_status:
        err_msg = step_status['FailureDetails']

      raise Exception(err_msg)
    elif state == 'COMPLETED':
      print('EMR Step finishes')
      break

    print("Step state: {}, wait 15s for step status update.".format(state))
    time.sleep(10)

def submit_pyspark_job(client, jobflow_id, job_name, py_file, extra_args):
  """Submits single spark job to a running cluster"""
  return submit_spark_job(client, jobflow_id, job_name, py_file, '', extra_args)
