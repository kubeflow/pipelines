# Copyright 2020 Google LLC
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

import argparse
import json
import os
import utils

import kfp
import kfp_server_api
from kfp.components import func_to_container_op
from kfp.containers._gcs_helper import GCSHelper

class MultiuserTest(object):

  def __init__(self, test_name, results_gcs_dir, host, client_id, other_client_id, other_client_secret, refresh_token, user_namespace, other_namespace):
    """Util class for checking python sample test running results.

    :param test_name: The name of the test.
    :param results_gcs_dir: The gs dir to store test results.
    :param host: The host name to use to talk to Kubeflow Pipelines.
    :param client_id: The client ID used by Identity-Aware Proxy.
    :param other_client_id: The client ID used to obtain the auth codes and refresh tokens.
    :param other_client_secret: The client secret used to obtain the auth codes and refresh tokens.
    :param refresh_token: The pre-acquired refresh token for the test account.
    :param user_namespace: The user namespace owned by the test account.
    :param other_namespace: The namespace not owned by the test account.
    """

    # Save credentails in local file before initialing KFP client to bypass the login process
    credentials = {}
    credentials[client_id] = {}
    credentials[client_id]['other_client_id'] = other_client_id
    credentials[client_id]['other_client_secret'] = other_client_secret
    credentials[client_id]['refresh_token'] = refresh_token

    LOCAL_KFP_CREDENTIAL = os.path.expanduser('~/.config/kfp/credentials.json')
    if not os.path.exists(os.path.dirname(LOCAL_KFP_CREDENTIAL)):
      os.makedirs(os.path.dirname(LOCAL_KFP_CREDENTIAL))
    with open(LOCAL_KFP_CREDENTIAL, 'w') as f:
      json.dump(credentials, f)

    self._test_name = test_name
    self._results_gcs_dir = results_gcs_dir
    self._client = kfp.Client(host=host, client_id=client_id, other_client_id=other_client_id, other_client_secret=other_client_secret)
    self._user_namespace = user_namespace
    self._other_namespace = other_namespace
    self._test_cases = []
    self._test_result = 'junit_{}_Output.xml'.format(self._test_name)

    print('Initilization completed.')

  def _copy_result(self):
    """ Copy generated test result to gcs, so that Prow can pick it. """
    print('Copy the test results to GCS %s/' % self._results_gcs_dir)
    GCSHelper.upload_gcs_file(
        self._test_result,
        os.path.join(self._results_gcs_dir, self._test_result))

  def run_test(self):
    """ Run multiuser test via SDK. """
    try:
      print('Start Multiuser test.')
      
      self._client.set_user_namespace(self._user_namespace)
      print('Set user namespace {} in context config'.format(self._client.get_user_namespace()))

      ###### Create Experiment ######
      print('Create experiment')
      experiment_name = 'experiment-{}'.format(self._user_namespace)
      response = self._client.create_experiment(name=experiment_name)
      experiment_id = response.id
      succ = (response.id != '')
      utils.add_junit_test(self._test_cases, 'create experiment', succ)
      if not succ:
        print('Error in creating experiment. Response: {}'.format(response))
        exit(1)

      ###### List Experiments ######
      print('List experiments')
      response = self._client.list_experiments()
      print(response)
      succ = (response.total_size == 1 and response.experiments[0].id == experiment_id)
      utils.add_junit_test(self._test_cases, 'list experiments', succ)
      if not succ:
        print('Error in listing experiments. Response: {}'.format(response))
        exit(1)

      ###### Create Pipeline Run ######
      print('Create pipeline run')
      @func_to_container_op
      def print_small_text(text: str):
        print(text)

      def constant_to_consumer_pipeline():
        consume_task = print_small_text('Hello world')

      response = self._client.create_run_from_pipeline_func(constant_to_consumer_pipeline, arguments={}, experiment_name=experiment_name)
      print(response)
      run_id = response.run_id
      succ = (response.run_id != '')
      utils.add_junit_test(self._test_cases, 'create pipeline run', succ)
      if not succ:
        print('Error in creating pipeline run. Response: {}'.format(response))
        exit(1)

      ###### Get Run Status ######
      print('Get run statu')
      response = self._client.get_run(run_id)
      print(response)
      response = self._client.wait_for_run_completion(run_id, 300)
      succ = (response.run.status.lower() == 'succeeded')
      utils.add_junit_test(self._test_cases, 'get run status', succ)
      if not succ:
        print('Error in getting run status. Response: {}'.format(response))
        exit(1)

      ###### Negative Test - Unauthorized access  ######
      print('List experiments in unauthorized namespace')
      if self._other_namespace:
        succ = False
        try:
          self._client.list_experiments(namespace=self._other_namespace)
        except kfp_server_api.rest.ApiException as e:
          succ = True
          print(e)

        utils.add_junit_test(self._test_cases, 'list experiments - unauthorized access', succ)
        if not succ:
          print('Error in listing experiment - negative case.')
          exit(1)

      print('Multiuser test succeeded.')

    finally:
      utils.write_junit_xml(self._test_name, self._test_result, self._test_cases)
      self._copy_result()


# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""
  parser = argparse.ArgumentParser()
  parser.add_argument('--test-name',
                      type=str,
                      required=True,
                      help='The name of the test.')
  parser.add_argument('--results-gcs-dir',
                      type=str,
                      required=True,
                      help='The gs dir to store test results.')
  parser.add_argument('--host',
                      type=str,
                      required=True,
                      help='The host name to use to talk to Kubeflow Pipelines.')
  parser.add_argument('--client-id',
                      type=str,
                      required=True,
                      help='The client ID used by Identity-Aware Proxy.')
  parser.add_argument('--other-client-id',
                      type=str,
                      required=True,
                      help='The client ID used to obtain the auth codes and refresh tokens.')
  parser.add_argument('--other-client-secret',
                      type=str,
                      required=True,
                      help='The client ID used to obtain the auth codes and refresh tokens.')
  parser.add_argument('--refresh-token',
                      type=str,
                      required=True,
                      help='The pre-acquired refresh token for the test account.')
  parser.add_argument('--user-namespace',
                      type=str,
                      required=True,
                      help='The user namespace owned by the test account.')
  parser.add_argument('--other-namespace',
                      type=str,
                      required=False,
                      help='The user namespace not owned by the test account.')
  args = parser.parse_args()
  return args


def main():
  args = parse_arguments()
  multiuser_test = MultiuserTest(
    test_name=args.test_name,
    results_gcs_dir=args.results_gcs_dir,
    host=args.host,
    client_id=args.client_id,
    other_client_id=args.other_client_id,
    other_client_secret=args.other_client_secret,
    refresh_token=args.refresh_token,
    user_namespace=args.user_namespace,
    other_namespace=args.other_namespace)
  multiuser_test.run_test()

if __name__ == '__main__':
  main()
