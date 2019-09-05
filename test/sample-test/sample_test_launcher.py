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
import utils

from constants import PAPERMILL_ERR_MSG, BASE_DIR, TEST_DIR
from check_notebook_results import NoteBookChecker
from run_sample_test import PySampleChecker


class SampleTest(object):

  def __init__(self, test_name, results_gcs_dir, target_image_prefix='',
               namespace='kubeflow'):
    """Launch a KFP sample_test provided its name.

    :param test_name: name of the corresponding sample test.
    :param results_gcs_dir: gs dir to store test result.
    :param target_image_prefix: prefix of docker image, default is empty.
    :param namespace: namespace for kfp, default is kubeflow.
    """
    self._test_name = test_name
    self._results_gcs_dir = results_gcs_dir
    # Capture the first segment after gs:// as the project name.
    self._bucket_name = results_gcs_dir.split('/')[2]
    self._target_image_prefix = target_image_prefix
    self._namespace = namespace
    self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
    self._sample_test_output = self._results_gcs_dir
    self._work_dir = os.path.join(BASE_DIR, 'samples/core/', self._test_name)

  def check_result(self):
    os.chdir(TEST_DIR)
    pysample_checker = PySampleChecker(testname=self._test_name,
                                       input=os.path.join(self._work_dir, '%s.yaml' % self._test_name),
                                       output=self._sample_test_output,
                                       result=self._sample_test_result,
                                       namespace=self._namespace)
    pysample_checker.check()

    print('Copy the test results to GCS %s/' % self._results_gcs_dir)
    utils.upload_blob(
      self._bucket_name,
      self._sample_test_result,
      os.path.join(self._results_gcs_dir, self._sample_test_result)
    )

  def check_notebook_result(self):
    # Workaround because papermill does not directly return exit code.
    exit_code = '1' if PAPERMILL_ERR_MSG in \
                       open('%s.ipynb' % self._test_name).read() else '0'

    os.chdir(TEST_DIR)

    if self._test_name == 'dsl_static_type_checking':
        nbchecker = NoteBookChecker(testname=self._test_name,
                                    result=self._sample_test_result,
                                    exit_code=exit_code)
        nbchecker.check()
    else:
        nbchecker = NoteBookChecker(testname=self._test_name,
                                    result=self._sample_test_result,
                                    exit_code=exit_code,
                                    experiment=None,
                                    namespace='kubeflow')
        nbchecker.check()

    print('Copy the test results to GCS %s/' % self._results_gcs_dir)

    utils.upload_blob(
        self._bucket_name,
        self._sample_test_result,
        os.path.join(self._results_gcs_dir, self._sample_test_result)
    )

  def _compile_sample(self):

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
    elif self._test_name == 'dsl_static_type_checking':
      pm.execute_notebook(
          input_path='DSL Static Type Checking.ipynb',
          output_path='%s.ipynb' % self._test_name,
          parameters={}
      )
    else:
      subprocess.call(['dsl-compile', '--py', '%s.py' % self._test_name,
                       '--output', '%s.yaml' % self._test_name])

  def run_test(self):
    self._compile_sample()
    if self._test_name in ['lightweight_component', 'dsl_static_type_checking']:
      self.check_notebook_result()
    else:
      self.check_result()


class ComponentTest(SampleTest):
  """ Launch a KFP sample test as component test provided its name.

  Currently follows the same logic as sample test for compatibility.
  include xgboost_training_cm, tfx_cab_classification
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
    subs = {
        'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-confusion-matrix:\w+':self._local_confusionmatrix_image,
        'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-roc:\w+':self._local_roc_image
    }
    if self._test_name == 'xgboost_training_cm':
      subs.update({
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:\w+':self._dataproc_create_cluster_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:\w+':self._dataproc_delete_cluster_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-analyze:\w+':self._dataproc_analyze_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-transform:\w+':self._dataproc_transform_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-train:\w+':self._dataproc_train_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataproc-predict:\w+':self._dataproc_predict_image,
      })

      utils.file_injection('%s.yaml' % self._test_name,
                           '%s.yaml.tmp' % self._test_name,
                           subs)
    elif self._test_name == 'tfx_cab_classification':
      subs.update({
          'gcr\.io/ml-pipeline/ml-pipeline-dataflow-tft:\w+':self._dataflow_tft_image,
          'gcr\.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:\w+':self._dataflow_predict_image,
          'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-dataflow-tfdv:\w+':self._dataflow_tfdv_image,
          'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-dataflow-tfma:\w+':self._dataflow_tfma_image,
          'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:\w+':self._kubeflow_dnntrainer_image,
          'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-kubeflow-deployer:\w+':self._kubeflow_deployer_image,
      })
    else:
      # Only the above two samples need injection for now.
      pass
    utils.file_injection('%s.yaml' % self._test_name,
                         '%s.yaml.tmp' % self._test_name,
                         subs)


  def run_test(self):
    # compile, injection, check_result
    self._compile_sample()
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