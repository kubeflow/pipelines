# Copyright 2018 Google LLC
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

import subprocess
import utils
import yaml

from constants import RUN_LIST_PAGE_SIZE, DEFAULT_CONFIG
from kfp import Client


class NoteBookChecker(object):
    def __init__(self, testname, result, run_pipeline, experiment_name, host, namespace='kubeflow'):
        """ Util class for checking notebook sample test running results.

        :param testname: test name in the json xml.
        :param result: name of the file that stores the test result
        :param run_pipeline: whether to submit for a pipeline run.
        :param host: The hostname of KFP API endpoint.
        :param namespace: where the pipeline system is deployed.
        :param experiment_name: Name of the experiment to monitor
        """
        self._testname = testname
        self._result = result
        self._exit_code = None
        self._run_pipeline = run_pipeline
        self._host = host
        self._namespace = namespace
        self._experiment_name = experiment_name

    def run(self):
        """ Run the notebook sample as a python script. """
        self._exit_code = str(
            subprocess.call(['ipython', '%s.py' % self._testname]))

    def check(self):
        """ Check the pipeline running results of the notebook sample. """
        test_cases = []
        test_name = self._testname + ' Sample Test'

        ###### Write the script exit code log ######
        utils.add_junit_test(test_cases, 'test script execution',
                             (self._exit_code == '0'),
                             'test script failure with exit code: '
                             + self._exit_code)

        try:
            with open(DEFAULT_CONFIG, 'r') as f:
                raw_args = yaml.safe_load(f)
        except yaml.YAMLError as yamlerr:
            raise RuntimeError('Illegal default config:{}'.format(yamlerr))
        except OSError as ose:
            raise FileExistsError('Default config not found:{}'.format(ose))
        else:
            test_timeout = raw_args['test_timeout']

        if self._run_pipeline:
            experiment = self._experiment_name
            ###### Initialization ######
            client = Client(host=self._host)

            ###### Get experiments ######
            experiment_id = client.get_experiment(experiment_name=experiment).id

            ###### Get runs ######
            list_runs_response = client.list_runs(page_size=RUN_LIST_PAGE_SIZE,
                                                  experiment_id=experiment_id)

            ###### Check all runs ######
            for run in list_runs_response.runs:
                run_id = run.id
                response = client.wait_for_run_completion(run_id, test_timeout)
                succ = (response.run.status.lower()=='succeeded')
                utils.add_junit_test(test_cases, 'job completion',
                                     succ, 'waiting for job completion failure')

                ###### Output Argo Log for Debugging ######
                workflow_json = client._get_workflow_json(run_id)
                workflow_id = workflow_json['metadata']['name']
                argo_log, _ = utils.run_bash_command(
                    'argo logs -n {} -w {}'.format(self._namespace, workflow_id))
                print("=========Argo Workflow Log=========")
                print(argo_log)

                if not succ:
                    utils.write_junit_xml(test_name, self._result, test_cases)
                    exit(1)

        ###### Write out the test result in junit xml ######
        utils.write_junit_xml(test_name, self._result, test_cases)