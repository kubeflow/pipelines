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

import fire
import os
import papermill as pm
import subprocess
import sys
import utils
import uuid


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

  def __init__(self, test_name, results_gcs_dir, target_image_prefix='',
                         namespace='kubeflow'):
    self._test_name = test_name
    self._results_gcs_dir = results_gcs_dir
    self._target_image_prefix = target_image_prefix
    self._namespace = namespace
    self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
    self._sample_test_output = self._results_gcs_dir
    self._work_dir = self.BASE_DIR + '/samples/core/' + self._test_name

  def check_result(self):
    os.chdir(self.TEST_DIR)
    subprocess.call([
        sys.executable,
        'run_sample_test.py',
        '--input',
        '%s/%s.yaml' % (self._work_dir, self._test_name),
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
    working_bucket = PROJECT_NAME

    utils.upload_blob(
        working_bucket,
        self._sample_test_result,
        self._results_gcs_dir + '/' + self._sample_test_result
    )

  def check_notebook_result(self):
    # Workaround because papermill does not directly return exit code.
    exit_code = '1' if PAPERMILL_ERR_MSG in \
                     open('%s.ipynb' % self._test_name).read() else '0'

    os.chdir(self.TEST_DIR)
    if self._test_name == 'dsl_static_type_checking':
      subprocess.call([
          sys.executable,
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
          sys.executable,
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
    working_bucket = PROJECT_NAME

    utils.upload_blob(
        working_bucket,
        self._sample_test_result,
        self._results_gcs_dir + '/' + self._sample_test_result
    )

  def _run_test(self):
    if len(self._results_gcs_dir) == 0:
      return 1

    os.chdir(self._work_dir)
    print('Run the sample tests...')

    # For presubmit check, do not do any image injection as for now.
    # Notebook samples need to be papermilled first.
    if self._test_name == 'lightweight_component':
      pm.execute_notebook(
          input_path='Lightweight Python components - basics.ipynb',
          output_path='%s.ipynb' % self._test_name,
          parameters=dict(
              EXPERIMENT_NAME='%s-test' % self._test_name
          )
      )
      self.check_notebook_result()
    elif self._test_name == 'dsl_static_type_checking':
      pm.execute_notebook(
          input_path='DSL Static Type Checking.ipynb',
          output_path='%s.ipynb' % self._test_name,
          parameters=dict(
              KFP_PACKAGE='tmp/kfp.tar.gz',
          )
      )
      self.check_notebook_result()
    else:
      subprocess.call(['dsl-compile', '--py', '%s.py' % self._test_name,
                       '--output', '%s.yaml' % self._test_name])

  def run_test(self):
    self._run_test()
    if self._test_name in ['lightweight_component', 'dsl_static_type_checking']:
      self.check_notebook_result()
    else:
      self.check_result()


class ComponentTest(SampleTest):
  """ Launch a KFP sample test as component test provided its name.

  Currently follows the same logic as sample test for compatibility.
  include xgboost_training_cm tfx_cab_classification
  """
  def __init__(self, test_name, results_gcs_dir,
      dataflow_tft_image,
      dataflow_predict_image,
      dataflow_tfma_image,
      dataflow_tfdv_image,
      dataproc_create_cluster_image,
      dataproc_delete_cluster_image,
      dataproc_analyze_image,
      dataproc_transform_image,
      dataproc_train_image,
      dataproc_predict_image,
      kubeflow_dnntrainer_image,
      kubeflow_deployer_image,
      local_confusionmatrix_image,
      local_roc_image,
      target_image_prefix='',
      namespace='kubeflow'):
    super().__init__(
        test_name=test_name,
        results_gcs_dir=results_gcs_dir,
        target_image_prefix=target_image_prefix,
        namespace=namespace
    )
    self._dataflow_tft_image = dataflow_tft_image
    self._dataflow_predict_image = dataflow_predict_image
    self._dataflow_tfma_image = dataflow_tfma_image
    self._dataflow_tfdv_image = dataflow_tfdv_image
    self._dataproc_create_cluster_image = dataproc_create_cluster_image
    self._dataproc_delete_cluster_image = dataproc_delete_cluster_image
    self._dataproc_analyze_image = dataproc_analyze_image
    self._dataproc_transform_image = dataproc_transform_image
    self._dataproc_train_image = dataproc_train_image
    self._dataproc_predict_image = dataproc_predict_image
    self._kubeflow_dnntrainer_image = kubeflow_dnntrainer_image
    self._kubeflow_deployer_image = kubeflow_deployer_image
    self._local_confusionmatrix_image = local_confusionmatrix_image
    self._local_roc_image = local_roc_image

  def _injection(self):
    """Sample-specific image injection into yaml file."""
    pass

  def run_test(self):
    # compile, injection, check_result
    self._run_test()
    self._injection()
    self.check_result()




def main():
  """Launches either KFP sample test or component test as a command entrypoint.

  Usage:
  python sample_test_launcher.py sample_test run_test arg1 arg2 to launch sample test, and
  python sample_test_launcher.py component_test run_test arg1 arg2 to launch component
  test.
  """
  fire.Fire({
      'sample_test': SampleTest,
      'component_test': ComponentTest
  })

if __name__ == '__main__':
  main()