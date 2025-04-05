# Copyright 2019-2021 The Kubeflow Authors
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

from datetime import datetime
import os
import tarfile
import time

from constants import CONFIG_DIR
from constants import DEFAULT_CONFIG
from constants import SCHEMA_CONFIG
import kfp
from kfp import Client
import utils
import yamale
import yaml


class PySampleChecker(object):

    def __init__(self,
                 testname,
                 input,
                 output,
                 result,
                 experiment_name,
                 host,
                 namespace='kubeflow',
                 expected_result='succeeded'):
        """Util class for checking python sample test running results.

        :param testname: test name.
        :param input: The path of a pipeline file that will be submitted.
        :param output: The path of the test output.
        :param result: The path of the test result that will be exported.
        :param host: The hostname of KFP API endpoint.
        :param namespace: namespace of the deployed pipeline system. Default: kubeflow
        :param expected_result: the expected status for the run, default is succeeded.
        :param experiment_name: Name of the experiment to monitor
        """
        self._testname = testname
        self._experiment_name = experiment_name
        self._input = input
        self._output = output
        self._result = result
        self._host = host
        self._namespace = namespace
        self._run_pipeline = None
        self._test_timeout = None

        self._test_cases = []
        self._test_name = self._testname + ' Sample Test'

        self._client = None
        self._experiment_id = None
        self._job_name = None
        self._test_args = None
        self._run_id = None
        self._expected_result = expected_result

    def run(self):
        """Run compiled KFP pipeline."""
        print("--- Entering PySampleChecker.run ---")

        ###### Initialization ######
        print(f"Initializing KFP client with host: {self._host}")
        self._client = Client(host=self._host)

        ###### Check Input File ######
        print(f"Checking for input file: {self._input}")
        utils.add_junit_test(self._test_cases, 'input generated yaml file',
                             os.path.exists(self._input),
                             'yaml file is not generated')
        if not os.path.exists(self._input):
            utils.write_junit_xml(self._test_name, self._result,
                                  self._test_cases)
            print('Error: job not found.')
            exit(1)

        ###### Create Experiment ######
        response = self._client.create_experiment(self._experiment_name)
        self._experiment_id = response.experiment_id
        utils.add_junit_test(self._test_cases, 'create experiment', True)

        ###### Create Job ######
        self._job_name = self._testname + '_sample'
        ###### Figure out arguments from associated config files. #######
        self._test_args = {}
        config_schema = yamale.make_schema(SCHEMA_CONFIG)
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
            self._test_timeout = raw_args['test_timeout']
            self._run_pipeline = raw_args['run_pipeline']

        try:
            config_file = os.path.join(CONFIG_DIR,
                                       '%s.config.yaml' % self._testname)
            with open(config_file, 'r') as f:
                raw_args = yaml.safe_load(f)
            test_config = yamale.make_data(config_file)
            yamale.validate(
                config_schema,
                test_config)  # If fails, a ValueError will be raised.
        except yaml.YAMLError as yamlerr:
            print('No legit yaml config file found, use default args:{}'.format(
                yamlerr))
        except OSError as ose:
            print(
                f'Config file "{config_file}" not found, using default args: {raw_args}')
        else:
            if 'arguments' in raw_args.keys() and raw_args['arguments']:
                self._test_args.update(raw_args['arguments'])
            if 'output' in self._test_args.keys(
            ):  # output is a special param that has to be specified dynamically.
                self._test_args['output'] = self._output
            if 'test_timeout' in raw_args.keys():
                self._test_timeout = raw_args['test_timeout']
            if 'run_pipeline' in raw_args.keys():
                self._run_pipeline = raw_args['run_pipeline']

        # TODO(numerology): Special treatment for TFX::OSS sample
        if self._testname == 'parameterized_tfx_oss':
            self._test_args['pipeline-root'] = os.path.join(
                self._test_args['output'],
                'tfx_taxi_simple_' + kfp.dsl.RUN_ID_PLACEHOLDER)
            del self._test_args['output']

        # Submit for pipeline running.
        if self._run_pipeline:
            print(f"Attempting to run pipeline: experiment_id={self._experiment_id}, job_name={self._job_name}, input={self._input}, args={self._test_args}")
            try:
                response = self._client.run_pipeline(self._experiment_id,
                                                 self._job_name, self._input,
                                                 self._test_args)
                self._run_id = response.run_id
                print(f"Successfully submitted pipeline run. Run ID: {self._run_id}")
                utils.add_junit_test(self._test_cases, 'create pipeline run', True)
            except Exception as e:
                print(f"ERROR during client.run_pipeline: {e}")
                utils.add_junit_test(self._test_cases, 'create pipeline run', False, str(e))
                # Re-raise the exception to potentially halt the process if needed
                raise
        else:
            print("Skipping pipeline run submission because self._run_pipeline is False.")

    def check(self):
        """Check pipeline run results."""
        if self._run_pipeline:
            ###### Monitor Job ######
            start_time = datetime.now()
            run_status = None
            error_message = None
            print(f'Waiting for run {self._run_id} to complete (timeout: {self._test_timeout}s)...')
            try:
                response = self._client.wait_for_run_completion(self._run_id, self._test_timeout)
                run_status = response.state.lower()
                print(f'Run {self._run_id} completed with status: {run_status}')
            except Exception as e:
                print(f'Error or timeout waiting for run {self._run_id}: {e}')
                error_message = str(e)
                # Attempt to get final status even after timeout/error
                try:
                    run_detail = self._client.get_run(self._run_id)
                    run_status = run_detail.run.state.lower()
                    print(f'Final run status fetched: {run_status}')
                    if run_detail.run.error:
                        error_message = run_detail.run.error
                        print(f'Run error message: {error_message}')
                except Exception as get_run_e:
                    print(f'Could not fetch final run status after error: {get_run_e}')

            # Check if the final status matches the expected result
            succ = (run_status == self._expected_result)

            end_time = datetime.now()
            elapsed_time = (end_time - start_time).seconds
            # Include status and error in junit failure message
            failure_message = f'waiting for job completion failure. Final Status: {run_status}. Error: {error_message}' if not succ else None
            utils.add_junit_test(self._test_cases, 'job completion', succ,
                                 failure_message,
                                 elapsed_time)
            print(f'Pipeline {"worked" if succ else "failed"}. Elapsed time: {elapsed_time}s')

            ###### Delete Job ######
            #TODO: add deletion when the backend API offers the interface.

            # Use the logged status/error for assertion message
            assert succ, f'Pipeline run {self._run_id} failed or did not reach expected state {self._expected_result}. Final Status: {run_status}. Error: {error_message}'
