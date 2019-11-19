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
  test_name = 'Resnet CMLE Test'

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
  experiment_name = 'resnet cmle sample experiment'
  response = client.create_experiment(experiment_name)
  experiment_id = response.id
  utils.add_junit_test(test_cases, 'create experiment', True)

  ###### Create Job ######
  job_name = 'cmle_sample'
  params = {'output': args.output,
            'project_id': 'ml-pipeline-test',
            'region': 'us-central1',
            'model': 'bolts',
            'version': 'beta1',
            'tf_version': '1.9', # Watch out! If 1.9 is no longer supported we need to set it to a newer version.
            'train_csv': 'gs://ml-pipeline-dataset/sample-test/bolts/bolt_images_train_sample1000.csv',
            'validation_csv': 'gs://ml-pipeline-dataset/sample-test/bolts/bolt_images_validate_sample200.csv',
            'labels': 'gs://bolts_image_dataset/labels.txt',
            'depth': 50,
            'train_batch_size': 32,
            'eval_batch_size': 32,
            'steps_per_eval': 128,
            'train_steps': 128,
            'num_train_images': 1000,
            'num_eval_images': 200,
            'num_label_classes': 10}
  response = client.run_pipeline(experiment_id, job_name, args.input, params)
  run_id = response.id
  utils.add_junit_test(test_cases, 'create pipeline run', True)

  ###### Monitor Job ######
  try:
    start_time = datetime.now()
    response = client.wait_for_run_completion(run_id, 1800)
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

  ###### Delete Job ######
  #TODO: add deletion when the backend API offers the interface.

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)

if __name__ == "__main__":
  main()
