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
from datetime import datetime
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
  parser.add_argument('--namespace',
                      type=str,
                      default='kubeflow',
                      help="namespace of the deployed pipeline system. Default: kubeflow")
  args = parser.parse_args()
  return args

def main():
  args = parse_arguments()
  test_cases = []
  test_name = 'Kubeflow Sample Test'

  ###### Initialization ######
  host = 'ml-pipeline.%s.svc.cluster.local:8888' % args.namespace
  client = Client(host=host)

  ###### Check Input File ######
  utils.add_junit_test(test_cases, 'input generated yaml file', os.path.exists(args.input), 'yaml file is not generated')
  if not os.path.exists(args.input):
    utils.write_junit_xml(test_name, args.result, test_cases)
    print('Error: job not found.')
    exit(1)

  ###### Create Experiment ######
  experiment_name = 'kubeflow sample experiment'
  response = client.create_experiment(experiment_name)
  experiment_id = response.id
  utils.add_junit_test(test_cases, 'create experiment', True)

  ###### Create Job ######
  job_name = 'kubeflow_sample'
  params = {'output': args.output,
            'project': 'ml-pipeline-test',
            'evaluation': 'gs://ml-pipeline-dataset/sample-test/flower/eval15.csv',
            'train': 'gs://ml-pipeline-dataset/sample-test/flower/train30.csv',
            'hidden-layer-size': '10,5',
            'steps': '5'}
  response = client.run_pipeline(experiment_id, job_name, args.input, params)
  run_id = response.id
  utils.add_junit_test(test_cases, 'create pipeline run', True)

  ###### Monitor Job ######
  try:
    start_time = datetime.now()
    response = client.wait_for_run_completion(run_id, 1200)
    succ = (response.run.status.lower()=='succeeded')
    end_time = datetime.now()
    elapsed_time = (end_time - start_time).seconds
    utils.add_junit_test(test_cases, 'job completion', succ, 'waiting for job completion failure', elapsed_time)
  finally:
    ###### Output Argo Log for Debugging ######
    workflow_json = client._get_workflow_json(run_id)
    workflow_id = workflow_json['metadata']['name']
    argo_log, _ = utils.run_bash_command('argo logs -n {} -w {}'.format(args.namespace, workflow_id))
    print("=========Argo Workflow Log=========")
    print(argo_log)

  if not succ:
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit(1)

  ###### Validate the results ######
  #   confusion matrix should show three columns for the flower data
  #     target, predicted, count
  cm_tar_path = './confusion_matrix.tar.gz'
  cm_filename = 'mlpipeline-ui-metadata.json'
  utils.get_artifact_in_minio(workflow_json, 'confusion-matrix', cm_tar_path)
  tar_handler = tarfile.open(cm_tar_path)
  tar_handler.extractall()

  with open(cm_filename, 'r') as f:
    cm_data = json.load(f)
    utils.add_junit_test(test_cases, 'confusion matrix format', (len(cm_data['outputs'][0]['schema']) == 3), 'the column number of the confusion matrix output is not equal to three')

  ###### Delete Job ######
  #TODO: add deletion when the backend API offers the interface.

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)

if __name__ == "__main__":
  main()
