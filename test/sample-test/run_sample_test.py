# Copyright 2019 Google LLC
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
import io
import json
import tarfile
import utils
from datetime import datetime
from kfp import Client

###### Input/Output Instruction ######
# input: yaml
# output: local file path


# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--input',
      type=str,
      required=True,
      help='The path of a pipeline package that will be submitted.')
  parser.add_argument(
      '--result',
      type=str,
      required=True,
      help='The path of the test result that will be exported.')
  parser.add_argument(
      '--output', type=str, required=True, help='The path of the test output')
  parser.add_argument('--testname', type=str, required=True, help='Test name')
  parser.add_argument(
      '--namespace',
      type=str,
      default='kubeflow',
      help='namespace of the deployed pipeline system. Default: kubeflow')
  args = parser.parse_args()
  return args


def main():
  args = parse_arguments()
  test_cases = []
  test_name = args.testname + ' Sample Test'

  ###### Initialization ######
  host = 'ml-pipeline.%s.svc.cluster.local:8888' % args.namespace
  client = Client(host=host)

  ###### Check Input File ######
  utils.add_junit_test(test_cases, 'input generated yaml file',
                       os.path.exists(args.input), 'yaml file is not generated')
  if not os.path.exists(args.input):
    utils.write_junit_xml(test_name, args.result, test_cases)
    print('Error: job not found.')
    exit(1)

  ###### Create Experiment ######
  experiment_name = args.testname + ' sample experiment'
  response = client.create_experiment(experiment_name)
  experiment_id = response.id
  utils.add_junit_test(test_cases, 'create experiment', True)

  ###### Create Job ######
  job_name = args.testname + '_sample'
  ###### Test-specific arguments #######
  if args.testname == 'tfx_cab_classification':
    params = {
        'output':
          args.output,
        'project':
          'ml-pipeline-test',
        'column-names':
          'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/column-names.json',
        'evaluation':
          'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv',
        'train':
          'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv',
        'hidden-layer-size':
          '5',
        'steps':
          '5'
    }
  elif args.testname == 'kubeflow_training_classification':
    params = {
        'output': args.output,
        'project': 'ml-pipeline-test',
        'evaluation': 'gs://ml-pipeline-dataset/sample-test/flower/eval15.csv',
        'train': 'gs://ml-pipeline-dataset/sample-test/flower/train30.csv',
        'hidden-layer-size': '10,5',
        'steps': '5'
    }
  elif args.testname == 'xgboost_training_cm':
    params = {
        'output': args.output,
        'project': 'ml-pipeline-test',
        'train-data': 'gs://ml-pipeline-dataset/sample-test/sfpd/train_50.csv',
        'eval-data': 'gs://ml-pipeline-dataset/sample-test/sfpd/eval_20.csv',
        'schema': 'gs://ml-pipeline-dataset/sample-test/sfpd/schema.json',
        'rounds': '20',
        'workers': '2'
    }
  else:
    # Basic tests require no additional params.
    params = {}

  response = client.run_pipeline(experiment_id, job_name, args.input, params)
  run_id = response.id
  utils.add_junit_test(test_cases, 'create pipeline run', True)

  ###### Monitor Job ######
  try:
    start_time = datetime.now()
    if args.testname == 'xgboost_training_cm':
      response = client.wait_for_run_completion(run_id, 1800)
    else:
      response = client.wait_for_run_completion(run_id, 1200)
    succ = (response.run.status.lower() == 'succeeded')
    end_time = datetime.now()
    elapsed_time = (end_time - start_time).seconds
    utils.add_junit_test(test_cases, 'job completion', succ,
                         'waiting for job completion failure', elapsed_time)
  finally:
    ###### Output Argo Log for Debugging ######
    workflow_json = client._get_workflow_json(run_id)
    workflow_id = workflow_json['metadata']['name']
    argo_log, _ = utils.run_bash_command('argo logs -n {} -w {}'.format(
        args.namespace, workflow_id))
    print('=========Argo Workflow Log=========')
    print(argo_log)

  if not succ:
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit(1)

  ###### Validate the results for specific test cases ######
  #TODO: Add result check for tfx-cab-classification after launch.
  if args.testname == 'kubeflow_training_classification':
    cm_tar_path = './confusion_matrix.tar.gz'
    utils.get_artifact_in_minio(workflow_json, 'confusion-matrix', cm_tar_path,
                                'mlpipeline-ui-metadata')
    with tarfile.open(cm_tar_path) as tar_handle:
      file_handles = tar_handle.getmembers()
      assert len(file_handles) == 1

      with tar_handle.extractfile(file_handles[0]) as f:
        cm_data = json.load(io.TextIOWrapper(f))
        utils.add_junit_test(
            test_cases, 'confusion matrix format',
            (len(cm_data['outputs'][0]['schema']) == 3),
            'the column number of the confusion matrix output is not equal to three'
        )
  elif args.testname == 'xgboost_training_cm':
    cm_tar_path = './confusion_matrix.tar.gz'
    utils.get_artifact_in_minio(workflow_json, 'confusion-matrix', cm_tar_path,
                                'mlpipeline-ui-metadata')
    with tarfile.open(cm_tar_path) as tar_handle:
      file_handles = tar_handle.getmembers()
      assert len(file_handles) == 1

      with tar_handle.extractfile(file_handles[0]) as f:
        cm_data = f.read()
        utils.add_junit_test(test_cases, 'confusion matrix format',
                             (len(cm_data) > 0),
                             'the confusion matrix file is empty')

  ###### Delete Job ######
  #TODO: add deletion when the backend API offers the interface.

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)


if __name__ == '__main__':
  main()
