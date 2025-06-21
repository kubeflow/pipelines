# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import os
import re
import subprocess

import boto3
from botocore.exceptions import ClientError
from junit_xml import TestSuite, TestCase

# Parse the workflow json to obtain the artifacts for a particular step.
#   Note: the step_name could be the key words.
def get_artifact_in_minio(workflow_json, step_name, output_path, artifact_name='mlpipeline-ui-metadata'):
  s3_data = {}
  s3_access_key = 'minio'
  s3_secret_key = 'minio123'
  try:
    for node in workflow_json['status']['nodes'].values():
      if step_name in node['name']:
        for artifact in node['outputs']['artifacts']:
          if artifact['name'] == artifact_name:
            s3_data = artifact['s3']
    
    # Create boto3 S3 client for SeaweedFS
    s3_client = boto3.client(
        's3',
        aws_access_key_id=s3_access_key,
        aws_secret_access_key=s3_secret_key,
        endpoint_url=f"http://{s3_data['endpoint']}",
        region_name='us-east-1'  # Required but not used by SeaweedFS
    )
    
    # Download the object from SeaweedFS
    response = s3_client.get_object(Bucket=s3_data['bucket'], Key=s3_data['key'])
    
    with open(output_path, 'wb') as file:
      # Read and write the data in chunks
      while True:
        chunk = response['Body'].read(32*1024)
        if not chunk:
          break
        file.write(chunk)
        
  except ClientError as e:
    error_code = e.response['Error']['Code']
    print(f'SeaweedFS S3 error in get_artifact_in_minio: {error_code} - {e}')
    print(workflow_json)
  except Exception as e:
    print('error in get_artifact_in_minio: %s', e)
    print(workflow_json)

# Junit xml utilities
def add_junit_test(test_cases, testname, succ, message='default message', elapsed_sec=0):
  test_case = TestCase(testname, elapsed_sec = elapsed_sec)
  if not succ:
    test_case.add_failure_info(message)
  test_cases.append(test_case)

def write_junit_xml(testname, filename, test_cases):
  with open(filename, 'w') as f:
    ts = TestSuite(testname, test_cases)
    TestSuite.to_file(f, [ts], prettyprint=False)

# Bash utilities
def run_bash_command(cmd):
  process = subprocess.Popen(cmd.split(), stdout=subprocess.PIPE)
  output_bytes, error_bytes = process.communicate()
  output_string = ''
  error_string = ''
  if output_bytes != None:
    output_string = output_bytes.decode('utf-8')
  if error_bytes != None:
    error_string = error_bytes.decode('utf-8')
  return output_string, error_string


def file_injection(file_in, tmp_file_out, subs):
  """Utility function that substitute several regex within a file by
  corresponding string.

  :param file_in: input file name.
  :param tmp_file_out: tmp output file name.
  :param subs: dict, key is the regex expr, value is the substituting string.
  """
  with open(file_in, 'rt') as fin:
    with open(tmp_file_out, 'wt') as fout:
      for line in fin:
        tmp_line = line
        for old, new in subs.items():
          regex = re.compile(old)
          tmp_line = re.sub(regex, new, line)
          line = tmp_line

        fout.write(tmp_line)

  os.rename(tmp_file_out, file_in)
