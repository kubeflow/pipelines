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
import sys
import time

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
        print(f"DEBUG: Initial host value: {self._host}")
        if self._host == '':
            print(f"DEBUG: Host is empty, attempting to discover endpoint")
            try:
                # Get inverse proxy hostname from a config map called 'inverse-proxy-config'
                # in the same namespace as KFP.
                try:
                    print(f"DEBUG: Attempting to load in-cluster config")
                    kubernetes.config.load_incluster_config()
                    print(f"DEBUG: Successfully loaded in-cluster config")
                except Exception as k8s_incluster_err:
                    print(f"DEBUG: Failed to load in-cluster config: {k8s_incluster_err}")
                    print(f"DEBUG: Attempting to load kube config")
                    kubernetes.config.load_kube_config()
                    print(f"DEBUG: Successfully loaded kube config")

                print(f"DEBUG: Setting host to http://localhost:8888")
                self._host = 'http://localhost:8888'
            except Exception as err:
                print(f"DEBUG: ERROR: Failed to set up Kubernetes config: {err}")
                raise RuntimeError(
                    'Failed to get inverse proxy hostname') from err
            print(f"DEBUG: Final host value after discovery: {self._host}")

        # With the healthz API in place, when the developer clicks the link,
        # it will lead to a functional URL instead of a 404 error.
        print(f'KFP API healthz endpoint is: {self._host}/apis/v1beta1/healthz')
        
        # Try to ping the healthz endpoint to verify connectivity
        try:
            import requests
            print(f"DEBUG: Attempting to connect to healthz endpoint: {self._host}/apis/v1beta1/healthz")
            start_time = time.time()
            response = requests.get(f"{self._host}/apis/v1beta1/healthz", timeout=10)
            elapsed = time.time() - start_time
            print(f"DEBUG: Healthz endpoint response: status={response.status_code}, time={elapsed:.2f}s")
            if response.status_code != 200:
                print(f"DEBUG: WARNING - Healthz endpoint returned non-200 status: {response.status_code}")
                print(f"DEBUG: Response content: {response.text[:200]}...")
        except Exception as health_err:
            print(f"DEBUG: ERROR connecting to healthz endpoint: {health_err}")
            print("DEBUG: Continuing anyway, but this may cause issues later")

        self._is_notebook = None
        self._work_dir = os.path.join(BASE_DIR, 'samples/core/',
                                      self._test_name)

        self._sample_test_result = 'junit_Sample%sOutput.xml' % self._test_name
        self._sample_test_output = self._results_gcs_dir
        self._expected_result = expected_result

    def _compile(self):
        print(f"--- Entering SampleTest._compile ---")
        print(f"DEBUG: Current directory before changing: {os.getcwd()}")
        print(f"DEBUG: Changing to work directory: {self._work_dir}")
        os.chdir(self._work_dir)
        print(f"DEBUG: Current directory after changing: {os.getcwd()}")
        print('Run the sample tests...')

        # Looking for the entry point of the test.
        list_of_files = os.listdir('.')
        for file in list_of_files:
            # matching by .py or .ipynb, there will be yaml ( compiled ) files in the folder.
            # if you rerun the test suite twice, the test suite will fail
            m = re.match(self._test_name + r'\.(py|ipynb)$', file)
            if m:
                file_name, ext_name = os.path.splitext(file)
                if self._is_notebook is not None:
                    raise (RuntimeError(
                        'Multiple entry points found under sample: {}'.format(
                            self._test_name)))
                if ext_name == '.py':
                    self._is_notebook = False
                if ext_name == '.ipynb':
                    self._is_notebook = True

        if self._is_notebook is None:
            raise (RuntimeError('No entry point found for sample: {}'.format(
                self._test_name)))

        config_schema = yamale.make_schema(SCHEMA_CONFIG)
        # Retrieve default config
        try:
            with open(DEFAULT_CONFIG, 'r') as f:
                raw_args = yaml.safe_load(f)
            default_config = yamale.make_data(DEFAULT_CONFIG)
            yamale.validate(
                config_schema,
                default_config)  # If fails, a ValueError will be raised.
        except yaml.YAMLError as yamlerr:
            raise RuntimeError('Illegal default config:{}'.format(yamlerr))
        except OSError as ose:
            raise FileExistsError('Default config not found:{}'.format(ose))
        else:
            self._run_pipeline = raw_args['run_pipeline']

        # For presubmit check, do not do any image injection as for now.
        # Notebook samples need to be papermilled first.
        if self._is_notebook:
            # Parse necessary params from config.yaml
            nb_params = {}
            try:
                config_file = os.path.join(CONFIG_DIR,
                                           '%s.config.yaml' % self._test_name)
                with open(config_file, 'r') as f:
                    raw_args = yaml.safe_load(f)
                test_config = yamale.make_data(config_file)
                yamale.validate(
                    config_schema,
                    test_config)  # If fails, a ValueError will be raised.
            except yaml.YAMLError as yamlerr:
                print('No legit yaml config file found, use default args:{}'
                      .format(yamlerr))
            except OSError as ose:
                print(
                    'Config file with the same name not found, use default args:{}'
                    .format(ose))
            else:
                if 'notebook_params' in raw_args.keys():
                    nb_params.update(raw_args['notebook_params'])
                    if 'output' in raw_args['notebook_params'].keys(
                    ):  # output is a special param that has to be specified dynamically.
                        nb_params['output'] = self._sample_test_output
                if 'run_pipeline' in raw_args.keys():
                    self._run_pipeline = raw_args['run_pipeline']

            pm.execute_notebook(
                input_path='%s.ipynb' % self._test_name,
                output_path='%s.ipynb' % self._test_name,
                parameters=nb_params,
                prepare_only=True)
            # Convert to python script.
            return_code = subprocess.call([
                'jupyter', 'nbconvert', '--to', 'python',
                '%s.ipynb' % self._test_name
            ])

        else:
            return_code = subprocess.call(['python3', '%s.py' % self._test_name])

        # Command executed successfully!
        assert return_code == 0

    def _injection(self):
        """Inject images for pipeline components.

        This is only valid for coimponent test
        """
        print(f"--- Entering SampleTest._injection ---")
        pass

    def run_test(self):
        print(f"--- Entering SampleTest.run_test ---")
        try:
            self._compile()
            print(f"DEBUG: _compile completed successfully")
            self._injection()
            print(f"DEBUG: _injection completed successfully")

            # Overriding the experiment name of pipeline runs
            experiment_name = self._test_name + '-test'
            print(f"DEBUG: Setting experiment name to: {experiment_name}")
            os.environ['KF_PIPELINES_OVERRIDE_EXPERIMENT_NAME'] = experiment_name
        except Exception as e:
            print(f"DEBUG: ERROR in run_test: {e}")
            import traceback
            traceback.print_exc()
            raise

        if self._is_notebook:
            nbchecker = NoteBookChecker(
                testname=self._test_name,
                result=self._sample_test_result,
                run_pipeline=self._run_pipeline,
                experiment_name=experiment_name,
                host=self._host,
            )
            nbchecker.run()
            os.chdir(TEST_DIR)
            nbchecker.check()
        else:
            os.chdir(TEST_DIR)
            input_file = os.path.join(self._work_dir,
                                      '%s.py.yaml' % self._test_name)

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
            pysample_checker.run()
            pysample_checker.check()


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
        utils.file_injection('%s.py.yaml' % self._test_name,
                             '%s.py.yaml.tmp' % self._test_name, subs)


def main():
    """Launches either KFP sample test or component test as a command
    entrypoint.

    Usage:
    python sample_test_launcher.py sample_test run_test arg1 arg2 to launch sample test, and
    python sample_test_launcher.py component_test run_test arg1 arg2 to launch component
    test.
    """
    print(f"--- Entering main ---")
    print(f"DEBUG: Python version: {sys.version}")
    print(f"DEBUG: Command line arguments: {sys.argv}")
    try:
        fire.Fire({'sample_test': SampleTest, 'component_test': ComponentTest})
    except Exception as e:
        print(f"DEBUG: ERROR in main(): {e}")
        import traceback
        traceback.print_exc()
        raise


if __name__ == '__main__':
    main()
