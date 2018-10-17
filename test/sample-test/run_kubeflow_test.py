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
import json
import tarfile

import swagger_job

import api_client_helper
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
  parser.add_argument('--output',
                      type=str,
                      required=True,
                      help='The path of the test result that will be exported.')
  args = parser.parse_args()
  return args

def main():
  args = parse_arguments()
  test_cases = []
  test_name = 'Kubeflow Sample Test'

  ###### Initialization ######
  config = api_client_helper.init_config()
  pipeline_upload_service_api = api_client_helper.init_pipeline_upload_api(config)
  job_service_api = api_client_helper.init_job_api(config)
  run_service_api = api_client_helper.init_run_api(config)

  utils.add_junit_test(test_cases, 'input generated yaml file', os.path.exists(args.input), 'yaml file is not generated')
  if not os.path.exists(args.input):
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()

  ###### Upload Pipeline ######
  pipeline_id, succ = api_client_helper.upload_pipeline(pipeline_upload_service_api, args.input)
  utils.add_junit_test(test_cases, 'upload pipeline yaml', succ, 'yaml file is not uploaded')
  if not succ:
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()

  ###### Create Job ######
  job_name = 'kubeflow_sample'
  params = [swagger_job.ApiParameter(name='output', value='gs://ml-pipeline-dataset/sample-test'),
            swagger_job.ApiParameter(name='project', value='ml-pipeline-test'),
            swagger_job.ApiParameter(name='evaluation', value='gs://ml-pipeline-playground/IntegTest/flower/eval15.csv'),
            swagger_job.ApiParameter(name='train', value='gs://ml-pipeline-playground/IntegTest/flower/train30.csv'),
            swagger_job.ApiParameter(name='hidden-layer-size', value='10,5'),
            swagger_job.ApiParameter(name='steps', value='5')]
  job_id, succ = api_client_helper.create_job(job_service_api, pipeline_id, params, job_name)
  utils.add_junit_test(test_cases, 'create pipeline job', succ, 'job creation failure')
  if not succ:
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()

  ###### Monitor Job ######
  status, elapsed_time = api_client_helper.wait_for_job_completion(job_service_api, job_id, timeout=1200)
  utils.add_junit_test(test_cases, 'job completion', (status.lower() == 'succeeded:'), 'waiting for job completion failure', elapsed_time)
  if status.lower() != 'succeeded:':
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()

  ###### Output Argo Log for Debugging ######
  argo_log, succ = api_client_helper.get_argo_log(job_service_api, run_service_api, job_id)
  if succ:
    print("=========Argo Workflow Log=========")
    print(argo_log)
  else:
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()
  ###### Validate the results ######
  #   confusion matrix should show three columns for the flower data
  #     target, predicted, count
  cm_tar_path = './confusion_matrix.tar.gz'
  cm_filename = 'mlpipeline-ui-metadata.json'
  workflow_json, succ = api_client_helper.get_workflow_json(job_service_api, run_service_api, job_id)
  if not succ:
    utils.write_junit_xml(test_name, args.output, test_cases)
    exit()
  utils.get_artifact_in_minio(workflow_json, 'confusionmatrix', cm_tar_path)
  tar_handler = tarfile.open(cm_tar_path)
  tar_handler.extractall()

  with open(cm_filename, 'r') as f:
    cm_data = json.load(f)
    utils.add_junit_test(test_cases, 'confusion matrix format', (len(cm_data['outputs'][0]['schema']) == 3), 'the column number of the confusion matrix output is not equal to three')

  ###### Delete Job ######
  succ = api_client_helper.delete_job(job_service_api, job_id)
  utils.add_junit_test(test_cases, 'delete pipeline job', succ, 'delete job failure')

  ###### Write out the test result in junit xml ######
  utils.write_junit_xml(test_name, args.output, test_cases)

if __name__ == "__main__":
  main()
