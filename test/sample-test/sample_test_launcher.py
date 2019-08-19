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
"""
This launcher module serves as the entry-point of the sample test image. It
decides which test to trigger based upon the arguments provided.
"""

import argparse
import fire
import importlib
import io
import json
import os
import sys
import tarfile
import logging
import utils

from datetime import datetime
from kfp import Client


class SampleTest(object):
  """Launch a KFP sample_test provided its name.

  Args:
    test_name: name of the sample test.
    input: The path of a pipeline package that will be submitted.
    result: The path of the test result that will be exported.
    output: The path of the test output.
    namespace: Namespace of the deployed pipeline system. Default: kubeflow
  """

  GITHUB_REPO = 'kubeflow/pipelines'
  BASE_DIR= '/python/src/github.com/' + GITHUB_REPO
  TEST_DIR = BASE_DIR + '/test/sample-test'

  def __init__(self, test_name, test_results_gcs_dir, target_image_prefix,
                         namespace='kubeflow'):
    self._test_name = test_name
    self._results_gcs_dir = test_results_gcs_dir
    self._target_image_prefix = target_image_prefix
    self._namespace = namespace
    self._sample_test_result = 'junit_Sample' + self._test_name + 'Output.xml'
    self._sample_test_output = self._results_gcs_dir
    self._work_dir = self.BASE_DIR + '/samples/core/' + self._test_name

    self._run_test()

  def _run_test(self):
    if len(self._results_gcs_dir) == 0:
      return 1

    os.chdir(self._work_dir)





    # variables needed for sample test logic.
    input = '%s/%s.yaml' % (self._work_dir, self._test_name)
    test_cases = [] # Currently, only capture run-time error, no result check.
    sample_test_name = self._test_name + ' Sample Test'

    ###### Initialization ######
    host = 'ml-pipeline.%s.svc.cluster.local:8888' % self._namespace
    client = Client(host=host)

    ###### Check Input File ######
    utils.add_junit_test(test_cases, 'input generated yaml file',
                         os.path.exists(input), 'yaml file is not generated')
    if not os.path.exists(input):
      utils.write_junit_xml(sample_test_name, self._sample_test_result, test_cases)
      print('Error: job not found.')
      exit(1)

    ###### Create Experiment ######
    experiment_name = self._test_name + ' sample experiment'
    response = client.create_experiment(experiment_name)
    experiment_id = response.id
    utils.add_junit_test(test_cases, 'create experiment', True)

    ###### Create Job ######
    job_name = self._test_name + '_sample'
    ###### Test-specific arguments #######
    if self._test_name == 'tfx_cab_classification':
      params = {
          'output':
            self._sample_test_output,
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
    elif self._test_name == 'kubeflow_training_classification':
      params = {
          'output': self._sample_test_output,
          'project': 'ml-pipeline-test',
          'evaluation': 'gs://ml-pipeline-dataset/sample-test/flower/eval15.csv',
          'train': 'gs://ml-pipeline-dataset/sample-test/flower/train30.csv',
          'hidden-layer-size': '10,5',
          'steps': '5'
      }
    elif self._test_name == 'xgboost_training_cm':
      params = {
          'output': self._sample_test_output,
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

    response = client.run_pipeline(experiment_id, job_name, input, params)
    run_id = response.id
    utils.add_junit_test(test_cases, 'create pipeline run', True)

    ###### Monitor Job ######
    try:
      start_time = datetime.now()
      if self._test_name == 'xgboost_training_cm':
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
          self._namespace, workflow_id))
      print('=========Argo Workflow Log=========')
      print(argo_log)

    if not succ:
      utils.write_junit_xml(sample_test_name, self._sample_test_result, test_cases)
      exit(1)

    ###### Validate the results for specific test cases ######
    #TODO: Add result check for tfx-cab-classification after launch.
    if self._test_name == 'kubeflow_training_classification':
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
    elif self._test_name == 'xgboost_training_cm':
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
    utils.write_junit_xml(sample_test_name, self._sample_test_result, test_cases)

class ComponentTest(SampleTest):
  """ Launch a KFP sample test as component test provided its name.

  Currently follows the same logic as sample test for compatibility.
  """
  def __init__(self, test_name, input, result, output,
      result_gcs_dir, target_image_prefix, dataflow_tft_image,
      namespace='kubeflow'):
    super().__init__(test_name, input, result, output, namespace)

def main():
  """Launches either KFP sample test or component test as a command entrypoint.

  Usage:
  python sample_test_launcher.py sample_test arg1 arg2 to launch sample test, and
  python sample_test_launcher.py component_test arg1 arg2 to launch component
  test.
  """
  fire.Fire({
      'sample_test': SampleTest,
      'component_test': ComponentTest
  })

if __name__ == '__main__':
  main()