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

import argparse
from kfp import Client
import utils

###### Input/Output Instruction ######
# input: experiment name, testname, and, namespace

# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--experiment',
                      type=str,
                      required=True,
                      help='The experiment name')
  parser.add_argument('--testname',
                      type=str,
                      required=True,
                      help="Test name")
  parser.add_argument('--namespace',
                      type=str,
                      default='kubeflow',
                      help="namespace of the deployed pipeline system. Default: kubeflow")
  parser.add_argument('--result',
                      type=str,
                      required=True,
                      help='The path of the test result that will be exported.')
  args = parser.parse_args()
  return args

def main():
  args = parse_arguments()
  test_cases = []
  test_name = args.testname + ' Sample Test'

  ###### Initialization ######
  client = Client(namespace=args.namespace)

  ###### Get experiments ######
  list_experiments_response = client.list_experiments(page_size=100)
  for experiment in list_experiments_response.experiments:
    if experiment.name == args.experiment:
      experiment_id = experiment.id

  ###### Get runs ######
  import kfp_run
  resource_reference_key_type =kfp_run.models.api_resource_type.ApiResourceType.EXPERIMENT
  resource_reference_key_id = experiment_id
  list_runs_response = client.list_runs(page_size=1000, resource_reference_key_type=resource_reference_key_type, resource_reference_key_id=resource_reference_key_id)

  ###### Check all runs ######
  for run in list_runs_response.runs:
    run_id = run.id
    response = client.wait_for_run_completion(run_id, 1200)
    succ = (response.run.status.lower()=='succeeded')
    utils.add_junit_test(test_cases, 'job completion', succ, 'waiting for job completion failure')

    ###### Output Argo Log for Debugging ######
    workflow_json = client._get_workflow_json(run_id)
    workflow_id = workflow_json['metadata']['name']
    argo_log, _ = utils.run_bash_command('argo logs -n {} -w {}'.format(args.namespace, workflow_id))
    print("=========Argo Workflow Log=========")
    print(argo_log)

    if not succ:
      utils.write_junit_xml(test_name, args.result, test_cases)
      exit(1)

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)

if __name__ == "__main__":
  main()
