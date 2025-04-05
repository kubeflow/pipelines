# Copyright 2019 The Kubeflow Authors
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
"""This launcher module serves as the entry-point of the sample test image.

It decides which test to trigger based upon the arguments provided.
"""

import os
import re
import subprocess

from check_notebook_results import NoteBookChecker
from constants import BASE_DIR
from constants import CONFIG_DIR
from constants import DEFAULT_CONFIG
from constants import SCHEMA_CONFIG
from constants import TEST_DIR
import fire
import kubernetes
import papermill as pm
from run_sample_test import PySampleChecker
import utils
import yamale
import yaml


class SampleTest(object):

    def __init__(self,
                 test_name,
                 results_gcs_dir,
                 host='',
                 target_image_prefix='',
                 namespace='kubeflow',
                 expected_result='succeeded'):
        """Launch a KFP sample_test provided its name.

        :param test_name: name of the corresponding sample test.
        :param results_gcs_dir: gs dir to store test result.
        :param host: host of KFP API endpoint, default is auto-discovery from inverse-proxy-config.
        :param target_image_prefix: prefix of docker image, default is empty.
        :param namespace: namespace for kfp, default is kubeflow.
        :param expected_result: the expected status for the run, default is succeeded.
        """
        print(f"--- Entering SampleTest.__init__ ---")
        self._test_name = test_name
        self._results_gcs_dir = results_gcs_dir
        # Capture the first segment after gs:// as the project name.
        self._target_image_prefix = target_image_prefix
        self._namespace = namespace
        self._host = host
        if self._host == '':
            try:
                # Get inverse proxy hostname from a config map called 'inverse-proxy-config'
                # in the same namespace as KFP.
                try:
                    kubernetes.config.load_incluster_config()
                except:
                    kubernetes.config.load_kube_config()

                self._host = 'http://localhost:8888'
            except Exception as err:
                raise RuntimeError(
                    'Failed to get inverse proxy hostname') from err

        # With the healthz API in place, when the developer clicks the link,
        # it will lead to a functional URL instead of a 404 error.
        print(f'KFP API healthz endpoint is: {self._host}/apis/v1beta1/healthz')

        self._is_notebook = None
        self._work_dir = os.path.join(BASE_DIR, 'samples/core/',
                                      self._test_name)

        self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
        self._sample_test_output = self._results_gcs_dir
        self._expected_result = expected_result

    def _compile(self):
        print(f"--- Entering SampleTest._compile ---")
        os.chdir(self._work_dir)
        print(f'Changed directory to: {self._work_dir}')

        # Looking for the entry point of the test.
        list_of_files = os.listdir('.')
        print(f'Files in work dir: {list_of_files}')
        entry_point = None
        for file in list_of_files:
            # matching by .py or .ipynb, there will be yaml ( compiled ) files in the folder.
            # if you rerun the test suite twice, the test suite will fail
            m = re.match(self._test_name + r'\.(py|ipynb)$', file)
            if m:
                file_name, ext_name = os.path.splitext(file)
                if self._is_notebook is not None:
                    raise (RuntimeError(
                        'There are both .py and .ipynb files for test: %s' %
                        self._test_name))
                entry_point = file
                if ext_name == '.ipynb':
                    self._is_notebook = True
                    print(f'Detected notebook entry point: {entry_point}')
                else:
                    self._is_notebook = False
                    print(f'Detected python script entry point: {entry_point}')

        if entry_point is None:
            raise (RuntimeError('No .py or .ipynb file found for test %s' %
                              self._test_name))

        if self._is_notebook:
            print(f'Executing notebook: {entry_point} using papermill...')
            try:
                pm.execute_notebook(
                    entry_point,
                    'out.ipynb',
                    parameters=dict(
                        output=self._sample_test_output,
                        project='YOUR_PROJECT_ID',
                        test_data_dir='gs://ml-pipeline-playground/testdata'
                    )
                )
                print(f'Papermill execution finished for {entry_point}')
            except Exception as e:
                print(f'ERROR during papermill execution: {e}')
                raise
            # Prepare KFP package
            print(f'Preparing KFP package for notebook...')
            self._run_pipeline = os.path.join(self._work_dir, 'out.ipynb')
            # Notebooks must be compiled to pipeline package yaml.
            subprocess.check_call([
                'jupyter', 'nbconvert', '--to', 'python',
                os.path.join(self._work_dir, 'out.ipynb')
            ])
            subprocess.check_call([
                'dsl-compile-ipynb', '--py',
                os.path.join(self._work_dir, 'out.py'), '--output',
                os.path.join(self._work_dir, 'out.yaml')
            ])
            self._run_pipeline = os.path.join(self._work_dir, 'out.yaml')
            print(f'KFP package for notebook created: {self._run_pipeline}')
        else:
            # Compile the pipeline
            pipeline_func = utils.load_module(entry_point)
            output_package = os.path.join(self._work_dir,
                                        '%s.py.yaml' % self._test_name)
            print(f'Compiling python script {entry_point} to {output_package}...')
            try:
                kfp.compiler.Compiler().compile(pipeline_func, output_package)
                print(f'Compilation finished for {entry_point}.')
            except Exception as e:
                print(f'ERROR during KFP compilation: {e}')
                raise

            # TODO: temporary workaround for running sample tests locally
            # The sample test workflow compiles the python DSL sample code into KFP yaml
            # static definition, then uses KFP client CLI to submit the pipeline run.
            # Running KFP sample tests need the artifacts to be passed correctly by
            # replacing the input placeholders. This is not working currently.
            # Also, this is tied to specifics of Argo CLI and needs more work.
            print('Executing python script locally (temporary workaround? Check comments)...')
            return_code = subprocess.call(['python3', entry_point])
            print(f'Local python script execution finished with code: {return_code}')

        # Command executed successfully!
        assert return_code == 0 # This assert seems problematic if compilation path taken?

    def _injection(self):
        """Inject images for pipeline components.

        This is only valid for coimponent test
        """
        print(f"--- Entering SampleTest._injection ---")
        pass

    def run_test(self):
        print(f"--- Entering SampleTest.run_test ---")
        print(f'Starting test: {self._test_name}, notebook: {self._is_notebook}') # Log before compile
        self._compile()
        print(f'Compilation/preparation finished for {self._test_name}. Notebook: {self._is_notebook}')
        self._injection()

        # Overriding the experiment name of pipeline runs
        experiment_name = self._test_name + '-test'
        print(f'Setting experiment name override to: {experiment_name}')
        os.environ['KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'] = experiment_name

        if self._is_notebook:
            print(f'Initializing NoteBookChecker for {self._test_name}')
            nbchecker = NoteBookChecker(
                testname=self._test_name,
                result=self._sample_test_result,
                run_pipeline=self._run_pipeline,
                experiment_name=experiment_name,
                host=self._host,
            )
            print(f'Calling NoteBookChecker.run() for {self._test_name}')
            nbchecker.run()
            print(f'NoteBookChecker.run() finished for {self._test_name}')
            os.chdir(TEST_DIR)
            print(f'Calling NoteBookChecker.check() for {self._test_name}')
            nbchecker.check()
            print(f'NoteBookChecker.check() finished for {self._test_name}')
        else:
            print(f'Initializing PySampleChecker for {self._test_name}')
            os.chdir(TEST_DIR)
            input_file = os.path.join(self._work_dir,
                                      '%s.py.yaml' % self._test_name)
            print(f'Using input file for PySampleChecker: {input_file}')

            pysample_checker = PySampleChecker(
                testname=self._test_name,
                input=input_file,
                output=self._sample_test_output,
                result=self._sample_test_result,
                host=self._host,
                namespace=self._namespace,
                experiment_name=experiment_name,
                expected_result=self._expected_result,
            )
            print(f'Calling PySampleChecker.run() for {self._test_name}')
            pysample_checker.run()
            print(f'PySampleChecker.run() finished for {self._test_name}')
            print(f'Calling PySampleChecker.check() for {self._test_name}')
            pysample_checker.check()
            print(f'PySampleChecker.check() finished for {self._test_name}')


class ComponentTest(SampleTest):
    """Launch a KFP sample test as component test provided its name.

    Currently follows the same logic as sample test for compatibility.
    include xgboost_training_cm
    """

    def __init__(self,
                 test_name,
                 results_gcs_dir,
                 gcp_image,
                 local_confusionmatrix_image,
                 local_roc_image,
                 target_image_prefix='',
                 namespace='kubeflow'):
        print(f"--- Entering ComponentTest.__init__ ---")
        super().__init__(
            test_name=test_name,
            results_gcs_dir=results_gcs_dir,
            target_image_prefix=target_image_prefix,
            namespace=namespace)
        self._local_confusionmatrix_image = local_confusionmatrix_image
        self._local_roc_image = local_roc_image
        self._dataproc_gcp_image = gcp_image

    def _injection(self):
        """Sample-specific image injection into yaml file."""
        print(f"--- Entering ComponentTest._injection ---")
        print(f'Injecting images for component test: {self._test_name}')
        subs = {  # Tag can look like 1.0.0-rc.3, so we need both "-" and "." in the regex.
            r'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-confusion-matrix:(\w+|[.-])+':
                self._local_confusionmatrix_image,
            r'gcr\.io/ml-pipeline/ml-pipeline/ml-pipeline-local-roc:(\w+|[.-])+':
                self._local_roc_image
        }
        if self._test_name == 'xgboost_training_cm':
            subs.update({
                r'gcr\.io/ml-pipeline/ml-pipeline-gcp:(\w|[.-])+':
                    self._dataproc_gcp_image
            })

            utils.file_injection('%s.py.yaml' % self._test_name,
                                 '%s.py.yaml.tmp' % self._test_name, subs)
        else:
            # Only the above sample need injection for now.
            pass
        print(f'Performing file injection into {self._test_name}.py.yaml')
        utils.file_injection('%s.py.yaml' % self._test_name,
                             '%s.py.yaml.tmp' % self._test_name, subs)
        print(f'File injection finished for {self._test_name}.py.yaml')


def main():
    """Launches either KFP sample test or component test as a command
    entrypoint.

    Usage:
    python sample_test_launcher.py sample_test run_test arg1 arg2 to launch sample test, and
    python sample_test_launcher.py component_test run_test arg1 arg2 to launch component
    test.
    """
    print(f"--- Entering main ---")
    fire.Fire({'sample_test': SampleTest, 'component_test': ComponentTest})


if __name__ == '__main__':
    main()
