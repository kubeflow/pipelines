# Copyright 2018 Google LLC
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
# limitations under the License.

import argparse
import os
import json
import tarfile
import utils
from kfp import Client

###### Input/Output Instruction ######
# input: yaml
# output: local file path


# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--input',
                      type=str,
                      required=True,
                      help='The path of a pipeline package that will be submitted.')
  parser.add_argument('--result',
                      type=str,
                      required=True,
                      help='The path of the test result that will be exported.')
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='The path of the test output')
  args = parser.parse_args()
  return args

def main():
  args = parse_arguments()
  test_cases = []
  test_name = 'Kubeflow Sample Test'

  ###### Initialization ######
  client = Client()

  ###### Check Input File ######
  utils.add_junit_test(test_cases, 'input generated yaml file', os.path.exists(args.input), 'yaml file is not generated')
  if not os.path.exists(args.input):
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit()

  ###### Upload Pipeline ######
  response = client.upload_pipeline(args.input)
  pipeline_id = response.id
  utils.add_junit_test(test_cases, 'upload pipeline yaml', True)

  ###### Create Job ######
  job_name = 'kubeflow_sample'
  params = {'output': args.output,
            'project': 'ml-pipeline-test',
            'evaluation': 'gs://ml-pipeline-playground/IntegTest/flower/eval15.csv',
            'train': 'gs://ml-pipeline-playground/IntegTest/flower/train30.csv',
            'hidden-layer-size': '10,5',
            'steps': '5'}
  response = client.run_pipeline(job_name, pipeline_id, params)
  job_id = response.id
  utils.add_junit_test(test_cases, 'create pipeline job', True)

  ###### Monitor Job ######
  succ, elapsed_time = client._wait_for_job_completion(job_id, 1200)
  utils.add_junit_test(test_cases, 'job completion', succ, 'waiting for job completion failure', elapsed_time)
  if not succ:
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit()

  ###### Output Argo Log for Debugging ######
  workflow_json = client._get_workflow_json(job_id, 0)
  workflow_id = workflow_json['metadata']['name']
  argo_log, _ = utils.run_bash_command('argo logs -w {}'.format(workflow_id))
  print("=========Argo Workflow Log=========")
  print(argo_log)

  ###### Validate the results ######
  #   confusion matrix should show three columns for the flower data
  #     target, predicted, count
  cm_tar_path = './confusion_matrix.tar.gz'
  cm_filename = 'mlpipeline-ui-metadata.json'
  utils.get_artifact_in_minio(workflow_json, 'confusionmatrix', cm_tar_path)
  tar_handler = tarfile.open(cm_tar_path)
  tar_handler.extractall()

  with open(cm_filename, 'r') as f:
    cm_data = json.load(f)
    utils.add_junit_test(test_cases, 'confusion matrix format', (len(cm_data['outputs'][0]['schema']) == 3), 'the column number of the confusion matrix output is not equal to three')

  ###### Delete Job ######
  client._delete_job(job_id)
  utils.add_junit_test(test_cases, 'delete pipeline job', True)

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)

if __name__ == "__main__":
  main()
