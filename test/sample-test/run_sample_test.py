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

import os
import tarfile
import utils
from datetime import datetime
from kfp import Client


_TEST_TIMEOUT = 1200
_XGB_TEST_TIMEOUT = 1800 # xgboost needs additional time to finish.


class PySampleChecker(object):
  def __init__(self, testname, input, output, result, namespace='kubeflow'):
    """Util class for checking python sample test running results.

    :param testname: test name.
    :param input: The path of a pipeline file that will be submitted.
    :param output: The path of the test output.
    :param result: The path of the test result that will be exported.
    :param namespace: namespace of the deployed pipeline system. Default: kubeflow
    """
    self._testname = testname
    self._input = input
    self._output = output
    self._result = result
    self._namespace = namespace

  def check(self):
    """Run sample test and check results."""
    test_cases = []
    test_name = self._testname + ' Sample Test'

    ###### Initialization ######
    host = 'ml-pipeline.%s.svc.cluster.local:8888' % self._namespace
    client = Client(host=host)

    ###### Check Input File ######
    utils.add_junit_test(test_cases, 'input generated yaml file',
                         os.path.exists(self._input), 'yaml file is not generated')
    if not os.path.exists(self._input):
      utils.write_junit_xml(test_name, self._result, test_cases)
      print('Error: job not found.')
      exit(1)

    ###### Create Experiment ######
    experiment_name = self._testname + ' sample experiment'
    response = client.create_experiment(experiment_name)
    experiment_id = response.id
    utils.add_junit_test(test_cases, 'create experiment', True)

    ###### Create Job ######
    job_name = self._testname + '_sample'
    ###### Test-specific arguments #######
    if self._testname == 'tfx_cab_classification':
      params = {
        'output':
          self._output,
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
    elif self._testname == 'xgboost_training_cm':
      params = {
        'output': self._output,
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

    response = client.run_pipeline(experiment_id, job_name, self._input, params)
    run_id = response.id
    utils.add_junit_test(test_cases, 'create pipeline run', True)

    ###### Monitor Job ######
    try:
      start_time = datetime.now()
      if self._testname == 'xgboost_training_cm':
        response = client.wait_for_run_completion(run_id, _XGB_TEST_TIMEOUT)
      else:
        response = client.wait_for_run_completion(run_id, _TEST_TIMEOUT)
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
      utils.write_junit_xml(test_name, self._result, test_cases)
      exit(1)

    ###### Validate the results for specific test cases ######
    #TODO: Add result check for tfx-cab-classification after launch.
    if self._testname == 'xgboost_training_cm':
      # For xgboost sample, check its confusion matrix.
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
    utils.write_junit_xml(test_name, self._result, test_cases)
