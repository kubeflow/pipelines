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
import papermill as pm
import sys
import subprocess
import tarfile
import logging
import utils
import uuid

from datetime import datetime
from google.cloud import storage
from kfp import Client

# List of notebook samples' test names and corresponding file names.
NOTEBOOK_SAMPLES = {
    'kubeflow_pipeline_using_TFX_OSS_components': 'KubeFlow Pipeline Using TFX OSS Components.ipynb',
    'lightweight_component': 'Lightweight Python components - basics.ipynb',
    'dsl_static_type_checking': 'DSL Static Type Checking.ipynb'
}


PROJECT_NAME = 'ml-pipeline-test'
PAPERMILL_ERR_MSG = 'An Exception was encountered at'


#TODO(numerology): Add unit-test for classes.
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
    #(TODO: numerology) target_image_prefix seems to be only used for post-submit
    # check.
    self._target_image_prefix = target_image_prefix
    self._namespace = namespace
    self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
    self._sample_test_output = self._results_gcs_dir
    self._work_dir = self.BASE_DIR + '/samples/core/' + self._test_name

    self._run_test()

  def check_result(self):
    os.chdir(self.TEST_DIR)
    subprocess.call([
        'python3',
        'run_sample_test.py',
        '--input',
        input,
        '--result',
        self._sample_test_result,
        '--output',
        self._sample_test_output,
        '--testname',
        self._test_name,
        '--namespace',
        self._namespace
    ])
    print('Copy the test results to GCS %s/' % self._results_gcs_dir)
    storage_client = storage.Client()
    working_bucket = PROJECT_NAME

    src_bucket = storage_client.get_bucket(working_bucket)
    dest_bucket =src_bucket # Currently copy to the same bucket.
    src_bucket.copy_blob(
        self._sample_test_result,
        dest_bucket,
        self._results_gcs_dir + '/' + self._sample_test_result)

  def check_notebook_result(self):
    # Workaround because papermill does not directly return exit code.
    exit_code = 1 if PAPERMILL_ERR_MSG in \
                     open('%s.ipynb' % self._test_name).read() else 0
    if self._test_name == 'dsl_static_type_checking':
      subprocess.call([
          'python3',
          'check_notebook_results.py',
          '--testname',
          self._test_name,
          '--result',
          self._sample_test_result,
          '--exit-code',
          exit_code
      ])
    else:
      subprocess.call([
          'python3',
          'check_notebook_results.py',
          '--testname',
          self._test_name,
          '--result',
          self._sample_test_result,
          '--namespace',
          self._namespace,
          '--exit-code',
          exit_code
      ])

    print('Copy the test results to GCS %s/' % self._results_gcs_dir)
    storage_client = storage.Client()
    working_bucket = PROJECT_NAME

    src_bucket = storage_client.get_bucket(working_bucket)
    dest_bucket =src_bucket # Currently copy to the same bucket.
    src_bucket.copy_blob(
        self._sample_test_result,
        dest_bucket,
        self._results_gcs_dir + '/' + self._sample_test_result)

  def _run_test(self):
    if len(self._results_gcs_dir) == 0:
      return 1

    # variables needed for sample test logic.
    input = '%s/%s.yaml' % (self._work_dir, self._test_name)
    test_cases = [] # Currently, only capture run-time error, no result check.
    sample_test_name = self._test_name + ' Sample Test'

    os.chdir(self._work_dir)
    print('Run the sample tests...')

    # For presubmit check, do not do any image injection as for now.
    # Notebook samples need to be papermilled first.
    if self._test_name == 'kubeflow_pipeline_using_TFX_OSS_components':
      bucket_prefix = 'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/'
      pm.execute_notebook(
          input_path='KubeFlow Pipeline Using TFX OSS Components.ipynb',
          output_path='%s.ipynb' % self._test_name,
          prepare_only=False,
          parameters=dict(
              EXPERIMENT_NAME='%s-test' % self._test_name,
              OUTPUT_DIR=self._results_gcs_dir,
              PROJECT_NAME=PROJECT_NAME,
              BASE_IMAGE='%spusherbase:dev' % self._target_image_prefix,
              TARGET_IMAGE='%spusher:dev' % self._target_image_prefix,
              TARGET_IMAGE_TWO='%spusher_two:dev' % self._target_image_prefix,
              KFP_PACKAGE='tmp/kfp.tar.gz',
              DEPLOYER_MODEL='Notebook_tfx_taxi_%s' % uuid.uuid1(),
              TRAINING_DATA=bucket_prefix + 'train50.csv',
              EVAL_DATA=bucket_prefix + 'eval20.csv',
              HIDDEN_LAYER_SIZE=10,
              STEPS=50
          )
      )
      self.check_notebook_result()
    elif self._test_name == 'lightweight_component':
      pm.execute_notebook(
          input_path='Lightweight Python components - basics.ipynb',
          output_path='%s.ipynb' % self._test_name,
          prepare_only=False,
          parameters=dict(
              EXPERIMENT_NAME='%s-test' % self._test_name,
              PROJECT_NAME=PROJECT_NAME,
              KFP_PACKAGE='tmp/kfp.tar.gz',
          )
      )
      self.check_notebook_result()
    elif self._test_name == 'dsl_static_type_checking':
      pm.execute_notebook(
          input_path='DSL Static Type Checking.ipynb',
          output_path='%s.ipynb' % self._test_name,
          prepare_only=False,
          parameters=dict(
              KFP_PACKAGE='tmp/kfp.tar.gz',
          )
      )
      self.check_notebook_result()
    else:
      subprocess.call(['dsl-compile', '--py', '%s.py' % self._test_name,
                       '--output', '%s.yaml' % self._test_name])
      self.check_result()


class ComponentTest(SampleTest):
  """ Launch a KFP sample test as component test provided its name.

  Currently follows the same logic as sample test for compatibility.
  """
  def __init__(self, test_name, input, result, output,
      result_gcs_dir, target_image_prefix, dataflow_tft_image,
      namespace='kubeflow'):
    super().__init__(test_name, input, result, output, namespace)
    pass

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