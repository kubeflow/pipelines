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
import re
import subprocess
import utils
import yamale
import yaml

from constants import PAPERMILL_ERR_MSG, BASE_DIR, TEST_DIR, SCHEMA_CONFIG, CONFIG_DIR, DEFAULT_CONFIG
from check_notebook_results import NoteBookChecker
from kfp.containers._gcs_helper import GCSHelper
from run_sample_test import PySampleChecker


class SampleTest(object):

  def __init__(self, test_path, results_gcs_dir, target_image_prefix='',
               namespace='kubeflow'):
    """Launch a KFP sample_test provided its name.

    :param test_path: Path of the sample code file that should be tested. Can be python file, notebook or test config YAML.
    :param results_gcs_dir: gs dir to store test result.
    :param target_image_prefix: prefix of docker image, default is empty.
    :param namespace: namespace for kfp, default is kubeflow.
    """

    test_file_name = os.path.basename(test_path)
    test_name, ext1 = os.path.splitext(test_file_name)
    if ext1 == '.yaml':
      test_name, ext2 = os.path.splitext(test_name)

    self._test_path = test_path
    self._config_dict = {}
    self._arguments = {}
    self._test_name = test_name
    self._results_gcs_dir = results_gcs_dir
    # Capture the first segment after gs:// as the project name.
    self._bucket_name = results_gcs_dir.split('/')[2]
    self._target_image_prefix = target_image_prefix
    self._namespace = namespace

    # TODO(numerology): special treatment for new TFX::OSS sample. Current decision
    # is that we directly run its compiled version, for its compilation brings
    # complex and unstable dependencies. See
    if test_name == 'parameterized_tfx_oss':
      self._is_notebook = False
      self._work_dir = os.path.join(BASE_DIR, 'samples/contrib/', self._test_name)
    else:
      self._is_notebook = None
      self._work_dir = os.path.join(BASE_DIR, 'samples/core/', self._test_name)

    self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
    self._sample_test_output = self._results_gcs_dir

  def _copy_result(self):
    """ Copy generated sample test result to gcs, so that Prow can pick it. """
    print('Copy the test results to GCS %s/' % self._results_gcs_dir)

    GCSHelper.upload_gcs_file(
        self._sample_test_result,
        os.path.join(self._results_gcs_dir, self._sample_test_result))

  def _compile(self):
    config_schema = yamale.make_schema(SCHEMA_CONFIG)
    # Retrieve default config
    try:
      with open(DEFAULT_CONFIG, 'r') as f:
        self._config_dict = yaml.safe_load(f)
      default_config = yamale.make_data(DEFAULT_CONFIG)
      yamale.validate(config_schema, default_config)  # If fails, a ValueError will be raised.
    except yaml.YAMLError as yamlerr:
      raise RuntimeError('Illegal default config:{}'.format(yamlerr))
    except OSError as ose:
      raise FileExistsError('Default config not found:{}'.format(ose))

    test_path = os.path.join(BASE_DIR, self._test_path)
    if not os.path.exists(test_path):
      raise RuntimeError('Test file "{}" not found.'.format(test_path))

    test_file_name = os.path.basename(test_path)
    test_name, ext1 = os.path.splitext(test_file_name)
    if ext1 == '.yaml':
      test_name, ext2 = os.path.splitext(test_name)
    elif ext1 in ['.py', '.ipynb']:
      pass
    else:
      raise ValueError('Test file "{}" has unknown file extension.'.format(test_path))
    self._test_name = test_name
    
    real_test_path = test_path
    if ext1 == '.yaml' and ext2 == '.config':
      config_path = os.path.join(BASE_DIR, self._test_path)
      with open(config_path, 'r') as f:
        config_dict = yaml.safe_load(f)
      test_config = yamale.make_data(config_path)
      yamale.validate(config_schema, test_config)  # If fails, a ValueError will be raised.

      self._config_dict.update(config_dict) # Does not do deep merging
      real_test_path = config_dict['test_path']
      real_test_path = os.path.join(BASE_DIR, real_test_path)

      test_name2, ext1 = os.path.splitext(os.path.basename(real_test_path)) # When test_path point to config, the test name is taken from config file name, not the file it points to

      if ext1 not in ['.py', '.ipynb']:
        raise ValueError('Test file "{}" specified in the config has unknown file extension.'.format(test_path))
    
    self._arguments = self._config_dict.get('arguments', {})
    if 'output' in self._arguments:  # output is a special param that has to be specified dynamically.
      self._arguments['output'] = self._sample_test_output

    self._real_test_path = real_test_path
    self._is_notebook = ext1 == '.ipynb'

    self._run_pipeline = self._config_dict['run_pipeline']

    # For presubmit check, do not do any image injection as for now.
    # Notebook samples need to be papermilled first.
    if self._is_notebook:
      # Parse necessary params from config.yaml
      nb_params = self._arguments.copy()

      pm.execute_notebook(
          input_path='%s.ipynb' % self._test_name,
          output_path='%s.ipynb' % self._test_name,
          parameters=nb_params,
          prepare_only=True
      )
      # Convert to python script.
      subprocess.run([
          'jupyter', 'nbconvert', '--to', 'python', '%s.ipynb' % self._test_name
      ], check=True)

    else:
      subprocess.run(['dsl-compile', '--py', self._real_test_path,
                       '--output', '%s.yaml' % self._test_name], check=True)

  def _injection(self):
    """Inject images for pipeline components.
    This is only valid for coimponent test
    """
    pass

  def run_test(self):
    # TODO(numerology): ad hoc logic for TFX::OSS sample
    if self._test_name != 'parameterized_tfx_oss':
      self._compile()
      self._injection()

    # Overriding the experiment name of pipeline runs
    experiment_name = self._test_name + '-test'
    os.environ['KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'] = experiment_name

    if self._is_notebook:
      nbchecker = NoteBookChecker(testname=self._test_name,
                                  test_path='%s.py' % self._test_name,
                                  config_dict=self._config_dict,
                                  result=self._sample_test_result,
                                  run_pipeline=self._run_pipeline,
                                  experiment_name=experiment_name,
      )
      nbchecker.run()
      os.chdir(TEST_DIR)
      nbchecker.check()
    else:
      os.chdir(TEST_DIR)
      if self._test_name != 'parameterized_tfx_oss':
        input_file = os.path.join(self._work_dir, '%s.yaml' % self._test_name)
      else:
        input_file = os.path.join(self._work_dir, '%s.tar.gz' % self._test_name)

      pysample_checker = PySampleChecker(testname=self._test_name,
                                         test_path=self._real_test_path,
                                         config_dict=self._config_dict,
                                         input=input_file,
                                         output=self._sample_test_output,
                                         result=self._sample_test_result,
                                         namespace=self._namespace,
                                         experiment_name=experiment_name,
      )
      pysample_checker.run()
      pysample_checker.check()

    self._copy_result()


class ComponentTest(SampleTest):
  """ Launch a KFP sample test as component test provided its name.

  Currently follows the same logic as sample test for compatibility.
  include xgboost_training_cm
  """
  def __init__(self, test_name, results_gcs_dir,
               dataproc_gcp_image,
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
    self._local_confusionmatrix_image = local_confusionmatrix_image
    self._local_roc_image = local_roc_image
    self._dataproc_gcp_image = dataproc_gcp_image

  def _injection(self):
    """Sample-specific image injection into yaml file."""
    subs = {
        'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-confusion-matrix:\w+':self._local_confusionmatrix_image,
        'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-roc:\w+':self._local_roc_image
    }
    if self._test_name == 'xgboost_training_cm':
      subs.update({
          'gcr\.io/ml-pipeline/ml-pipeline-gcp:\w+':self._dataproc_gcp_image
      })

      utils.file_injection('%s.yaml' % self._test_name,
                           '%s.yaml.tmp' % self._test_name,
                           subs)
    else:
      # Only the above two samples need injection for now.
      pass
    utils.file_injection('%s.yaml' % self._test_name,
                         '%s.yaml.tmp' % self._test_name,
                         subs)


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