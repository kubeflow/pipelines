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
import os
from datetime import datetime
from kfp import Client
import utils

###### Input/Output Instruction ######
# input: yaml
# output: local file path


# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--input',
                      type=str,
                      required=True,
                      help='The path of a pipeline package that will be submitted.')
  parser.add_argument('--result',
                      type=str,
                      required=True,
                      help='The path of the test result that will be exported.')
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='The path of the test output')
  args = parser.parse_args()
  return args

def main():
  args = parse_arguments()
  test_cases = []
  test_name = 'TFX Sample Test'

  ###### Initialization ######
  client = Client()

  ###### Check Input File ######
  utils.add_junit_test(test_cases, 'input generated yaml file', os.path.exists(args.input), 'yaml file is not generated')
  if not os.path.exists(args.input):
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit()

  ###### Create Experiment ######
  experiment_name = 'TFX sample experiment'
  response = client.create_experiment(experiment_name)
  experiment_id = response.id
  utils.add_junit_test(test_cases, 'create experiment', True)

  ###### Create Job ######
  job_name = 'TFX_sample'
  params = {'output': args.output,
            'project': 'ml-pipeline-test',
            'column-names': 'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/column-names.json',
            'evaluation': 'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv',
            'train': 'gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv',
            'hidden-layer-size': '5',
            'steps': '5'}
  response = client.run_pipeline(experiment_id, job_name, args.input, params)
  run_id = response.id
  utils.add_junit_test(test_cases, 'create pipeline run', True)


  ###### Monitor Job ######
  start_time = datetime.now()
  response = client.wait_for_run_completion(run_id, 1200)
  succ = (response.run.status.lower()=='succeeded')
  end_time = datetime.now()
  elapsed_time = (end_time - start_time).seconds
  utils.add_junit_test(test_cases, 'job completion', succ, 'waiting for job completion failure', elapsed_time)
  if not succ:
    utils.write_junit_xml(test_name, args.result, test_cases)
    exit()

  ###### Output Argo Log for Debugging ######
  workflow_json = client._get_workflow_json(run_id)
  workflow_id = workflow_json['metadata']['name']
  #TODO: remove the namespace dependency or make is configurable.
  argo_log, _ = utils.run_bash_command('argo logs -n kubeflow -w {}'.format(workflow_id))
  print("=========Argo Workflow Log=========")
  print(argo_log)

  ###### Validate the results ######
  #TODO: enable after launch
  #   model analysis html is validated
  # argo_workflow_id = workflow_json['metadata']['name']
  # gcs_html_path = os.path.join(os.path.join(args.output, str(argo_workflow_id)), 'analysis/output_display.html')
  # print('Output display HTML path is ' + gcs_html_path)
  # utils.run_bash_command('gsutil cp ' + gcs_html_path + './')
  # display_file = open('./output_display.html', 'r')
  # is_next_line_state = False
  # for line in display_file:
  #   if is_next_line_state:
  #     state = line.strip()
  #     break
  #   if line.strip() == '<script type="application/vnd.jupyter.widget-state+json">':
  #     is_next_line_state = True
  # import json
  # state_json = json.loads(state)
  # succ = ('state' in state_json and 'version_major' in state_json and 'version_minor' in state_json)
  # utils.add_junit_test(test_cases, 'output display html', succ, 'the state json does not contain state, version_major, or version_inor')

  ###### Delete Job ######
  #TODO: add deletion when the backend API offers the interface.

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.result, test_cases)

if __name__ == "__main__":
  main()
